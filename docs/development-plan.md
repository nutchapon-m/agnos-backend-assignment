# Development Plan

Working plan for the hospital middleware backend. Companion to
[api-spec.md](./api-spec.md), which documents the contract as it stands today.

- **Repository**: `agnos-backend-assignment`
- **Language / stack**: Go, Gin, sqlx, PostgreSQL, golang-migrate, viper, testify
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

Rules the code follows today, worth keeping:

- The app layer never touches SQL; the store layer never formats HTTP.
- Business packages own their sentinel errors (`ErrNotFound`, `ErrDuplicate`, …);
  the app layer maps those to status codes and nothing else.
- Stores translate postgres constraint codes into `sqldb.Err*` values, so business
  code never sees a driver type.
- `Store` and `Business` are interfaces, so every layer is testable with a mock.
- Optional text columns are pointers in db models, keeping `NULL` distinct from `""`
  and keeping partial unique indexes honest.

### Domains

| Domain | Business | Store | Store mock | App | Routes | Tests |
| --- | --- | --- | --- | --- | --- | --- |
| `user` | done | done (2 SQL bugs) | done | done | 4 | bus + app |
| `staff` | done | done | done | done | 4 | bus + app |
| `patient` | done | done | done | done | 4 | app |
| `hospital` | done | done | done | done | 4 | app |
| `hospital_patient` | done | done | done | done | 4 | app |
| `hospital_staff` | done | done | done | done | 4 | app |

Every domain exposes create, list (filter + page + order), get-by-id and delete.
`Update` exists at the store layer everywhere but is not routed.

### Test coverage

175 tests across 28 packages, all passing (`make test`). They are unit tests only:
handlers run against the real business layer with a mocked store, over `httptest`.

**Not covered:** every line of SQL. No test executes a query against postgres, which
is exactly why the two `userdb` bugs below survived.

### Infrastructure

- Config: `config.yaml` read through viper (`db.*` keys); `.env` holds `DATABASE_URL`
  for the migrate CLI only — the app itself does not read it.
- Migrations: `src/business/sdk/migrate/sql/000001_init-tables.{up,down}.sql`, run
  either by the `make migrate-*` targets or by starting the binary with `-migration`.
- No Dockerfile, no docker-compose, no CI pipeline.

---

## 3. Known defects

Ordered by severity. Items 1 and 2 mean the user endpoint cannot work against a real
database, so they gate everything else.

