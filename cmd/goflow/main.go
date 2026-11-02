package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"goflow/internal/api"
	"goflow/internal/job"
	"goflow/internal/store"
	"goflow/internal/worker"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
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

	pollingDone := make(chan struct{})
	go func() {
		defer close(pollingDone)
		worker.RunPolling(ctx, workerCount(), 500*time.Millisecond, s.ClaimPending, func(ctx context.Context, j job.Job) (job.Job, error) {
			nextJob, err := job.Execute(ctx, j)
			if err != nil {
				return s.MarkFailed(ctx, j, err.Error())
			}
			return s.MarkSucceeded(ctx, nextJob, nextJob.Result)
		})
	}()

	httpHandler := api.NewHandler(s)
	server := &http.Server{
		Addr:         ":8080",
		Handler:      httpHandler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.ListenAndServe()
	}()

	fmt.Println("GoFlow running on :8080")
	select {
	case <-ctx.Done():
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			log.Printf("http server error: %v", err)
		}
		stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
	<-pollingDone
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
