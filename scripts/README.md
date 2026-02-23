# SuwaCare LK – Database Scripts

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
```

Or one shot:

```bash
psql -d ncvms -f scripts/00_schema.sql -f scripts/01_indexes.sql -f scripts/02_seed.sql
```

## Scripts

| Script           | Purpose                          |
|------------------|----------------------------------|
| `00_schema.sql`  | Tables, constraints, triggers   |
| `01_indexes.sql` | Indexes for queries and filters |
| `02_seed.sql`    | Seed vaccines (optional)        |

## Connection

Application expects:

- `DATABASE_URL` or separate `PGHOST`, `PGPORT`, `PGUSER`, `PGPASSWORD`, `PGDATABASE`
- Default database name: `ncvms`
