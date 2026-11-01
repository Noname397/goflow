# GoFlow

GoFlow is a small distributed job execution engine built in Go. The initial version focuses on the core engine: a bounded worker pool, job lifecycle tracking, and a storage abstraction with a PostgreSQL migration.

## Features

- Submit jobs through a small HTTP API
- Maintain job state in memory or in PostgreSQL
- Run a fixed-size worker pool with graceful shutdown
- Support `sleep` and `compute` workloads
- Recover pending jobs on startup

## Quick start

```bash
go test ./...
```

To run with PostgreSQL locally, start the database and apply the migration:

```bash
docker compose up -d postgres
```

In PowerShell, create a local `.env` from the example:

```powershell
Copy-Item .env.example .env
Get-Content -Raw .\migrations\001_create_jobs.sql |
  docker compose exec -T postgres psql -U goflow -d goflow
```

Then start the API:

```powershell
go run ./cmd/goflow
```

The application loads `GOFLOW_DSN` from `.env` in the current directory. Existing process environment variables take precedence. The application listens on `:8080` by default. If `GOFLOW_DSN` is unset, it uses in-memory storage.
