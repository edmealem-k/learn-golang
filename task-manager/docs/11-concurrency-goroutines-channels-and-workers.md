# 11. Concurrency: Goroutines, Channels & Background Workers

Go is renowned for its lightweight concurrency model. A Go thread (called a **goroutine**) requires only ~2KB of initial stack memory (compared to ~1MB for traditional OS threads). You can run tens of thousands of goroutines concurrently on a modest server.

In this Task Manager project, concurrency is used for:
1. **Non-blocking Email Worker Pool** (using buffered channels and worker goroutines).
2. **Periodic Background Cleanup Cron** (using `time.Ticker`).
3. **Graceful Worker Shutdown** (using `sync.WaitGroup` and `context.Context`).

---

## 1. Concurrency Primitives Quick Reference

| Primitive | What It Is | Practical Use Case |
|---|---|---|
| `go func()` | Spawns a lightweight concurrent goroutine | Running tasks asynchronously without blocking the caller. |
| `chan T` | Channel for typed communication between goroutines | Passing jobs to workers without shared-memory race conditions. |
| `select` | Multiplexes channel operations (like `switch` for channels) | Waiting on multiple channels or handling timeouts/cancellations. |
| `sync.WaitGroup` | Counter that waits for a collection of goroutines to finish | Ensuring all background jobs finish before server shuts down. |
| `time.Ticker` | Sends timestamp ticks over a channel at regular intervals | Running recurring tasks (e.g. clean up expired tokens every hour). |

> [!TIP]
> **The Go Concurrency Motto:**
> *"Do not communicate by sharing memory; instead, share memory by communicating."* (Use channels rather than mutex locks whenever possible!)

---

## 2. Asynchronous Email Worker Pool

### The Problem:
Sending an email via SMTP takes 1–3 seconds. If done inside the HTTP handler, the user's browser spins waiting for the email to send before receiving a response.

### The Solution:
We create a **Worker Pool** using a Go buffered channel. The HTTP handler pushes an `EmailJob` to the channel in microseconds and immediately responds to the user. A fixed pool of worker goroutines pulls jobs from the channel and sends the emails in the background.

Create `internal/worker/email_worker.go`:

```go
package worker

import (
	"context"
	"log"
	"sync"

	"go-task-manager/internal/service"
)

type EmailJob struct {
	ToEmail string
	Token   string
}

type EmailWorkerPool struct {
	emailService service.EmailService
	jobQueue     chan EmailJob
	workerCount  int
	wg           sync.WaitGroup
}

func NewEmailWorkerPool(emailService service.EmailService, workerCount int, queueSize int) *EmailWorkerPool {
	return &EmailWorkerPool{
		emailService: emailService,
		jobQueue:     make(chan EmailJob, queueSize), // Buffered channel
		workerCount:  workerCount,
	}
}

// Start launches the worker goroutines
func (p *EmailWorkerPool) Start(ctx context.Context) {
	for i := 1; i <= p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}
	log.Printf("Started %d background email workers\n", p.workerCount)
}

func (p *EmailWorkerPool) worker(ctx context.Context, id int) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done(): // Context cancelled (server shutting down)
			log.Printf("Email worker %d stopping...\n", id)
			return

		case job, ok := <-p.jobQueue:
			if !ok {
				// Channel closed
				return
			}
			log.Printf("[Worker %d] Processing password reset email for %s\n", id, job.ToEmail)
			if err := p.emailService.SendPasswordResetEmail(job.ToEmail, job.Token); err != nil {
				log.Printf("[Worker %d] Error sending email to %s: %v\n", id, job.ToEmail, err)
			} else {
				log.Printf("[Worker %d] Successfully sent email to %s\n", id, job.ToEmail)
			}
		}
	}
}

// EnqueueJob non-blockingly queues an email job
func (p *EmailWorkerPool) EnqueueJob(job EmailJob) bool {
	select {
	case p.jobQueue <- job:
		return true
	default:
		// Queue is full! In production, log warning or push to persistent queue (RabbitMQ/Redis)
		log.Println("WARNING: Email job queue is full, dropping job")
		return false
	}
}

// Stop waits for all active jobs to complete before returning
func (p *EmailWorkerPool) Stop() {
	close(p.jobQueue)
	p.wg.Wait()
	log.Println("All email workers stopped cleanly")
}
```

