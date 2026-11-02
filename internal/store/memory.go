package store

import (
    "context"
    "fmt"
    "sort"
    "sync"
    "time"

    "goflow/internal/job"
)

// MemoryStore is an in-memory implementation used for tests and local development.
type MemoryStore struct {
    mu   sync.RWMutex
    jobs map[string]job.Job
}

func NewMemoryStore() *MemoryStore {
    return &MemoryStore{jobs: make(map[string]job.Job)}
}

func (s *MemoryStore) Create(ctx context.Context, j job.Job) (job.Job, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    if _, ok := s.jobs[j.ID]; ok {
        return job.Job{}, fmt.Errorf("job %s already exists", j.ID)
    }
    s.jobs[j.ID] = j
    return j, nil
}

func (s *MemoryStore) GetByID(ctx context.Context, id string) (job.Job, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    j, ok := s.jobs[id]
    if !ok {
        return job.Job{}, fmt.Errorf("job %s not found", id)
    }
    return j, nil
}

func (s *MemoryStore) List(ctx context.Context, status ...job.JobStatus) ([]job.Job, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    var jobs []job.Job
    for _, j := range s.jobs {
        if len(status) == 0 || matchesStatus(j.Status, status) {
            jobs = append(jobs, j)
        }
    }
    sort.Slice(jobs, func(i, j int) bool {
        return jobs[i].CreatedAt.After(jobs[j].CreatedAt)
    })
    return jobs, nil
}

func (s *MemoryStore) UpdateStatus(ctx context.Context, j job.Job) (job.Job, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    if _, ok := s.jobs[j.ID]; !ok {
        return job.Job{}, fmt.Errorf("job %s not found", j.ID)
    }
    s.jobs[j.ID] = j
    return j, nil
}

func (s *MemoryStore) ClaimPending(ctx context.Context) (job.Job, bool, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    var candidate job.Job
    found := false
    for _, current := range s.jobs {
        if current.Status != job.StatusPending {
            continue
        }
        if !found || current.CreatedAt.Before(candidate.CreatedAt) {
            candidate = current
            found = true
        }
    }
    if !found {
        return job.Job{}, false, nil
    }

    now := time.Now().UTC()
    candidate.Status = job.StatusRunning
    candidate.StartedAt = &now
    s.jobs[candidate.ID] = candidate
    return candidate, true, nil
}

func (s *MemoryStore) RecoverPending(ctx context.Context) ([]job.Job, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    var pending []job.Job
    for _, j := range s.jobs {
        if j.Status == job.StatusPending {
            pending = append(pending, j)
        }
    }
    sort.Slice(pending, func(i, j int) bool {
        return pending[i].CreatedAt.After(pending[j].CreatedAt)
    })
    return pending, nil
}

func matchesStatus(current job.JobStatus, wanted []job.JobStatus) bool {
    for _, s := range wanted {
        if s == "" || current == s {
            return true
        }
    }
    return false
}