| # | Severity | Defect | Location |
| --- | --- | --- | --- |
| 1 | Blocker | `INSERT` lists 4 columns against 5 placeholders (`$1..$5`) — every create fails | [userdb.go:47](../src/business/domain/userbus/stores/userdb/userdb.go#L47) |
| 2 | Blocker | Same insert omits `id`, which has no `DEFAULT nextval(...)` → not-null violation. Other stores work around this with an explicit `nextval()`; the migration should set the default instead | [000001_init-tables.up.sql](../src/business/sdk/migrate/sql/000001_init-tables.up.sql) |
| 3 | High | Passwords are stored in plaintext | [userbus.go](../src/business/domain/userbus/userbus.go) |
| 4 | Medium | `UPDATE users` has a trailing comma before `WHERE` → syntax error (unrouted, so latent) | [userdb.go](../src/business/domain/userbus/stores/userdb/userdb.go) |
| 5 | Medium | `userdb.Create` does not map the unique violation, so a duplicate username returns 500 instead of 409 | [userdb.go](../src/business/domain/userbus/stores/userdb/userdb.go) |
| 6 | Low | `make test-integration` targets `./repository/...`, a package tree that no longer exists | [Makefile](../Makefile) |
| 7 | Low | `env.Load` error is discarded in `main`, so a missing/broken config surfaces later as confusing DB errors | [main.go:52](../src/main.go#L52) |
| 8 | Low | Validator messages leak Go struct and field names to clients (`Key: 'NewUser.Password' Error:...`) | [response.go](../src/app/sdk/response/response.go) |
| 9 | Low | `meta.total` is the current page's row count; `total_pages` is never set | all `*app` Query handlers |

---

## 4. Workstreams

Sizing is relative: **S** ≤ half a day, **M** ≈ 1–2 days, **L** ≈ 3–5 days.
Sequence matters more than the estimates.

### Phase 0 — Make it run (blockers)

| Task | Size | Acceptance |
| --- | --- | --- |
| Fix the `userdb` insert (columns/placeholders, id) | S | `POST /user` creates a row against a real postgres |
| Add `DEFAULT nextval('<table>_id_seq')` to all six id columns in the migration, then drop the explicit `nextval()` from the six inserts | M | Every create works with the id omitted; `make migrate-down && make migrate-up` is clean |
| Fix the `UPDATE users` syntax error | S | Statement parses; covered once update is routed |
| Map the unique violation in `userdb.Create` | S | Duplicate username returns 409, matching every other resource |
| Point `make test-integration` at a path that exists | S | Target runs |
| Check the `env.Load` error in `main` | S | Bad config fails fast with a clear message |

Exit criteria: the six create endpoints and the six list endpoints work end to end
against a local postgres, verified by hand or by the Phase 3 integration tests.

### Phase 1 — Security

Assumption to confirm (see §8): staff authenticate, and a staff member may only see
patients of hospitals they are assigned to.

| Task | Size | Acceptance |
| --- | --- | --- |
| Hash passwords (bcrypt or argon2id) in `userbus.Create`; add `Authenticate(ctx, username, password)` | M | Stored hash is never reversible; wrong password is indistinguishable in timing from unknown user |
| Issue and verify JWTs; add an auth middleware in `app/sdk/mid` that puts the caller's claims in the context | M | Protected routes reject a missing/expired/tampered token with 401 |
| Add `POST /auth/login` (and refresh, if sessions are needed) | M | Returns a token for valid credentials, 401 otherwise |
| Decide which routes are public; protect the rest | S | Route table in api-spec.md marks each endpoint public or protected |
| Scope patient reads to the caller's hospitals via `hospital_staffs` | L | A staff member cannot read a patient registered only at another hospital, proven by test |
| Return 401/403 distinctly and never leak whether a resource exists to an unauthorised caller | S | Tests assert 403 (not 404) for cross-hospital access |

### Phase 2 — Contract completeness

| Task | Size | Acceptance |
| --- | --- | --- |
| Route `Update` for every domain (`PATCH /{resource}/{id}`) with partial-update semantics | L | Only supplied fields change; `updated_at` moves; 404 on unknown id |
| Add `Count` to each store and populate `meta.total` / `meta.total_pages` | M | Paged list reports the full match count |
| Return 201 with a `Location` header on create | S | Contract and spec updated together |
| Normalise validation errors into a field-keyed structure | M | 400 body lists offending fields without Go type names |
| Patient search endpoint shaped for the real use case (national id / passport / name, hospital-scoped) | M | One request answers "find this patient in my hospital" |
| Endpoint to close a `hospital_staff` assignment (set `effective_to`) rather than hard-deleting it | S | History is preserved; `uq_sha_active` still holds |
| `GET /health` (liveness) and `GET /ready` (db ping) | S | Both return 200 with the process healthy |

### Phase 3 — Test depth

| Task | Size | Acceptance |
| --- | --- | --- |
| Integration harness: postgres via testcontainers (or dockertest), migrations applied per run | M | `make test-integration` boots a throwaway db and tears it down |
| Store-layer tests for all six stores: create/get/query/update/delete, filters, paging, ordering | L | Every SQL statement in the repo is executed at least once |
| Constraint tests: unique violations → 409, FK violations → 422, soft delete hides rows | M | Each mapping asserted against real postgres error codes |
| Transaction test: a handler that fails mid-request leaves no rows behind | M | Rollback proven, not assumed |
| Add `-race` and a coverage floor to the test target | S | `make test` runs with `-race`; coverage reported |

### Phase 4 — Operations

| Task | Size | Acceptance |
| --- | --- | --- |
| Dockerfile (multi-stage, non-root) and docker-compose with postgres | M | `docker compose up` yields a working API and db |
| Single config path: read everything from env, keep `config.yaml` as local default only | M | App and migrate CLI agree on one connection string |
| Make the listen address, CORS origins and log level configurable | S | No hardcoded `:8000` or `localhost:3000` |
| CI: build, vet, unit tests, integration tests, lint on every push | M | Pipeline red on any failure |
| Request id middleware, propagated into log lines | S | Every log line for a request shares an id |
| Structured error logging with stack context at the boundary | S | 500s are traceable from the response to a log line |

---

## 5. Backlog

Single ordered list, for picking up work without re-reading the phases.

| Priority | Item | Phase | Status |
| --- | --- | --- | --- |
| P0 | `userdb` insert placeholders / missing id | 0 | todo |
| P0 | Sequence defaults in migration | 0 | todo |
| P0 | Password hashing | 1 | todo |
| P1 | JWT auth + middleware + login endpoint | 1 | todo |
| P1 | Hospital-scoped authorisation for patient reads | 1 | todo |
| P1 | Integration harness + store tests | 3 | todo |
| P1 | `userdb` duplicate mapping, `UPDATE users` syntax | 0 | todo |
| P2 | Update endpoints | 2 | todo |
| P2 | `Count` / real pagination metadata | 2 | todo |
| P2 | Validation error normalisation | 2 | todo |
| P2 | Patient search endpoint | 2 | todo |
| P2 | Docker + compose + CI | 4 | todo |
| P3 | 201 + `Location` on create | 2 | todo |
| P3 | Health / readiness endpoints | 2 | todo |
| P3 | Close-assignment endpoint for `hospital_staff` | 2 | todo |
| P3 | Request id + log configuration | 4 | todo |
| P3 | Makefile integration target path, `env.Load` error check | 0 | todo |

---

## 6. Testing strategy

| Level | Scope | Tooling | State |
| --- | --- | --- | --- |
| Unit — business | Domain logic, error mapping, defaults | testify mocks over `Store` | done for `user`, `staff`; other four are covered indirectly through the app tests |
| Unit — app | Binding, validation, status mapping, filter/page/order pass-through, transaction scoping | `httptest` + real business + store mock | done for all six |
| Integration — store | Every SQL statement, constraint mappings, soft delete | testcontainers postgres | **missing** |
| End to end | A few flows across domains (register staff → assign to hospital → register patient → search) | compose + HTTP client | **missing** |

Conventions worth keeping in new tests:

- One `t.Run` per behaviour, named as a sentence.
- Assert what the store was called with, not just what came back — that is what caught
  the filter and default-value regressions.
- Always assert the negative: `store.AssertNotCalled(...)` when validation should have
  stopped the request before the business layer.

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
   pair in [main.go](../src/main.go).
5. Tests: app-level suite covering success, validation, each error mapping, filters,
   and the transaction-scoped store.
6. Document the resource in [api-spec.md](./api-spec.md).

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
6. **Deployment target** — container platform, expected traffic, single or multi
   tenant? Determines how much of Phase 4 is required.

---

## 9. Definition of done

A change is complete when:

- `make test` passes (build, vet, unit tests) and integration tests pass once Phase 3
  lands.
- New SQL is exercised by an integration test.
- Error paths return the documented status code, not just the happy path.
- [api-spec.md](./api-spec.md) reflects any contract change in the same commit.
- No new hardcoded configuration.
- Any deliberate shortcut is written down here, in §3, rather than left in the code.
