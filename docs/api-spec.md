# API Specification

Hospital middleware API. All endpoints are versioned under `/api/v1`.

- **Base URL**: `http://localhost:8000/api/v1`
- **Content type**: `application/json` on every request with a body
- **CORS**: allowed origin `http://localhost:3000` (credentials enabled)
- **Auth**: none yet — every endpoint is public

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
| 404 | `<Resource> Not Found` | No such id (or it is soft deleted) |
| 409 | `<Resource> Already Exist` | Unique constraint violation |
| 422 | `Invalid Reference` | A referenced row does not exist (junction resources only) |
| 500 | `Internal Server Error` | Unexpected failure |

## Conventions

Every resource exposes the same four operations:

| Method | Path | Description | Transaction |
| --- | --- | --- | --- |
| `POST` | `/{resource}` | Create | yes |
| `GET` | `/{resource}` | List with filters, paging, ordering | no |
| `GET` | `/{resource}/{id}` | Fetch one | no |
| `DELETE` | `/{resource}/{id}` | Delete | yes |

`POST` and `DELETE` run inside a database transaction that commits when the handler
succeeds and rolls back otherwise.

`DELETE` is a soft delete (`deleted_at` is stamped and the row disappears from every
read) for all resources **except** `hospital-staff`, whose table has no `deleted_at`
column and is therefore removed for real. Deleting an unknown id returns 404, not 200.

There are no update endpoints yet. `Update` exists at the store layer but is not
exposed over HTTP.

### Paging

| Param | Type | Default | Rules |
| --- | --- | --- | --- |
| `page` | int | `1` | must be > 0 |
| `limit` | int | `10` | must be > 0 and <= 100 |

Out-of-range values return 400. `meta.total` is the number of rows **on the current
page**, and `meta.total_pages` is not computed — there is no count query yet.

### Ordering

`order_by=<field>[,<direction>]`, where direction is `ASC` (default) or `DESC`.
An unknown field or direction returns 400. Allowed fields per resource:

| Resource | Fields | Default |
| --- | --- | --- |
| `user` | `id`, `created_at` | `id,ASC` |
| `staff` | `id`, `created_at` | `id,ASC` |
| `patient` | `id`, `created_at` | `id,ASC` |
| `hospital` | `id`, `code`, `created_at` | `id,ASC` |
| `hospital-patient` | `id`, `registered_at`, `created_at` | `id,ASC` |
| `hospital-staff` | `id`, `effective_from`, `created_at` | `id,ASC` |

### Formats

- Timestamps (`created_at`, `updated_at`, `registered_at`): RFC3339 — `2024-01-02T15:04:05Z`
- Dates (`date_of_birth`, `effective_from`, `effective_to`): `YYYY-MM-DD`
- Path ids must be integers greater than 0

---

## User

`/api/v1/user`

### Create — `POST /user`

| Field | Type | Required | Rules |
| --- | --- | --- | --- |
| `username` | string | yes | 3–255 chars |
| `password` | string | yes | min 6 chars |

```json
{ "username": "gopher", "password": "secret1" }
```

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

Example: `GET /user?page=2&limit=20&order_by=created_at,DESC`

### Errors

