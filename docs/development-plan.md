# Development Plan

Working plan for the hospital middleware backend. Companion to
[api-spec.md](./api-spec.md), which documents the contract as it stands today, and
[er-diagram.md](./er-diagram.md), which documents the schema.

- **Repository**: `agnos-backend-assignment`
- **Language / stack**: Go 1.25, Gin, sqlx, pgx, PostgreSQL 17, golang-migrate, viper,
  testify; nginx in front under compose
- **Last reviewed**: 2026-08-28

---

## 1. Purpose

This document records what is built, what is broken, and the order in which the
remaining work should be picked up. It is meant to be edited as work lands — the
status column is the source of truth for progress.

---

## 2. Current status

### Architecture

Four layers, each depending only on the one below it:

```
src/api           route registration, middleware wiring
src/app/domain    HTTP handlers, request/response models, validation      (*app)
src/business      domain logic, orchestration, sentinel errors            (*bus)
  └─ stores       SQL, db models, filters                                 (*db)
src/business/sdk  page, order, sqldb, migrate
src/foundation    logger, env
```

Deployed shape under `docker compose`:

```
client → nginx :8000 → api :8000 (gin) → postgres :5432
         rate limit,    handlers,          6 tables,
         X-Request-ID,  transactions       one migration
         error envelope
```

Rules the code follows today, worth keeping:

- The app layer never touches SQL; the store layer never formats HTTP.
- Business packages own their sentinel errors (`ErrNotFound`, `ErrDuplicate`, …);
  the app layer maps those to status codes and nothing else.
- Stores translate postgres constraint codes into `sqldb.Err*` values, so business
  code never sees a driver type.
- `Store` and `Business` are interfaces, so every layer is testable with a mock.
- Optional text columns are pointers in db models, keeping `NULL` distinct from `""`
  and keeping partial unique indexes honest.
- A multi-write handler takes all its businesses from one transaction
  (`staffapp.buses`), so a half-finished registration is never committed.

### Domains

| Domain | Business | Store | Store mock | App handlers | HTTP routes | Tests |
| --- | --- | --- | --- | --- | --- | --- |
| `user` | done | done (2 SQL bugs) | done | done | 4 | bus + app |
| `staff` | done | done | done | done | 2 | bus + app |
| `patient` | done | done | done | done | 2 | app |
| `hospital` | done | done | done | done | 4 | app |
| `hospital_patient` | done | done | done | none | 0 | bus |
| `hospital_staff` | done | done | done | none | 0 | bus |

