package store

import (
    "context"
    "sync"
    "testing"

    "goflow/internal/job"
)

func TestMemoryStoreClaimsPendingJobsOnceConcurrently(t *testing.T) {
    ctx := context.Background()
    const jobCount = 20
    s := NewMemoryStore()

    for i := 0; i < jobCount; i++ {
        j, err := job.NewJob("compute", map[string]any{"value": i})
        if err != nil {
            t.Fatal(err)
        }
        if _, err := s.Create(ctx, j); err != nil {
            t.Fatal(err)
        }
    }

    claimedIDs := make(chan string, jobCount)
    errs := make(chan error, jobCount)
    var wg sync.WaitGroup
    for i := 0; i < jobCount; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            j, claimed, err := s.ClaimPending(ctx)
            if err != nil {
                errs <- err
                return
            }
            if claimed {
                claimedIDs <- j.ID
            }
        }()
    }
    wg.Wait()
    close(claimedIDs)
    close(errs)

    for err := range errs {
        t.Fatal(err)
    }

    seen := make(map[string]bool, jobCount)
    for id := range claimedIDs {
        if seen[id] {
            t.Fatalf("job %s was claimed more than once", id)
        }
        seen[id] = true
    }
    if len(seen) != jobCount {
        t.Fatalf("claimed %d jobs; want %d", len(seen), jobCount)
    }
}