---

## 3. Scheduled Background Jobs (`time.Ticker`)

Let's implement a cleaner that runs every hour in the background to delete expired tokens and archive overdue tasks.

Create `internal/worker/scheduler.go`:

```go
package worker

import (
	"context"
	"log"
	"time"

	"gorm.io/gorm"
)

type Scheduler struct {
	db *gorm.DB
}

func NewScheduler(db *gorm.DB) *Scheduler {
	return &Scheduler{db: db}
}

// StartTokenCleanup runs cleanup every interval until context is cancelled
func (s *Scheduler) StartTokenCleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()
		log.Printf("Scheduled token cleaner running every %v\n", interval)

		for {
			select {
			case <-ctx.Done():
				log.Println("Token cleaner stopped")
				return
			case t := <-ticker.C:
				s.cleanupExpiredTokens(t)
			}
		}
	}()
}

func (s *Scheduler) cleanupExpiredTokens(t time.Time) {
	log.Printf("[Scheduler] Running expired token cleanup at %s...\n", t.Format(time.RFC3339))

	// Delete expired password reset tokens
	resReset := s.db.Exec("DELETE FROM password_reset_tokens WHERE expires_at < ? OR used_at IS NOT NULL", time.Now())
	if resReset.Error != nil {
		log.Printf("[Scheduler] Error cleaning reset tokens: %v\n", resReset.Error)
	} else {
		log.Printf("[Scheduler] Cleaned up %d expired reset tokens\n", resReset.RowsAffected)
	}

	// Delete expired refresh tokens
	resRefresh := s.db.Exec("DELETE FROM refresh_tokens WHERE expires_at < ? OR revoked_at IS NOT NULL", time.Now())
	if resRefresh.Error != nil {
		log.Printf("[Scheduler] Error cleaning refresh tokens: %v\n", resRefresh.Error)
	} else {
		log.Printf("[Scheduler] Cleaned up %d expired refresh tokens\n", resRefresh.RowsAffected)
	}
}
```

---

## 4. Integrating Concurrency & Graceful Shutdown in `main.go`

Here is how to start workers and coordinate their graceful shutdown in `cmd/api/main.go`:

```go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-task-manager/internal/config"
	"go-task-manager/internal/database"
	"go-task-manager/internal/service"
	"go-task-manager/internal/worker"
)

func main() {
	cfg, _ := config.LoadConfig()
	db, _ := database.Connect(cfg)

	// 1. Create a root cancelable context for all background workers
	workerCtx, cancelWorkers := context.WithCancel(context.Background())
	defer cancelWorkers()

	// 2. Initialize Email Worker Pool (3 worker goroutines, buffer size 100)
	emailSvc := service.NewSMTPEmailService(cfg)
	emailPool := worker.NewEmailWorkerPool(emailSvc, 3, 100)
	emailPool.Start(workerCtx)

	// 3. Start Background Scheduler (Runs every 1 hour)
	scheduler := worker.NewScheduler(db)
	scheduler.StartTokenCleanup(workerCtx, 1*time.Hour)

	// 4. Start HTTP Server
	srv := &http.Server{Addr: ":" + cfg.Port}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v\n", err)
		}
	}()

	// 5. Wait for OS interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down gracefully...")

	// 6. Stop accepting new HTTP requests (10s timeout)
	httpCtx, cancelHTTP := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelHTTP()
	_ = srv.Shutdown(httpCtx)

	// 7. Stop background workers gracefully
	cancelWorkers()  // Signal scheduler and workers to finish
	emailPool.Stop() // Wait for active email jobs to finish

	log.Println("Everything exited cleanly. Goodbye!")
}
```

---

## 5. Summary of Advanced Techniques Learned

1. **Channels (`chan EmailJob`)**: Safe communication between the HTTP layer and background threads with zero locks.
2. **Worker Pool Pattern**: Limits resource consumption by fixing the number of active goroutines.
3. **`select` Multiplexing**: Non-blocking channel operations and timeout/cancellation handling.
4. **`time.Ticker`**: Idiomatic Go periodic background jobs.
5. **`sync.WaitGroup`**: Coordinated, leak-free shutdowns in microservices.

