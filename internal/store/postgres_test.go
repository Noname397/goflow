package store

import (
    "context"
    "testing"
	"time"

    "goflow/internal/job"
)

func TestMemoryStorePersistsJobs(t *testing.T) {
    store := NewMemoryStore()
    j, err := job.NewJob("sleep", map[string]any{"duration_ms": 10})
    if err != nil {
        t.Fatal(err)
    }
    if _, err = store.Create(context.Background(), j); err != nil {
        t.Fatal(err)
    }

    got, err := store.GetByID(context.Background(), j.ID)
    if err != nil {
        t.Fatal(err)
    }
    if got.Type != "sleep" {
        t.Fatalf("unexpected type: %s", got.Type)
    }

    now := time.Now().UTC()
    got.Status = job.StatusRunning
    got.StartedAt = &now
    if _, err = store.UpdateStatus(context.Background(), got); err != nil {
        t.Fatal(err)
    }

    listed, err := store.List(context.Background(), job.StatusRunning)
    if err != nil {
        t.Fatal(err)
    }
    if len(listed) != 1 {
        t.Fatalf("expected one running job, got %d", len(listed))
    }
}
