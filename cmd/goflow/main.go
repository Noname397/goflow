package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"

	"goflow/internal/api"
	"goflow/internal/job"
	"goflow/internal/store"
	"goflow/internal/worker"
)

func main() {
	ctx := context.Background()
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("load .env: %v", err)
	}
	dbDSN := os.Getenv("GOFLOW_DSN")

	var s *job.Service
	if dbDSN != "" {
		pg, err := store.OpenPostgres(ctx, dbDSN)
		if err != nil {
			log.Fatalf("connect to postgres: %v", err)
		}
		defer pg.Close()
		s = job.NewService(pg)
	} else {
		s = job.NewService(store.NewMemoryStore())
	}

	jobs := make(chan job.Job, 32)
	runner := worker.NewPool(workerCount(), jobs, func(ctx context.Context, j job.Job) (job.Job, error) {
		queued, err := s.MarkRunning(ctx, j)
		if err != nil {
			return j, err
		}
		nextJob, err := job.Execute(ctx, queued)
		if err != nil {
			return s.MarkFailed(ctx, queued, err.Error())
		}
		return s.MarkSucceeded(ctx, nextJob, nextJob.Result)
	})
	runner.Start(ctx)
	defer runner.Stop()

	httpHandler := api.NewHandler(s, func(ctx context.Context, j job.Job) (job.Job, error) {
		jobs <- j
		return j, nil
	})
	server := &http.Server{
		Addr:         ":8080",
		Handler:      httpHandler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("http server error: %v", err)
		}
	}()

	fmt.Println("GoFlow running on :8080")
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}

func workerCount() int {
	value := os.Getenv("GOFLOW_WORKERS")
	if value == "" {
		return 4
	}
	count, err := strconv.Atoi(value)
	if err != nil || count <= 0 {
		return 4
	}
	return count
}
