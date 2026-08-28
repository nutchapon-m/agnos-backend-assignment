# API Specification

Hospital middleware API. All endpoints are versioned under `/api/v1`.

- **Base URL**: `http://localhost:8000/api/v1` — the Go process listens on `:8000`
  directly, and under `docker compose` nginx publishes the same port (`HTTP_PORT`,
  default `8000`) and proxies to the API container
- **Content type**: `application/json` on every request with a body
- **Request size**: behind nginx a body is capped at 1 MB (`client_max_body_size`)
- **CORS**: origins come from `cors_origins` / `CORS_ORIGINS`, default
  `http://localhost:3000` (credentials enabled). CORS is owned by the API, not by
  nginx, so a browser sees exactly one set of `Access-Control-*` headers.
- **Auth**: `POST /staff/login` verifies credentials and answers with the caller's
  identity, but issues **no token**. Nothing on the wire carries a session, so every
  endpoint below is reachable without logging in first.
- **Last reviewed**: 2026-08-28

## Route table

Twelve routes are registered, from four `*app` packages wired in
[api.go](../src/api/api.go).

| Method | Path | Handler | Transaction |
| --- | --- | --- | --- |
| `POST` | `/user` | `userapp.Create` | yes |
| `GET` | `/user` | `userapp.Query` | no |
| `GET` | `/user/{id}` | `userapp.GetByID` | no |
| `DELETE` | `/user/{id}` | `userapp.Delete` | yes |
| `POST` | `/staff/create` | `staffapp.Create` | yes |
| `POST` | `/staff/login` | `staffapp.Login` | no |
| `GET` | `/patient/search` | `patientapp.Query` | no |
| `GET` | `/patient/{id}` | `patientapp.GetByID` | no |
| `POST` | `/hospital` | `hospitalapp.Create` | yes |
| `GET` | `/hospital` | `hospitalapp.Query` | no |
| `GET` | `/hospital/{id}` | `hospitalapp.GetByID` | no |
| `DELETE` | `/hospital/{id}` | `hospitalapp.Delete` | yes |

