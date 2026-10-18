package worker

import (
    "context"
    "testing"
    "time"

    "goflow/internal/job"
)

func TestPoolRespectsWorkerLimit(t *testing.T) {
    jobs := make(chan job.Job, 20)
    workerCount := 4
    pool := NewPool(workerCount, jobs, func(ctx context.Context, j job.Job) (job.Job, error) {
        time.Sleep(20 * time.Millisecond)
        return j, nil
    })

    pool.Start(context.Background())
    defer pool.Stop()

    for i := 0; i < 20; i++ {
        j, err := job.NewJob("sleep", map[string]any{"duration_ms": 1})
        if err != nil {
            t.Fatal(err)
        }
        jobs <- j
    }

    close(jobs)
    time.Sleep(150 * time.Millisecond)
    if pool.MaxActive() > int64(workerCount) {
        t.Fatalf("worker limit exceeded: max active=%d workers=%d", pool.MaxActive(), workerCount)
    }
}