Twelve routes are registered in total; the table in
[api-spec.md](./api-spec.md#route-table) lists them. Both junction domains are reached
only from inside the staff and patient handlers. Handlers that exist but are not routed:
`patientapp.Create` / `Delete`, `staffapp.GetByID` / `Query` / `Delete`. `Update` is
implemented in every store and routed nowhere.

### Test coverage

8 of 26 packages carry tests: the four routed `*app` packages and `userbus`,
`staffbus`, `hospitalstaffbus`, `hospitalpatientbus`. 40 test functions expand to
roughly 210 sub-tests. They are unit tests only: handlers run against the real business
layer with a mocked store, over `httptest`.

`make test` is **red** — one sub-test fails, see [defect 3](#3-known-defects). Because
the Dockerfile runs `go test -race ./...` in its builder stage, the same failure also
fails `docker compose build`.

Not covered:

- **Every line of SQL.** No test executes a query against postgres, which is exactly
  why the two `userdb` bugs survived.
- **The production route table.** Each app suite builds its own router with its own
  paths (`patientapp_test.go` registers `/patient` and `/patient/search/:id`, while
  production registers `/patient/search` and `/patient/:id`), so route registration,
  the middleware chain and `api.Handler`'s wiring are untested — which is how
  [defect 4](#3-known-defects) got in.
- **Startup.** Nothing exercises `main`/`env.Load`/`mid.Cors` configuration paths.

### Infrastructure

- **Config**: `config.yaml` read through viper, plus `AutomaticEnv` with `.` → `_`, so
  `db.user` also comes from `DB_USER`. `main` calls `env.Load(".", "config.yml")` while
  the file on disk is `config.yaml`; that works only because viper probes every
  supported extension for the base name `config`.
- **Config in containers**: the runtime image holds the binary and nothing else, so
  `ReadInConfig` fails there and every value arrives through the environment. The
  discarded `env.Load` error is therefore load-bearing — see
  [defect 11](#3-known-defects) before "fixing" it.
- **Migrations**: `src/business/sdk/migrate/sql/000001_init-tables.{up,down}.sql`, run
  by the `make migrate-*` targets, or by the binary itself with `-migration`
  (compose starts the API as `./backend --migration=true`).
- **Container**: multi-stage `Dockerfile` — `golang:1.26-alpine` builder that runs
  `go test -race ./...` before `go build`, then a bare `alpine` with
  `TZ=Asia/Bangkok`. No `ENTRYPOINT`/`CMD`; the command comes from compose. Runs as
  root.
- **Compose**: `postgres:17-alpine` (healthcheck-gated, `pgdata` volume) → `api`
  (expose only) → `nginx:1.27-alpine`, the only published port. All three on one
  bridge network, all `restart: unless-stopped`.
- **Edge**: `nginx.conf` terminates the client connection, rate-limits reads and writes
  separately per IP, mints/propagates `X-Request-ID`, answers `/healthz` itself, and
  keeps its own 404/429/503 bodies in the API's response envelope.
- **Still missing**: CI, TLS, a non-root container user, request-id in the app's log
  lines, and any integration test.

### Closed since the previous review

- **Passwords are hashed.** `userbus.Create` stores a bcrypt hash at default cost;
  `userbus.Authenticate` compares against it and burns the same bcrypt cost for an
  unknown username, so timing does not reveal which usernames exist.
- **Staff login exists.** `POST /staff/login` authenticates against one hospital
  (credentials + an open `hospital_staffs` assignment). It still issues no token.
- **Container and edge exist.** Dockerfile, docker-compose and nginx replaced the
  "no Dockerfile, no docker-compose" state of the last review.

---

## 3. Known defects

Ordered by severity. Items 1–3 mean the repo cannot be built into a working
deployment: two of them break every write to `users`, the third breaks the image build.

| # | Severity | Defect | Location |
| --- | --- | --- | --- |
| 1 | Blocker | `INSERT` lists 4 columns against 5 placeholders (`$1..$5`) — every create fails | [userdb.go:47](../src/business/domain/userbus/stores/userdb/userdb.go#L47) |
| 2 | Blocker | Same insert omits `id`, which has no `DEFAULT nextval(...)` → not-null violation. The other five stores work around this with an explicit `nextval()`; the migration should set the default instead | [000001_init-tables.up.sql](../src/business/sdk/migrate/sql/000001_init-tables.up.sql) |
| 3 | Blocker | `make test` and `docker compose build` fail: a stale sub-test drives the search with `?phone=` while the handler binds `form:"phone_number"`, so the store mock panics on an unexpected call — which also aborts the rest of that package | [patientapp_test.go](../src/app/domain/patientapp/patientapp_test.go) vs [filter.go](../src/app/domain/patientapp/filter.go) |
| 4 | High | `GET /patient/{id}` always 500s: `api.Handler` builds `patientapp.Config` without `HospitalPatientBus`, so the handler calls `Query` on a nil interface | [api.go](../src/api/api.go) |
| 5 | High | No route is protected and login issues no token, so patient data is world-readable across hospitals | [api.go](../src/api/api.go), [staffapp.go](../src/app/domain/staffapp/staffapp.go) |
| 6 | Medium | `UPDATE users` has a trailing comma before `WHERE` → syntax error (unrouted, so latent) | [userdb.go](../src/business/domain/userbus/stores/userdb/userdb.go) |
| 7 | Medium | `userdb.Create` does not map the unique violation, so a duplicate username returns 500 instead of 409 | [userdb.go](../src/business/domain/userbus/stores/userdb/userdb.go) |
| 8 | Medium | `mid.Cors` panics at startup on an empty origin list (`conflict settings: all origins disabled`). With no `config.yaml` in the image, an unset `CORS_ORIGINS` kills the process at boot instead of falling back to a default | [cors.go](../src/app/sdk/mid/cors.go), [main.go](../src/main.go) |
| 9 | Medium | Image has no `ENTRYPOINT`/`CMD` and runs as root; `docker run` on the image alone does nothing useful | [Dockerfile](../Dockerfile) |
| 10 | Low | `make test-integration` targets `./repository/...`, a package tree that no longer exists | [Makefile](../Makefile) |
| 11 | Low | `env.Load` error is discarded in `main`. A naive fail-fast fix breaks the container, where the config file is absent by design — the check has to tolerate "file not found" and still fail on a malformed file | [main.go](../src/main.go) |
| 12 | Low | Validator messages leak Go struct and field names to clients (`Key: 'NewUser.Password' Error:...`) | [response.go](../src/app/sdk/response/response.go) |
| 13 | Low | `meta.total` is the current page's row count; `total_pages` is never set | all `*app` Query handlers |
| 14 | Low | Patient search filters on unindexed columns and on `lower(...)` expressions — sequential scan per request | [patientdb/filter.go](../src/business/domain/patientbus/stores/patientdb/filter.go) |
| 15 | Low | No liveness/readiness endpoint in the app; the compose healthcheck probes a db-backed list endpoint, and nginx's `/healthz` stays green while the API is down | [docker-compose.yml](../docker-compose.yml) |
| 16 | Low | `userdb.Query` selects `*` instead of an explicit column list, so a schema change silently changes the result set | [userdb.go](../src/business/domain/userbus/stores/userdb/userdb.go) |

---

## 4. Workstreams

Sizing is relative: **S** ≤ half a day, **M** ≈ 1–2 days, **L** ≈ 3–5 days.
Sequence matters more than the estimates.

### Phase 0 — Make it run (blockers)

| Task | Size | Acceptance |
| --- | --- | --- |
| Point the stale patient search sub-test at `phone_number` | S | `make test` green; `docker compose build` reaches the build step |
| Fix the `userdb` insert (columns/placeholders, id) | S | `POST /user` creates a row against a real postgres |
| Add `DEFAULT nextval('<table>_id_seq')` to all six id columns in the migration, then drop the explicit `nextval()` from the five inserts that compensate | M | Every create works with the id omitted; `make migrate-down && make migrate-up` is clean |
| Wire `HospitalPatientBus` into `patientapp.Config` | S | `GET /patient/{id}` returns the patient and its HN |
| Add a test that asserts the production route table and the wiring in `api.Handler` | S | A missing business in a `Config` fails a test, not a request |
| Fix the `UPDATE users` syntax error | S | Statement parses; covered once update is routed |
| Map the unique violation in `userdb.Create` | S | Duplicate username returns 409, matching every other resource |
| Default the CORS origin list instead of panicking on empty | S | Process starts with no config file and no `CORS_ORIGINS` |
| Point `make test-integration` at a path that exists | S | Target runs |
| Check the `env.Load` error in `main`, tolerating a missing file | S | Malformed config fails fast; container start is unaffected |

Exit criteria: every create and list endpoint works end to end against a local
postgres, verified by hand or by the Phase 3 integration tests, and `docker compose up`
serves them through nginx.

### Phase 1 — Security

Assumption to confirm (see §8): staff authenticate, and a staff member may only see
patients of hospitals they are assigned to.

| Task | Size | Acceptance |
| --- | --- | --- |
| Issue and verify JWTs; add an auth middleware in `app/sdk/mid` that puts the caller's claims in the context | M | Protected routes reject a missing/expired/tampered token with 401 |
| Return a token from `POST /staff/login` (and a refresh path, if sessions are needed) | M | Valid credentials return a token; 401/403 behaviour is unchanged |
| Decide which routes are public; protect the rest | S | Route table in api-spec.md marks each endpoint public or protected |
| Scope patient reads to the caller's hospitals via `hospital_staffs` | L | A staff member cannot read a patient registered only at another hospital, proven by test |
| Return 401/403 distinctly and never leak whether a resource exists to an unauthorised caller | S | Tests assert 403 (not 404) for cross-hospital access |
| Run the container as a non-root user; drop the build-time test run in favour of CI | S | Image runs as uid ≠ 0 and builds without a database or test dependency |

Password hashing landed before this review; `userbus.Authenticate` is the entry point
the middleware should build on.

### Phase 2 — Contract completeness

| Task | Size | Acceptance |
| --- | --- | --- |
| Route `Update` for every domain (`PATCH /{resource}/{id}`) with partial-update semantics | L | Only supplied fields change; `updated_at` moves; 404 on unknown id |
| Give the patient response an `id` and route `patientapp.Create` / `Delete` | S | A search result can be followed to `GET /patient/{id}` |
| Add `Count` to each store and populate `meta.total` / `meta.total_pages` | M | Paged list reports the full match count |
| Return 201 with a `Location` header on create | S | Contract and spec updated together |
| Normalise validation errors into a field-keyed structure, and align `error.code` casing with nginx's | M | 400 body lists offending fields without Go type names; one casing convention end to end |
| Bind `date_of_birth` as a date (`time_format`) and ignore an empty value | S | `?date_of_birth=1990-05-04` works; `?date_of_birth=` behaves like omitting it |
| Patient search shaped for the real use case (national id / passport / name, hospital-scoped) with supporting indexes | M | One request answers "find this patient in my hospital", no sequential scan |
| Endpoint to close a `hospital_staff` assignment (set `effective_to`) rather than hard-deleting it | S | History is preserved; `uq_sha_active` still holds |
| `GET /health` (liveness) and `GET /ready` (db ping), and point the compose healthcheck at them | S | Both return 200 with the process healthy; the probe no longer needs a query |

### Phase 3 — Test depth

| Task | Size | Acceptance |
| --- | --- | --- |
| Integration harness: postgres via testcontainers (or dockertest), migrations applied per run | M | `make test-integration` boots a throwaway db and tears it down |
| Store-layer tests for all six stores: create/get/query/update/delete, filters, paging, ordering | L | Every SQL statement in the repo is executed at least once |
| Constraint tests: unique violations → 409, FK violations → 404/422, soft delete hides rows | M | Each mapping asserted against real postgres error codes |
| Transaction test: a handler that fails mid-request leaves no rows behind | M | Rollback proven, not assumed |
| Business tests for `patientbus` and `hospitalbus`, the two without their own suite | S | All six businesses covered directly, not only through handlers |
| Add `-race` and a coverage floor to `make test` | S | `make test` matches what the Dockerfile runs today |

### Phase 4 — Operations

| Task | Size | Acceptance |
| --- | --- | --- |
| CI: build, vet, unit tests, integration tests, lint on every push | M | Pipeline red on any failure |
| Single config path: read everything from env, keep `config.yaml` as a local default only, and make `env.Load` agree with the filename it is given | M | App and migrate CLI agree on one connection string |
| Make the listen address and log level configurable | S | No hardcoded `:8000` |
| Log the `X-Request-ID` nginx already sends, in every line of a request | S | A response header can be grepped straight to its log lines |
| Structured error logging with stack context at the boundary | S | 500s are traceable from the response to a log line |
| TLS at the edge and a hardened nginx image (non-root, read-only fs) | M | HTTPS locally with a self-signed cert; container passes a basic hardening check |

---

## 5. Backlog

Single ordered list, for picking up work without re-reading the phases.

| Priority | Item | Phase | Status |
| --- | --- | --- | --- |
| P0 | Stale patient search sub-test — `make test` / image build red | 0 | todo |
| P0 | `userdb` insert placeholders / missing id | 0 | todo |
| P0 | Sequence defaults in migration | 0 | todo |
| P0 | `HospitalPatientBus` missing from `patientapp.Config` | 0 | todo |
| P0 | ~~Password hashing~~ | 1 | done |
| P1 | JWT auth + middleware + token from login | 1 | todo |
| P1 | Hospital-scoped authorisation for patient reads | 1 | todo |
| P1 | Route/wiring test for `api.Handler` | 0 | todo |
| P1 | Integration harness + store tests | 3 | todo |
| P1 | `userdb` duplicate mapping, `UPDATE users` syntax | 0 | todo |
| P1 | CORS panic on empty origin list | 0 | todo |
| P2 | Update endpoints | 2 | todo |
| P2 | Patient `id` in the response + create/delete routes | 2 | todo |
| P2 | `Count` / real pagination metadata | 2 | todo |
| P2 | Validation error normalisation + `error.code` casing | 2 | todo |
| P2 | Patient search shape + indexes | 2 | todo |
| P2 | CI pipeline | 4 | todo |
| P2 | ~~Docker + compose + nginx edge~~ | 4 | done |
| P3 | 201 + `Location` on create | 2 | todo |
| P3 | Health / readiness endpoints + compose probe | 2 | todo |
| P3 | `date_of_birth` binding | 2 | todo |
| P3 | Close-assignment endpoint for `hospital_staff` | 2 | todo |
| P3 | Request id in app logs, log configuration | 4 | todo |
| P3 | Non-root container, TLS at the edge | 1 / 4 | todo |
| P3 | Makefile integration target path, `env.Load` error check | 0 | todo |

---

## 6. Testing strategy

| Level | Scope | Tooling | State |
| --- | --- | --- | --- |
| Unit — business | Domain logic, error mapping, defaults, role validation | testify mocks over `Store` | done for `user`, `staff`, `hospital_staff`, `hospital_patient`; `patient` and `hospital` covered only through the app tests |
| Unit — app | Binding, validation, status mapping, filter/page/order pass-through, transaction scoping | `httptest` + real business + store mock | done for the four routed packages; one sub-test stale and failing |
| Wiring | Production route table, `api.Handler` config | `httptest` over `api.Handler` | **missing** |
| Integration — store | Every SQL statement, constraint mappings, soft delete | testcontainers postgres | **missing** |
| End to end | A few flows across domains (register staff → log in → search patient) | compose + HTTP client | **missing** |
| Edge | Rate limits, error envelopes, `/healthz` | compose + HTTP client | **missing** |

Conventions worth keeping in new tests:

- One `t.Run` per behaviour, named as a sentence.
- Assert what the store was called with, not just what came back — that is what caught
  the filter and default-value regressions.
- Always assert the negative: `store.AssertNotCalled(...)` when validation should have
  stopped the request before the business layer.
- When a binding tag changes, change the sub-test in the same commit. Defect 3 is what
  the other order looks like.

---

## 7. Adding a new domain

Checklist, in the order the existing domains were built:

1. `src/business/domain/<x>bus/` — `model.go` (entity + `New<X>` + domain constants),
   `filter.go` (`QueryFilter`), `order.go` (`OrderBy*` constants + `DefaultOrderBy`),
   `<x>bus.go` (`Store` and `Business` interfaces, sentinel errors, `business` impl).
2. `.../stores/<x>db/` — `model.go` (db struct + mappers, pointers for nullable
   columns), `order.go` (`orderByFields` + `orderByClause`), `filter.go`
   (`applyFilters`, including the soft-delete predicate), `<x>db.go` (CRUD + constraint
   mapping), `<x>db_mock.go`.
3. `src/app/domain/<x>app/` — `model.go` (JSON models, binding tags, mappers),
   `filter.go` (`queryParams` + `parseFilter`), `order.go` (public name → constant),
   `<x>app.go` (handlers + `bus(c)` + `parseID`), `route.go`.
4. Wire `BusConfig` in [api.go](../src/api/api.go) and construct the store/business
   pair in [main.go](../src/main.go). Every field of the app `Config` must be set — a
   forgotten one is a nil interface and a 500, not a compile error.
5. Tests: app-level suite covering success, validation, each error mapping, filters,
   and the transaction-scoped store.
6. Document the resource in [api-spec.md](./api-spec.md) and any schema change in
   [er-diagram.md](./er-diagram.md).
7. Regenerate the combined document (§10).

Copy the `staff` domain — it is the most complete reference (nullable columns, unique
constraints, full test suite).

---

## 8. Open questions

These change the shape of Phase 1 and 2 and should be settled before that work starts.

1. **Who are the API's clients?** Staff-facing UI only, or other hospital systems too?
   Determines whether auth is user-session or service-to-service.
2. **Is patient data hospital-scoped?** The `hospital_patients` table implies yes. If a
   staff member may only see their own hospitals' patients, that constraint belongs in
   the business layer, not in handlers.
3. **Are the `users` and `staffs` tables one concept or two?** Right now a staff row
   points at a user row, but nothing enforces that a staff-facing endpoint is called by
   the matching user.
4. **What identifies a patient at intake** — national id, passport, or HN? Drives the
   search endpoint and which uniqueness rules matter.
5. **Is hard-deleting a `hospital_staff` assignment acceptable**, or must history be
   retained (in which case add `deleted_at`, or only ever close with `effective_to`)?
6. **Where does registration get real staff details?** `POST /staff/create` writes a
   blank `first_name`/`last_name` and seeds `employee_code` from the username. Should
   the request carry them, or is a follow-up update endpoint the intended path?
7. **Deployment target** — container platform, expected traffic, single or multi
   tenant? Determines how much of Phase 4 is required, and whether TLS belongs in nginx
   or in front of it.

---

## 9. Definition of done

A change is complete when:

- `make test` passes (build, vet, unit tests) and integration tests pass once Phase 3
  lands.
- New SQL is exercised by an integration test.
- Error paths return the documented status code, not just the happy path.
- [api-spec.md](./api-spec.md) reflects any contract change, and
  [er-diagram.md](./er-diagram.md) any schema change, in the same commit.
- No new hardcoded configuration.
- Any deliberate shortcut is written down here, in §3, rather than left in the code.

---

## 10. Documentation

Three markdown files are the source of truth, in reading order:

| File | Covers |
| --- | --- |
| [development-plan.md](./development-plan.md) | This document: status, defects, plan |
| [er-diagram.md](./er-diagram.md) | Schema, constraints, indexes, design notes |
| [api-spec.md](./api-spec.md) | HTTP contract, conventions, edge behaviour, known gaps |

`agnos-backend-assignment-docs.docx` is a generated compilation of all three, in that
order, with a title page and a table of contents. Rebuild it after editing any of them:

```bash
python3 -m pip install python-docx   # once
python3 docs/build-docx.py
```

The generator reads the markdown directly, so the docx never drifts from it: headings,
tables, lists, inline code and fenced blocks (including the mermaid ER diagram, kept as
source text) are carried across. Nothing is edited in Word — a hand edit there is lost
on the next build.
