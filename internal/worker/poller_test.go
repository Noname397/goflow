package worker

import (
	"context"
	"testing"
	"time"

	"goflow/internal/job"
	"goflow/internal/store"
)

func TestRunPollingProcessesClaimedJob(t *testing.T) {
	service := job.NewService(store.NewMemoryStore())
	created, err := service.Submit(context.Background(), "compute", map[string]any{"value": 10})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		RunPolling(ctx, 2, time.Millisecond, service.ClaimPending, func(ctx context.Context, claimed job.Job) (job.Job, error) {
			completed, err := job.Execute(ctx, claimed)
			if err != nil {
				return service.MarkFailed(ctx, claimed, err.Error())
			}
			completed, err = service.MarkSucceeded(ctx, completed, completed.Result)
			if err == nil {
				cancel()
			}
			return completed, err
		})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("polling worker did not stop after processing the job")
	}

	got, err := service.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != job.StatusSuccess {
		t.Fatalf("job status = %q; want %q", got.Status, job.StatusSuccess)
	}
}
