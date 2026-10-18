package store

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "strings"
    "time"

    _ "github.com/lib/pq"
    "goflow/internal/job"
)

// PostgresStore persists jobs in PostgreSQL.
type PostgresStore struct {
    db *sql.DB
}

func OpenPostgres(ctx context.Context, dsn string) (*PostgresStore, error) {
    if strings.TrimSpace(dsn) == "" {
        return nil, fmt.Errorf("postgres DSN is required")
    }
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        return nil, err
    }
    if err := db.PingContext(ctx); err != nil {
        _ = db.Close()
        return nil, err
    }
    return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) Create(ctx context.Context, j job.Job) (job.Job, error) {
    if s == nil || s.db == nil {
        return j, nil
    }
    query := `INSERT INTO jobs (id, type, payload, status, result, error_message, created_at, started_at, completed_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
    _, err := s.db.ExecContext(ctx, query,
        j.ID, j.Type, payloadToJSON(j.Payload), j.Status, resultToJSON(j.Result), j.ErrorMessage, j.CreatedAt.UTC(), nilTime(j.StartedAt), nilTime(j.CompletedAt),
    )
    if err != nil {
        return job.Job{}, err
    }
    return j, nil
}

func (s *PostgresStore) GetByID(ctx context.Context, id string) (job.Job, error) {
    if s == nil || s.db == nil {
        return job.Job{}, fmt.Errorf("job %s not found", id)
    }
    query := `SELECT id, type, payload, status, result, error_message, created_at, started_at, completed_at FROM jobs WHERE id = $1`
    row := s.db.QueryRowContext(ctx, query, id)
    var j job.Job
    var payload []byte
    var result []byte
    var createdAt, startedAt, completedAt sql.NullTime
    if err := row.Scan(&j.ID, &j.Type, &payload, &j.Status, &result, &j.ErrorMessage, &createdAt, &startedAt, &completedAt); err != nil {
        return job.Job{}, err
    }
    j.Payload = parsePayload(payload)
    j.Result = parseResult(result)
    if createdAt.Valid {
        j.CreatedAt = createdAt.Time
    }
    if startedAt.Valid {
        ts := startedAt.Time
        j.StartedAt = &ts
    }
    if completedAt.Valid {
        ts := completedAt.Time
        j.CompletedAt = &ts
    }
    return j, nil
}

func (s *PostgresStore) List(ctx context.Context, status ...job.JobStatus) ([]job.Job, error) {
    if s == nil || s.db == nil {
        return nil, nil
    }
    query := `SELECT id, type, payload, status, result, error_message, created_at, started_at, completed_at FROM jobs`
    if len(status) > 0 && status[0] != "" {
        query += ` WHERE status = $1`
    }
    query += ` ORDER BY created_at DESC`

    var rows *sql.Rows
    var err error
    if len(status) > 0 && status[0] != "" {
        rows, err = s.db.QueryContext(ctx, query, string(status[0]))
    } else {
        rows, err = s.db.QueryContext(ctx, query)
    }
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var jobs []job.Job
    for rows.Next() {
        var j job.Job
        var payload, result []byte
        var createdAt, startedAt, completedAt sql.NullTime
        if err := rows.Scan(&j.ID, &j.Type, &payload, &j.Status, &result, &j.ErrorMessage, &createdAt, &startedAt, &completedAt); err != nil {
            return nil, err
        }
        j.Payload = parsePayload(payload)
        j.Result = parseResult(result)
        if createdAt.Valid {
            j.CreatedAt = createdAt.Time
        }
        if startedAt.Valid {
            ts := startedAt.Time
            j.StartedAt = &ts
        }
        if completedAt.Valid {
            ts := completedAt.Time
            j.CompletedAt = &ts
        }
        jobs = append(jobs, j)
    }
    return jobs, rows.Err()
}

func (s *PostgresStore) UpdateStatus(ctx context.Context, j job.Job) (job.Job, error) {
    if s == nil || s.db == nil {
        return j, nil
    }
    query := `UPDATE jobs SET status = $1, result = $2, error_message = $3, started_at = $4, completed_at = $5 WHERE id = $6`
    _, err := s.db.ExecContext(ctx, query,
        j.Status,
        resultToJSON(j.Result),
        j.ErrorMessage,
        nilTime(j.StartedAt),
        nilTime(j.CompletedAt),
        j.ID,
    )
    if err != nil {
        return job.Job{}, err
    }
    return j, nil
}

func (s *PostgresStore) RecoverPending(ctx context.Context) ([]job.Job, error) {
    if s == nil || s.db == nil {
        return nil, nil
    }
    rows, err := s.db.QueryContext(ctx, `SELECT id, type, payload, status, result, error_message, created_at, started_at, completed_at FROM jobs WHERE status = 'pending' ORDER BY created_at DESC`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var all []job.Job
    for rows.Next() {
        var j job.Job
        var payload, result []byte
        var createdAt, startedAt, completedAt sql.NullTime
        if err := rows.Scan(&j.ID, &j.Type, &payload, &j.Status, &result, &j.ErrorMessage, &createdAt, &startedAt, &completedAt); err != nil {
            return nil, err
        }
        j.Payload = parsePayload(payload)
        j.Result = parseResult(result)
        if createdAt.Valid {
            j.CreatedAt = createdAt.Time
        }
        if startedAt.Valid {
            ts := startedAt.Time
            j.StartedAt = &ts
        }
        if completedAt.Valid {
            ts := completedAt.Time
            j.CompletedAt = &ts
        }
        all = append(all, j)
    }
    return all, rows.Err()
}

func (s *PostgresStore) Close() error {
    if s == nil || s.db == nil {
        return nil
    }
    return s.db.Close()
}

func payloadToJSON(v any) []byte {
    if v == nil {
        return []byte("{}")
    }
    b, _ := MarshalJSON(v)
    return b
}

func resultToJSON(v any) any {
    if v == nil {
        return nil
    }
    b, _ := MarshalJSON(v)
    return b
}

func parsePayload(raw []byte) map[string]any {
    if len(raw) == 0 {
        return map[string]any{}
    }
    var out map[string]any
    if err := json.Unmarshal(raw, &out); err != nil {
        return map[string]any{}
    }
    if out == nil {
        return map[string]any{}
    }
    return out
}

func parseResult(raw []byte) any {
    if len(raw) == 0 {
        return nil
    }
    var out any
    if err := json.Unmarshal(raw, &out); err != nil {
        return nil
    }
    return out
}

func nilTime(t *time.Time) any {
    if t == nil {
        return nil
    }
    return t.UTC()
}

func MarshalJSON(v any) ([]byte, error) {
    return json.Marshal(v)
}