`GET /healthz` is answered by nginx itself, outside `/api/v1`, and never reaches the
Go process. See [Edge behaviour](#edge-behaviour).

Routes marked *transaction* run inside `mid.BeginCommitRollback`: the transaction
commits when the handler writes a 2xx and rolls back on any status ≥ 400.

There are **no HTTP resources for `hospital-patient` or `hospital-staff`**. Both
business packages still exist and are used from inside the staff and patient
handlers, but the `*app` packages that exposed them over HTTP were removed. Neither
is `patientapp.Create` / `patientapp.Delete` nor `staffapp.GetByID` / `Query` /
`Delete` routed — those handlers are written and tested but unreachable.

A request to an unmatched path *under* `/api/v1` reaches gin and gets its default
`404 page not found` — plain text, not the envelope below. Any other path is answered
by nginx, in the envelope.

## Edge behaviour

Under `docker compose` every request passes through nginx before the API. What the
proxy answers on its own, from [nginx.conf](../nginx.conf):

| Path / condition | Status | Body |
| --- | --- | --- |
| `GET /healthz` | 200 | `{"success":true,"data":{"status":"ok"}}` — proxy liveness, **not** app or db health |
| Anything outside `/api/v1/` | 404 | `{"success":false,"error":{"code":"NOT_FOUND","message":"route not found"}}` |
| Rate limit exceeded | 429 | `{"success":false,"error":{"code":"TOO_MANY_REQUESTS","message":"rate limit exceeded, slow down"}}` |
| API unreachable or 502/503/504 | 503 | `{"success":false,"error":{"code":"SERVICE_UNAVAILABLE","message":"upstream api is unavailable"}}` |

Note that `NOT_FOUND`, `TOO_MANY_REQUESTS` and `SERVICE_UNAVAILABLE` are
`SCREAMING_SNAKE_CASE`, while every `error.code` the Go app emits is title-cased
(`Invalid Argument`, `Patient Not Found`). A client that switches on `error.code`
has to handle both spellings.

Per-client limits, keyed on the caller's IP:

| Bucket | Methods | Rate | Burst |
| --- | --- | --- | --- |
| `api_read` | `GET`, `HEAD`, … | 30 r/s | 60 |
| `api_write` | `POST`, `PUT`, `PATCH`, `DELETE` | 10 r/s | 20 |

Concurrent connections are capped at 32 per IP. Every request also carries an
`X-Request-ID` — the caller's own if present, otherwise one minted by nginx — which is
echoed back on the response. The Go logger does not yet include it in log lines.

Requests that never reached the API are retried once against the upstream; proxy read
and send timeouts are 15s, comfortably above the API's own 5s read / 10s write.

## Response envelope

Every response — success or failure — uses the same envelope. Empty fields are omitted.

```json
{
  "success": true,
  "data": {},
  "error": { "code": "string", "message": "string" },
  "meta": { "page": 1, "per_page": 10, "total": 2, "total_pages": 0 }
}
```

| Field | When present | Notes |
| --- | --- | --- |
| `success` | always | `true` on 2xx, `false` otherwise |
| `data` | success | object for single-resource endpoints, array for list endpoints, omitted on delete |
| `error` | failure | `code` is a short human-readable label, `message` carries the detail |
| `meta` | list endpoints | pagination echo |

### Status codes

| Code | `error.code` | Meaning |
| --- | --- | --- |
| 200 | — | Success, including create and delete |
| 400 | `Invalid Argument` | Malformed JSON, failed field validation, bad path/query param |
| 401 | `Unauthorized` | Login: bad credentials, or a user with no staff record |
| 403 | `Staff Inactive`, `Hospital Not Allowed` | Login: the account is real but may not use this hospital |
| 404 | `<Resource> Not Found` | No such id (or it is soft deleted); also an unknown `hospital` on staff creation |
| 409 | `<Resource> Already Exist`, `Staff Already Assigned` | Unique constraint violation |
| 500 | `Internal Server Error` | Unexpected failure |

## Conventions

### Paging

Applies to `GET /user`, `GET /patient/search` and `GET /hospital`.

| Param | Type | Default | Rules |
| --- | --- | --- | --- |
| `page` | int | `1` | must be > 0 |
| `limit` | int | `10` | must be > 0 and <= 100 |

Out-of-range values return 400. `meta.total` is the number of rows **on the current
page**, and `meta.total_pages` is not computed — there is no count query yet.

### Ordering

`order_by=<field>[,<direction>]`, where direction is `ASC` (default) or `DESC`.
An unknown field or direction returns 400. Allowed fields per endpoint:

| Endpoint | Fields | Default |
| --- | --- | --- |
| `GET /user` | `id`, `created_at` | `id,ASC` |
| `GET /patient/search` | `id`, `created_at` | `id,ASC` |
| `GET /hospital` | `id`, `code`, `created_at` | `id,ASC` |

### Formats

- Timestamps (`created_at`, `updated_at`): RFC3339 — `2024-01-02T15:04:05Z`
- `date_of_birth` in a **request body**: `YYYY-MM-DD`
- `date_of_birth` in a **query string**: RFC3339 — see [Patient search](#search--get-patientsearch)
- `date_of_birth` in a **response**: `YYYY-MM-DD`
- Path ids must be integers greater than 0

### Soft delete

`DELETE` stamps `deleted_at` and the row disappears from every read. Deleting an
unknown id returns 404, not 200. Only `user` and `hospital` expose a delete route.

---

## Staff

`/api/v1/staff`

Staff are not created as a bare resource. One request creates the login account, the
staff record that belongs to it, and the staff member's assignment to a hospital, all
inside a single transaction — a half-finished registration is never committed.

### Register — `POST /staff/create`

| Field | Type | Required | Rules |
| --- | --- | --- | --- |
| `username` | string | yes | 3–255 chars |
| `password` | string | yes | min 6 chars |
| `hospital` | int | yes | > 0, must reference an existing hospital |

```json
{ "username": "somchai", "password": "secret123", "hospital": 7 }
```

The request carries no staff details. `employee_code` is seeded from `username` —
the column is `NOT NULL` under a unique index and the username is the only unique
value the request supplies — and the names are left blank for a later update. The
assignment is created with `role = "registrar"` and `is_primary = true`.

### Response object

```json
{
  "staff": {
    "id": 42,
    "user_id": 17,
    "employee_code": "somchai",
    "first_name": "",
    "last_name": "",
    "is_active": true,
    "created_at": "2024-01-02T15:04:05Z",
    "updated_at": "2024-01-02T15:04:05Z"
  },
  "hospital_id": 7,
  "role": "registrar"
}
```

The password never leaves the business layer; the user is represented only by
`staff.user_id`. `email` and `license_no` are omitted when empty, `first_name` and
`last_name` are not.

### Errors

| Code | `error.code` | Cause |
| --- | --- | --- |
| 400 | `Invalid Argument` | Validation failure, or a password bcrypt cannot hash (over 72 bytes) |
| 404 | `Hospital Not Found` | `hospital` does not reference an existing hospital |
| 409 | `User Already Exist` | Duplicate `username` |
| 409 | `Staff Already Exist` | Duplicate `employee_code` |
| 409 | `Staff Already Assigned` | The staff already holds an open assignment to that hospital, or already has a primary hospital |

> This endpoint cannot succeed against a real postgres today: `userdb.Create` is
> broken (see [Known gaps](#known-gaps) 1). It returns 500 at the first step.

### Login — `POST /staff/login`

A staff member logs in **against one hospital**: the credentials alone are not
enough, the account must also hold an open assignment to that hospital.

| Field | Type | Required | Rules |
| --- | --- | --- | --- |
| `username` | string | yes | 3–255 chars |
| `password` | string | yes | min 6 chars |
| `hospital` | int | yes | > 0 |

```json
{ "username": "somchai", "password": "secret123", "hospital": 7 }
```

### Response object

```json
{
  "authenticate": true,
  "user_id": 17,
  "staff_id": 42,
  "employee_code": "somchai",
  "first_name": "",
  "last_name": "",
  "hospital_id": 7,
  "role": "registrar"
}
```

**No token, cookie or session is issued.** The response states who the caller is; it
does not let them prove it on the next request.

### Errors

| Code | `error.code` | Message | Cause |
| --- | --- | --- | --- |
| 400 | `Invalid Argument` | binding error | Validation failure |
| 401 | `Unauthorized` | `invalid username or password` | Unknown username, wrong password, **or** a user with no staff record |
| 403 | `Staff Inactive` | `staff account is not active` | `staffs.is_active` is false |
| 403 | `Hospital Not Allowed` | `staff is not assigned to this hospital` | No open assignment to `hospital` |
| 500 | `Internal Server Error` | — | Store failure |

The three 401 causes are deliberately indistinguishable, and an unknown username
burns the same bcrypt cost as a real comparison, so the endpoint cannot be used to
discover which usernames exist. The 403s do distinguish themselves — by then the
caller has already proved the password.

---

## Patient

`/api/v1/patient`

Read-only over HTTP. Patients are not created or deleted through the API — the
handlers exist but no route reaches them.

### Search — `GET /patient/search`

| Filter | Type | Matches |
| --- | --- | --- |
| `id` | int | exact |
| `national_id` | string | exact |
| `passport_no` | string | exact |
| `first_name` | string | case-insensitive, Thai **or** English spelling |
| `middle_name` | string | case-insensitive, Thai **or** English spelling |
| `last_name` | string | case-insensitive, Thai **or** English spelling |
| `phone_number` | string | exact |
| `email` | string | case-insensitive |
| `date_of_birth` | RFC3339 | compared as a calendar date |

Plus `page`, `limit` and `order_by`.

A name filter matches when either spelling of that name part does:
`lower(first_name_th) = lower(:v) OR lower(first_name_en) = lower(:v)`. Filters
combine with `AND`.

`date_of_birth` is bound as a `time.Time` with no `time_format` tag, so gin parses it
as RFC3339, not as a date:

```
GET /patient/search?date_of_birth=1990-05-04T00:00:00Z   → 200
GET /patient/search?date_of_birth=1990-05-04             → 400
GET /patient/search?date_of_birth=                       → 200, but matches nothing
```

The last line is a trap: an empty value still binds, as the zero time, and the query
then filters on `date_of_birth = '0001-01-01'`. See [Known gaps](#known-gaps) 4.

Example: `GET /patient/search?last_name=ใจดี&page=1&limit=20&order_by=created_at,DESC`

### Response object

```json
{
  "national_id": "1234567890123",
  "first_name_th": "สมชาย",
  "last_name_th": "ใจดี",
  "date_of_birth": "1990-05-04",
  "gender": "M",
  "phone_number": "0812345678",
  "email": "somchai@example.com",
  "created_at": "2024-01-02T15:04:05Z",
  "updated_at": "2024-01-02T15:04:05Z"
}
```

Full field set: `national_id`, `passport_no`, `first_name_th`, `middle_name_th`,
`last_name_th`, `first_name_en`, `middle_name_en`, `last_name_en`, `date_of_birth`,
`patient_hn`, `gender`, `phone_number`, `email`, `created_at`, `updated_at`.
Everything except the two timestamps is omitted when empty.

**The patient object carries no `id`.** Search results therefore cannot be followed
to `GET /patient/{id}` — see [Known gaps](#known-gaps) 3.

`patient_hn` is only ever populated by `GET /patient/{id}`; it is always absent from
search results.

### Fetch one — `GET /patient/{id}`

Returns the same object, plus `patient_hn` taken from the patient's hospital
registrations. The lookup is not scoped to any hospital: it queries every
registration for that patient and returns the HN of the **last** row in `id ASC`
order.

| Code | `error.code` | Cause |
| --- | --- | --- |
| 400 | `Invalid Argument` | `id` is not a number, or is <= 0 |
| 404 | `Patient Not Found` | Unknown id, or the row is soft deleted |
| 500 | `Internal Server Error` | Store failure |

> This endpoint returns 500 today. `api.Handler` never wires `HospitalPatientBus`
> into `patientapp.Config`, so the handler dereferences a nil business and gin's
> recovery middleware turns the panic into a 500. See [Known gaps](#known-gaps) 2.

---

## User

`/api/v1/user`

The raw account resource. It is not part of the staff flow — `POST /staff/create`
creates its own user — and exists mainly as the CRUD reference implementation.

### Create — `POST /user`

| Field | Type | Required | Rules |
| --- | --- | --- | --- |
| `username` | string | yes | 3–255 chars |
| `password` | string | yes | min 6 chars |

```json
{ "username": "gopher", "password": "secret1" }
```

The password is stored as a bcrypt hash at default cost. The plaintext is never
persisted, returned, or logged.

### Response object

```json
{
  "id": 42,
  "username": "gopher",
  "created_at": "2024-01-02T15:04:05Z",
  "updated_at": "2024-01-02T15:04:05Z"
}
```

`password` is never returned.

### List — `GET /user`

| Filter | Type | Matches |
| --- | --- | --- |
| `id` | int | exact |

The store also filters on `username`, but no query parameter is bound to it — only
`userbus.Authenticate` uses that filter.

Example: `GET /user?page=2&limit=20&order_by=created_at,DESC`

### Errors

| Code | Cause |
| --- | --- |
| 400 | Validation failure, bad id |
| 404 | Unknown user |
| 409 | Duplicate `username` — currently returns 500, see [Known gaps](#known-gaps) 1 |
| 500 | A password bcrypt cannot hash (over 72 bytes) — `staffapp` maps this to 400, `userapp` does not |

> `POST /user` cannot succeed against a real postgres either — same broken insert as
> `POST /staff/create`.

---

## Hospital

`/api/v1/hospital`

### Create — `POST /hospital`

| Field | Type | Required | Rules |
| --- | --- | --- | --- |
| `code` | string | yes | max 20 |
| `name` | string | no | max 255 |
| `province_code` | string | no | exactly 2 digits |

```json
{ "code": "H001", "name": "Bangkok Hospital", "province_code": "10" }
```

`is_active` is set to `true` on create.

### Response object

```json
{
  "id": 42,
  "code": "H001",
  "name": "Bangkok Hospital",
  "province_code": "10",
  "is_active": true,
  "created_at": "2024-01-02T15:04:05Z",
  "updated_at": "2024-01-02T15:04:05Z"
}
```

### List — `GET /hospital`

| Filter | Type | Matches |
| --- | --- | --- |
| `id` | int | exact |
| `code` | string | exact |
| `province_code` | string | exact |
| `is_active` | bool | exact |

Example: `GET /hospital?province_code=10&is_active=true&order_by=code`

### Errors

| Code | Cause |
| --- | --- |
| 400 | Validation failure, bad id |
| 404 | Unknown hospital |
| 409 | Duplicate `code` |

---

## Examples

Create a hospital:

```bash
curl -X POST http://localhost:8000/api/v1/hospital \
  -H 'Content-Type: application/json' \
  -d '{"code":"H001","name":"Bangkok Hospital","province_code":"10"}'
```

```json
{
  "success": true,
  "data": {
    "id": 1,
    "code": "H001",
    "name": "Bangkok Hospital",
    "province_code": "10",
    "is_active": true,
    "created_at": "2024-01-02T15:04:05Z",
    "updated_at": "2024-01-02T15:04:05Z"
  }
}
```

Register a staff member at that hospital:

```bash
curl -X POST http://localhost:8000/api/v1/staff/create \
  -H 'Content-Type: application/json' \
  -d '{"username":"somchai","password":"secret123","hospital":1}'
```

```json
{
  "success": true,
  "data": {
    "staff": {
      "id": 1,
      "user_id": 1,
      "employee_code": "somchai",
      "first_name": "",
      "last_name": "",
      "is_active": true,
      "created_at": "2024-01-02T15:04:05Z",
      "updated_at": "2024-01-02T15:04:05Z"
    },
    "hospital_id": 1,
    "role": "registrar"
  }
}
```

Log in against a hospital the staff member is not assigned to:

```bash
curl -X POST http://localhost:8000/api/v1/staff/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"somchai","password":"secret123","hospital":99}'
```

```json
{
  "success": false,
  "error": {
    "code": "Hospital Not Allowed",
    "message": "staff is not assigned to this hospital"
  }
}
```

Search patients by name, one page at a time:

```bash
curl 'http://localhost:8000/api/v1/patient/search?last_name=Jaidee&page=1&limit=2&order_by=created_at,DESC'
```

```json
{
  "success": true,
  "data": [
    { "first_name_th": "สมศรี", "last_name_th": "ดี", "created_at": "2024-01-03T09:00:00Z", "updated_at": "2024-01-03T09:00:00Z" },
    { "first_name_th": "สมชาย", "last_name_th": "ใจดี", "created_at": "2024-01-02T15:04:05Z", "updated_at": "2024-01-02T15:04:05Z" }
  ],
  "meta": { "page": 1, "per_page": 2, "total": 2 }
}
```

Validation failure:

```bash
curl -X POST http://localhost:8000/api/v1/user \
  -H 'Content-Type: application/json' \
  -d '{"username":"gopher","password":"123"}'
```

```json
{
  "success": false,
  "error": {
    "code": "Invalid Argument",
    "message": "Key: 'NewUser.Password' Error:Field validation for 'Password' failed on the 'min' tag"
  }
}
```

Delete:

```bash
curl -X DELETE http://localhost:8000/api/v1/hospital/1
```

```json
{ "success": true }
```

---

## Known gaps

Current behaviours to be aware of, not planned contract changes. Severity-ordered;
the first two make endpoints unusable. Tracked in
[development-plan.md](./development-plan.md) §3.

1. **Every write to `users` fails.** `userdb.Create` lists 4 columns against 5
   placeholders and omits `id`, which has no `DEFAULT nextval(...)`. That breaks
   `POST /user` *and* `POST /staff/create`, which creates the user first. The same
   store also fails to translate the postgres unique violation, so a duplicate
   username would return 500 rather than 409 even once the insert is fixed.
2. **`GET /patient/{id}` returns 500.** `api.Handler` builds `patientapp.Config`
   without `HospitalPatientBus`, so the handler calls `Query` on a nil interface.
3. **The patient object has no `id`.** `GET /patient/search` cannot be followed to
   `GET /patient/{id}`, and a client has no stable key for a search result.
4. **`date_of_birth=` (empty) is not the same as omitting it.** An empty value binds
   to the zero time and filters on `0001-01-01`, silently matching nothing.
5. **Login proves nothing.** No token is issued and no route is protected, so
   `GET /patient/search` is world-readable and returns patients from every hospital.
   nginx's rate limit is the only thing between an anonymous caller and the whole
   patient table.
6. **`meta.total` counts the current page only** and `meta.total_pages` is always
   omitted; there is no `COUNT(*)` behind the list endpoints.
7. **Create returns 200, not 201**, and sets no `Location` header.
8. **No update endpoints.** `Update` is implemented in every store but not routed —
   which also means the blank `first_name` / `last_name` a registration writes can
   never be filled in over HTTP.
9. **Validation messages leak Go type names** (`Key: 'NewUser.Password' Error:...`).
10. **Ids come from explicit `nextval()` calls** in the insert statements, because
    the migration creates sequences `OWNED BY` the id columns without setting
    `DEFAULT nextval(...)`. `userdb` is the one store that forgot to compensate.
11. **The API has no health or readiness endpoint.** `/healthz` is nginx answering for
    itself, so it stays 200 while the API is down (nginx then returns the 503 envelope
    on `/api/v1/`). The compose healthcheck for the API container probes
    `GET /api/v1/hospital?page=1&limit=1`, which needs a working database — a liveness
    probe that fails on a db blip.
12. **Patient search runs sequential scans.** None of the searchable columns is
    indexed and the name and email comparisons are `lower(...)` expressions, which need
    expression indexes. Fine at assignment scale, not at hospital scale — see
    [er-diagram.md](./er-diagram.md) note 8.
