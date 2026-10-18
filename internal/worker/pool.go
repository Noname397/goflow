package worker

import (
    "context"
    "fmt"
    "sync"
    "sync/atomic"

    "goflow/internal/job"
)

// Handler executes a job and returns the final job state.
type Handler func(context.Context, job.Job) (job.Job, error)

// Pool is a bounded worker pool for executing jobs concurrently.
type Pool struct {
    workers int
    jobs    chan job.Job
    handler Handler

    ctx    context.Context
    cancel context.CancelFunc
    wg     sync.WaitGroup

    currentActive int64
    maxActive     int64
}

func NewPool(workerCount int, jobs chan job.Job, handler Handler) *Pool {
    if workerCount <= 0 {
        workerCount = 1
    }
    if jobs == nil {
        jobs = make(chan job.Job, workerCount)
    }
    if handler == nil {
        handler = func(ctx context.Context, j job.Job) (job.Job, error) {
            return job.Execute(ctx, j)
        }
    }
    return &Pool{
        workers: workerCount,
        jobs:    jobs,
        handler: handler,
    }
}

func (p *Pool) Start(ctx context.Context) {
    if p == nil {
        return
    }
    p.ctx, p.cancel = context.WithCancel(ctx)
    for i := 0; i < p.workers; i++ {
        p.wg.Add(1)
        go p.worker(i)
    }
}

func (p *Pool) worker(id int) {
    defer p.wg.Done()
    for {
        select {
        case <-p.ctx.Done():
            return
        case j, ok := <-p.jobs:
            if !ok {
                return
            }

            active := atomic.AddInt64(&p.currentActive, 1)
            for {
                currentMax := atomic.LoadInt64(&p.maxActive)
                if active <= currentMax || atomic.CompareAndSwapInt64(&p.maxActive, currentMax, active) {
                    break
                }
            }

            if p.handler != nil {
                _, err := p.handler(p.ctx, j)
                if err != nil {
                    _ = err
                }
            }
            atomic.AddInt64(&p.currentActive, -1)
        }
    }
}

func (p *Pool) Stop() {
    if p == nil {
        return
    }
    if p.cancel != nil {
        p.cancel()
    }
    p.wg.Wait()
}

func (p *Pool) Enqueue(j job.Job) {
    if p == nil {
        return
    }
    p.jobs <- j
}

func (p *Pool) MaxActive() int64 {
    if p == nil {
        return 0
    }
    return atomic.LoadInt64(&p.maxActive)
}

func (p *Pool) Len() int {
    if p == nil {
        return 0
    }
    return len(p.jobs)
}

func BuildDefaultPool(workerCount int) *Pool {
    jobs := make(chan job.Job, workerCount)
    return NewPool(workerCount, jobs, nil)
}

func (p *Pool) String() string {
    if p == nil {
        return "<nil>"
    }
    return fmt.Sprintf("worker.Pool{workers=%d, queue=%d}", p.workers, len(p.jobs))
}
