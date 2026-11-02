# GoFlow

GoFlow is a small distributed job execution engine built in Go. Workers poll the configured store for jobs, while the API accepts submissions and reports job state.

## Features

- Submit jobs through a small HTTP API
- Maintain job state in memory or in PostgreSQL
- Run configurable in-process workers that claim pending jobs from the shared store
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

Workers claim pending jobs from the store every 500 ms. PostgreSQL claims are atomic, so concurrent workers skip rows already claimed by another worker. API and workers still run in the same process. Jobs left in `running` after a worker crash are not recovered yet; lease-based recovery is a follow-up.
