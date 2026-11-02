package worker

import (
	"context"
	"log"
	"sync"
	"time"

	"goflow/internal/job"
)

type ClaimFunc func(context.Context) (job.Job, bool, error)

// RunPolling starts workers that claim jobs from a shared store until ctx ends.
func RunPolling(ctx context.Context, workerCount int, pollInterval time.Duration, claim ClaimFunc, handler Handler) {
	if workerCount <= 0 {
		workerCount = 1
	}
	if pollInterval <= 0 {
		pollInterval = 500 * time.Millisecond
	}
	if claim == nil || handler == nil {
		log.Printf("worker polling requires claim and handler functions")
		return
	}

	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			runPollingWorker(ctx, id, pollInterval, claim, handler)
		}(i)
	}
	wg.Wait()
}

func runPollingWorker(ctx context.Context, id int, pollInterval time.Duration, claim ClaimFunc, handler Handler) {
	for {
		if ctx.Err() != nil {
			return
		}

		j, claimed, err := claim(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("worker %d failed to claim job: %v", id, err)
			if !waitForPoll(ctx, pollInterval) {
				return
			}
			continue
		}
		if !claimed {
			if !waitForPoll(ctx, pollInterval) {
				return
			}
			continue
		}
		if _, err := handler(ctx, j); err != nil {
			log.Printf("worker %d failed to process job %s: %v", id, j.ID, err)
		}
	}
}

func waitForPoll(ctx context.Context, interval time.Duration) bool {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
