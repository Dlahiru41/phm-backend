# SuwaCare LK – NCVMS Backend

National Child Vaccination Management System (NCVMS) API – Go backend and PostgreSQL schema.

## Database setup

1. Create database:
   ```bash
   createdb ncvms
   ```

2. Run scripts in order:
   ```bash
   psql -d ncvms -f scripts/00_schema.sql
   psql -d ncvms -f scripts/01_indexes.sql
   psql -d ncvms -f scripts/02_seed.sql
   psql -d ncvms -f scripts/03_parent_child_linking_schema.sql
   psql -d ncvms -f scripts/04_parent_child_linking_indexes.sql
   psql -d ncvms -f scripts/06_mobile_change_otp_schema.sql
   psql -d ncvms -f scripts/07_mobile_change_otp_indexes.sql
   psql -d ncvms -f scripts/12_clinic_scheduling.sql
   psql -d ncvms -f scripts/13_clinic_notification_type.sql
   psql -d ncvms -f scripts/14_clinic_vaccination_tracking_enhancements.sql
   ```

See `scripts/README.md` for details.

## Application setup

1. Copy env example and set your values:
   ```bash
   copy .env.example .env
   ```
   Set at least `DATABASE_URL` and `JWT_SECRET`.

2. Install dependencies and run:
   ```bash
   go mod tidy
   go run ./cmd/api
   ```

   API listens on `http://localhost:8080` (or `PORT` from env).

Optional OTP tuning env vars:

- `MOBILE_CHANGE_OTP_TTL_MIN` (default `5`)
- `MOBILE_CHANGE_OTP_COOLDOWN_SEC` (default `60`)
- `MOBILE_CHANGE_OTP_MAX_ATTEMPTS` (default `5`)

## API base URL

All endpoints are under **`/api/v1`**.

- **Auth:** Bearer JWT in `Authorization` header (except login, register, forgot/reset password).
- **Roles:** `parent` | `phm` | `moh`

See the API specification document for full endpoint list (Authentication, Users, Children, Vaccines, Vaccination Records, Schedules, Growth Records, Notifications, Reports, Audit Logs, Analytics).

## Project layout

- `cmd/api` – main entry point
- `internal/auth` – JWT issue/parse
- `internal/config` – config load
- `internal/db` – PostgreSQL pool
- `internal/handlers` – HTTP handlers
- `internal/middleware` – auth and role middleware
- `internal/models` – request/response models
- `internal/response` – JSON error/success helpers
- `internal/router` – route registration
- `internal/store` – database access
- `scripts/` – SQL schema, indexes, seed
