package job

import (
    "context"
    "errors"
    "fmt"
    "math"
    "math/rand"
    "time"
)

// JobStore persists job records and exposes a minimal lifecycle API.
type JobStore interface {
    Create(ctx context.Context, job Job) (Job, error)
    GetByID(ctx context.Context, id string) (Job, error)
    List(ctx context.Context, status ...JobStatus) ([]Job, error)
    UpdateStatus(ctx context.Context, job Job) (Job, error)
    ClaimPending(ctx context.Context) (Job, bool, error)
    RecoverPending(ctx context.Context) ([]Job, error)
}

// Service manages job validation, execution, and persistence.
type Service struct {
    store JobStore
}

func NewService(store JobStore) *Service {
    return &Service{store: store}
}

func (s *Service) Submit(ctx context.Context, jobType string, payload map[string]any) (Job, error) {
    if s == nil {
        return Job{}, errors.New("job service is nil")
    }
    newJob, err := NewJob(jobType, payload)
    if err != nil {
        return Job{}, err
    }
    if s.store == nil {
        return newJob, nil
    }
    return s.store.Create(ctx, newJob)
}

func (s *Service) GetByID(ctx context.Context, id string) (Job, error) {
    if s == nil {
        return Job{}, errors.New("job service is nil")
    }
    if id == "" {
        return Job{}, errors.New("job id is required")
    }
    if s.store == nil {
        return Job{}, fmt.Errorf("job %s not found", id)
    }
    return s.store.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context, status ...JobStatus) ([]Job, error) {
    if s == nil {
        return nil, errors.New("job service is nil")
    }
    if s.store == nil {
        return nil, nil
    }
    return s.store.List(ctx, status...)
}

func (s *Service) RecoverPending(ctx context.Context) ([]Job, error) {
    if s == nil {
        return nil, errors.New("job service is nil")
    }
    if s.store == nil {
        return nil, nil
    }
    return s.store.RecoverPending(ctx)
}

func (s *Service) ClaimPending(ctx context.Context) (Job, bool, error) {
    if s == nil {
        return Job{}, false, errors.New("job service is nil")
    }
    if s.store == nil {
        return Job{}, false, errors.New("job store is nil")
    }
    return s.store.ClaimPending(ctx)
}

func (s *Service) MarkRunning(ctx context.Context, job Job) (Job, error) {
    if s == nil {
        return Job{}, errors.New("job service is nil")
    }
    if s.store == nil {
        now := time.Now().UTC()
        job.Status = StatusRunning
        job.StartedAt = &now
        return job, nil
    }
    job.Status = StatusRunning
    now := time.Now().UTC()
    job.StartedAt = &now
    return s.store.UpdateStatus(ctx, job)
}

func (s *Service) MarkSucceeded(ctx context.Context, job Job, result any) (Job, error) {
    if s == nil {
        return Job{}, errors.New("job service is nil")
    }
    job.Status = StatusSuccess
    job.Result = result
    now := time.Now().UTC()
    job.CompletedAt = &now
    if s.store == nil {
        return job, nil
    }
    return s.store.UpdateStatus(ctx, job)
}

func (s *Service) MarkFailed(ctx context.Context, job Job, errMsg string) (Job, error) {
    if s == nil {
        return Job{}, errors.New("job service is nil")
    }
    job.Status = StatusFailed
    job.ErrorMessage = errMsg
    now := time.Now().UTC()
    job.CompletedAt = &now
    if s.store == nil {
        return job, nil
    }
    return s.store.UpdateStatus(ctx, job)
}

// Execute handles the runtime work for each supported job type.
func Execute(ctx context.Context, job Job) (Job, error) {
    if job.Type == "" {
        return Job{}, errors.New("job type is required")
    }

    switch job.Type {
    case "sleep":
        return executeSleep(ctx, job)
    case "compute":
        return executeCompute(ctx, job)
    default:
        return Job{}, fmt.Errorf("unsupported job type %q", job.Type)
    }
}

func executeSleep(ctx context.Context, job Job) (Job, error) {
    raw, ok := job.Payload["duration_ms"]
    if !ok {
        return Job{}, fmt.Errorf("sleep jobs require duration_ms")
    }

    ms, err := toInt64(raw)
    if err != nil {
        return Job{}, fmt.Errorf("invalid duration_ms: %w", err)
    }
    if ms < 0 {
        return Job{}, fmt.Errorf("duration_ms must be >= 0")
    }

    if ms > 0 {
        timer := time.NewTimer(time.Duration(ms) * time.Millisecond)
        select {
        case <-ctx.Done():
            timer.Stop()
            return Job{}, ctx.Err()
        case <-timer.C:
        }
    }

    job.Status = StatusSuccess
    job.Result = map[string]any{"duration_ms": ms}
    return job, nil
}

func executeCompute(ctx context.Context, job Job) (Job, error) {
    if err := ctx.Err(); err != nil {
        return Job{}, err
    }

    raw, ok := job.Payload["value"]
    if !ok {
        raw, ok = job.Payload["n"]
    }
    if !ok {
        raw = 10
    }

    n, err := toInt64(raw)
    if err != nil {
        return Job{}, fmt.Errorf("invalid compute input: %w", err)
    }
    if n < 0 {
        return Job{}, fmt.Errorf("compute input must be >= 0")
    }
    if n > 40 {
        n = 40
    }

    fib := fibonacci(n)
    job.Status = StatusSuccess
    job.Result = map[string]any{"value": n, "fib": fib}
    return job, nil
}

func toInt64(v any) (int64, error) {
    switch value := v.(type) {
    case int:
        return int64(value), nil
    case int8:
        return int64(value), nil
    case int16:
        return int64(value), nil
    case int32:
        return int64(value), nil
    case int64:
        return value, nil
    case uint:
        return int64(value), nil
    case uint8:
        return int64(value), nil
    case uint16:
        return int64(value), nil
    case uint32:
        return int64(value), nil
    case uint64:
        if value > uint64(math.MaxInt64) {
            return 0, fmt.Errorf("value too large")
        }
        return int64(value), nil
    case float32:
        return int64(value), nil
    case float64:
        return int64(value), nil
    case jsonNumber:
        return value.Int64()
    case string:
        var x int64
        if _, err := fmt.Sscanf(value, "%d", &x); err != nil {
            return 0, err
        }
        return x, nil
    default:
        return 0, fmt.Errorf("unsupported numeric type %T", v)
    }
}

type jsonNumber interface {
    Int64() (int64, error)
}

func fibonacci(n int64) int64 {
    if n <= 1 {
        return n
    }
    prev, curr := int64(0), int64(1)
    for i := int64(2); i <= n; i++ {
        prev, curr = curr, prev+curr
    }
    return curr
}

func NewRandomJob() Job {
    r := rand.Intn(2)
    jobType := "sleep"
    if r == 1 {
        jobType = "compute"
    }
    payload := map[string]any{"duration_ms": rand.Intn(100)}
    if jobType == "compute" {
        payload = map[string]any{"value": rand.Intn(12)}
    }
    j, _ := NewJob(jobType, payload)
    return j
}
