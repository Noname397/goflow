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

To run the API locally:

```bash
go run ./cmd/goflow
```

The application listens on `:8080` by default.
