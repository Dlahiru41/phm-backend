# SuwaCare LK - Database Scripts

## Prerequisites

- PostgreSQL 12+ (recommended 14+)
- A database and user with `CREATE` privileges

## Setup

```bash
# Create database
createdb ncvms

# Or in psql:
# CREATE DATABASE ncvms;

# Run in order (from project root or scripts directory)
psql -d ncvms -f scripts/00_schema.sql
psql -d ncvms -f scripts/01_indexes.sql
psql -d ncvms -f scripts/02_seed.sql
psql -d ncvms -f scripts/03_parent_child_linking_schema.sql
psql -d ncvms -f scripts/04_parent_child_linking_indexes.sql
psql -d ncvms -f scripts/06_mobile_change_otp_schema.sql
psql -d ncvms -f scripts/07_mobile_change_otp_indexes.sql
```

Or one shot:

```bash
psql -d ncvms -f scripts/00_schema.sql -f scripts/01_indexes.sql -f scripts/02_seed.sql -f scripts/03_parent_child_linking_schema.sql -f scripts/04_parent_child_linking_indexes.sql -f scripts/06_mobile_change_otp_schema.sql -f scripts/07_mobile_change_otp_indexes.sql
```

## Scripts

| Script                                | Purpose                                  |
|---------------------------------------|------------------------------------------|
| `00_schema.sql`                       | Tables, constraints, triggers            |
| `01_indexes.sql`                      | Indexes for queries and filters          |
| `02_seed.sql`                         | Seed vaccines (optional)                 |
| `03_parent_child_linking_schema.sql`  | Add WhatsApp field and OTP table         |
| `04_parent_child_linking_indexes.sql` | Add indexes for OTP linking flow         |
| `06_mobile_change_otp_schema.sql`     | Add OTP table for mobile number changes  |
| `07_mobile_change_otp_indexes.sql`    | Add indexes for mobile change OTP flow   |

## Connection

Application expects:

- `DATABASE_URL` or separate `PGHOST`, `PGPORT`, `PGUSER`, `PGPASSWORD`, `PGDATABASE`
- Default database name: `ncvms`