| Code | Cause |
| --- | --- |
| 400 | Validation failure, bad id |
| 404 | Unknown user |
| 409 | Duplicate `username` — see [Known gaps](#known-gaps) |

---

## Staff

`/api/v1/staff`

### Create — `POST /staff`

| Field | Type | Required | Rules |
| --- | --- | --- | --- |
| `user_id` | int | yes | > 0, must reference an existing user |
| `employee_code` | string | yes | max 32 |
| `first_name` | string | yes | max 32 |
| `last_name` | string | yes | max 32 |
| `email` | string | no | valid email, max 255 |
| `license_no` | string | no | max 64 |

```json
{
  "user_id": 1,
  "employee_code": "EMP-001",
  "first_name": "Somchai",
  "last_name": "Jaidee",
  "email": "somchai@hospital.co.th",
  "license_no": "MD-12345"
}
```

`is_active` is set to `true` on create. Omitted `email` and `license_no` are stored as
`NULL`, so blank values never collide with the unique index on `lower(email)`.

### Response object

```json
{
  "id": 42,
  "user_id": 1,
  "employee_code": "EMP-001",
  "first_name": "Somchai",
  "last_name": "Jaidee",
  "email": "somchai@hospital.co.th",
  "license_no": "MD-12345",
  "is_active": true,
  "created_at": "2024-01-02T15:04:05Z",
  "updated_at": "2024-01-02T15:04:05Z"
}
```

`email` and `license_no` are omitted when empty.

### List — `GET /staff`

| Filter | Type | Matches |
| --- | --- | --- |
| `id` | int | exact |
| `user_id` | int | exact |

### Errors

| Code | Cause |
| --- | --- |
| 400 | Validation failure, bad id |
| 404 | Unknown staff |
| 409 | Duplicate `employee_code` or `email` |

---

## Patient

`/api/v1/patient`

### Create — `POST /patient`

| Field | Type | Required | Rules |
| --- | --- | --- | --- |
| `national_id` | string | no | exactly 13 digits |
| `passport_no` | string | no | max 32 |
| `first_name_th` | string | yes | max 100 |
| `middle_name_th` | string | no | max 100 |
| `last_name_th` | string | yes | max 100 |
| `first_name_en` | string | no | max 100 |
| `middle_name_en` | string | no | max 100 |
| `last_name_en` | string | no | max 100 |
| `date_of_birth` | string | no | `YYYY-MM-DD` |
| `gender` | string | no | `M` or `F` |
| `phone` | string | no | max 32 |

```json
{
  "national_id": "1234567890123",
  "first_name_th": "สมชาย",
  "last_name_th": "ใจดี",
  "date_of_birth": "1990-05-04",
  "gender": "M",
  "phone": "0812345678"
}
```

### Response object

```json
{
  "id": 42,
  "national_id": "1234567890123",
  "first_name_th": "สมชาย",
  "last_name_th": "ใจดี",
  "date_of_birth": "1990-05-04",
  "gender": "M",
  "phone": "0812345678",
  "created_at": "2024-01-02T15:04:05Z",
  "updated_at": "2024-01-02T15:04:05Z"
}
```

Every optional field is omitted when empty.

### List — `GET /patient`

| Filter | Type | Matches |
| --- | --- | --- |
| `id` | int | exact |
| `national_id` | string | exact |
| `passport_no` | string | exact |
| `phone` | string | exact |

Example: `GET /patient?national_id=1234567890123`

### Errors

| Code | Cause |
| --- | --- |
| 400 | Validation failure, bad date format, bad id |
| 404 | Unknown patient |
| 409 | Duplicate `national_id` |

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

## Hospital patient

Registration of a patient at a hospital, carrying the hospital number (HN).

`/api/v1/hospital-patient`

### Create — `POST /hospital-patient`

| Field | Type | Required | Rules |
| --- | --- | --- | --- |
| `hospital_id` | int | yes | > 0, must reference an existing hospital |
| `patient_id` | int | yes | > 0, must reference an existing patient |
| `hn` | string | yes | max 32, unique per hospital |
| `status` | string | no | `active` (default) or `inactive` |
| `registered_at` | string | no | RFC3339, defaults to now |

```json
{ "hospital_id": 1, "patient_id": 2, "hn": "HN-0001" }
```

### Response object

```json
{
  "id": 42,
  "hospital_id": 1,
  "patient_id": 2,
  "hn": "HN-0001",
  "status": "active",
  "registered_at": "2024-01-02T15:04:05Z",
  "created_at": "2024-01-02T15:04:05Z",
  "updated_at": "2024-01-02T15:04:05Z"
}
```

### List — `GET /hospital-patient`

| Filter | Type | Matches |
| --- | --- | --- |
| `id` | int | exact |
| `hospital_id` | int | exact |
| `patient_id` | int | exact |
| `hn` | string | exact |
| `status` | string | exact |

Example: `GET /hospital-patient?hospital_id=1&status=active&order_by=registered_at,DESC`

### Errors

| Code | Cause |
| --- | --- |
| 400 | Validation failure, bad `registered_at` format, bad id |
| 404 | Unknown registration |
| 409 | `hn` already used at that hospital, or that patient is already registered there |
| 422 | `hospital_id` or `patient_id` does not exist |

---

## Hospital staff

Assignment of a staff member to a hospital. An assignment with no `effective_to` is
still active; setting `effective_to` closes it.

`/api/v1/hospital-staff`

### Create — `POST /hospital-staff`

| Field | Type | Required | Rules |
| --- | --- | --- | --- |
| `hospital_id` | int | yes | > 0, must reference an existing hospital |
| `staff_id` | int | yes | > 0, must reference an existing staff |
| `role` | string | no | `doctor`, `nurse`, `registrar` or `admin` |
| `is_primary` | bool | no | defaults to `false`, at most one primary per staff |
| `effective_from` | string | no | `YYYY-MM-DD`, defaults to today |

```json
{
  "hospital_id": 1,
  "staff_id": 2,
  "role": "doctor",
  "is_primary": true,
  "effective_from": "2024-01-02"
}
```

### Response object

```json
{
  "id": 42,
  "hospital_id": 1,
  "staff_id": 2,
  "role": "doctor",
  "is_primary": true,
  "effective_from": "2024-01-02",
  "effective_to": "2024-06-30",
  "created_at": "2024-01-02T15:04:05Z",
  "updated_at": "2024-01-02T15:04:05Z"
}
```

`effective_to` is omitted while the assignment is open.

### List — `GET /hospital-staff`

| Filter | Type | Matches |
| --- | --- | --- |
| `id` | int | exact |
| `hospital_id` | int | exact |
| `staff_id` | int | exact |
| `role` | string | exact |
| `is_primary` | bool | exact |
| `active` | bool | `true` → `effective_to IS NULL`, `false` → closed assignments |

Example: `GET /hospital-staff?staff_id=2&active=true&order_by=effective_from,DESC`

### Delete — `DELETE /hospital-staff/{id}`

A hard delete: the `hospital_staffs` table has no `deleted_at` column. To end an
assignment without losing history, set `effective_to` instead (not yet exposed over
HTTP).

### Errors

| Code | Cause |
| --- | --- |
| 400 | Validation failure, unknown `role`, bad date format, bad id |
| 404 | Unknown assignment |
| 409 | Staff already assigned to that hospital, or already has a primary hospital |
| 422 | `hospital_id` or `staff_id` does not exist |

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

List with paging:

```bash
curl 'http://localhost:8000/api/v1/patient?page=1&limit=2&order_by=created_at,DESC'
```

```json
{
  "success": true,
  "data": [
    { "id": 2, "first_name_th": "สมศรี", "last_name_th": "ดี", "created_at": "2024-01-03T09:00:00Z", "updated_at": "2024-01-03T09:00:00Z" },
    { "id": 1, "first_name_th": "สมชาย", "last_name_th": "ใจดี", "created_at": "2024-01-02T15:04:05Z", "updated_at": "2024-01-02T15:04:05Z" }
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

Unknown reference:

```bash
curl -X POST http://localhost:8000/api/v1/hospital-staff \
  -H 'Content-Type: application/json' \
  -d '{"hospital_id":999,"staff_id":999}'
```

```json
{
  "success": false,
  "error": {
    "code": "Invalid Reference",
    "message": "error hospital or staff does not exist"
  }
}
```

Delete:

```bash
curl -X DELETE http://localhost:8000/api/v1/patient/1
```

```json
{ "success": true }
```

---

## Known gaps

These are current behaviours to be aware of, not planned contract changes:

1. **Duplicate username returns 500, not 409.** `userdb.Create` does not translate the
   postgres unique violation into `ErrDBDuplicatedEntry` the way the other stores do,
   so the handler cannot recognise it. Every other resource returns 409 correctly.
2. **`meta.total` counts the current page only** and `meta.total_pages` is always
   omitted; there is no `COUNT(*)` query behind the list endpoints.
3. **Create returns 200, not 201**, and sets no `Location` header.
4. **No update endpoints.** `Update` is implemented in every store but not routed.
5. **Ids come from explicit `nextval()` calls** in the insert statements, because the
   migration creates sequences `OWNED BY` the id columns without setting
   `DEFAULT nextval(...)`.
