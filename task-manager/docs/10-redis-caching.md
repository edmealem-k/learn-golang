# 10. Redis In-Memory Caching

In high-throughput web APIs, querying PostgreSQL for every read request introduces unnecessary disk I/O and query latency. **Redis** is an in-memory key-value data store used to cache frequently accessed data with sub-millisecond response times.

---

## 1. The Cache-Aside Pattern

The industry-standard caching pattern for REST APIs is **Cache-Aside**:

```
                       GET /api/v1/tasks/:id
                                 │
                                 ▼
                     ┌───────────────────────┐
                     │ Check Redis Key Exists│
                     └───────────┬───────────┘
                                 │
                 Yes (Hit)       │       No (Miss)
           ┌─────────────────────┴─────────────────────┐
           ▼                                           ▼
 ┌───────────────────┐                       ┌───────────────────┐
 │ Return from Redis │                       │ Query PostgreSQL  │
 │   (< 1ms latency) │                       └─────────┬─────────┘
 └───────────────────┘                                 │
                                                       ▼
                                             ┌───────────────────┐
                                             │  Write to Redis   │
                                             │ (TTL: 15 minutes) │
                                             └───────────────────┘
```

### Key Principles:
1. **Read Path**: The application first checks Redis. If the key exists (Cache Hit), it returns the cached data immediately. If missing (Cache Miss), it fetches from PostgreSQL, stores the result in Redis with a Time-To-Live (TTL), and returns.
2. **Write Path (Cache Invalidation)**: When a task is created, updated, or deleted, the corresponding cache key in Redis must be **deleted (busted)** so subsequent requests never return stale data.

---

## 2. Redis Connection Client (`internal/database/redis.go`)

Install the official Go Redis client:
```bash
go get github.com/redis/go-redis/v9
```

Create `internal/database/redis.go`:

```go
package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"go-task-manager/internal/config"

	"github.com/redis/go-redis/v9"
)

func ConnectRedis(cfg *config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     20, // Connection pool size
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Println("Successfully connected to Redis")
	return client, nil
}
```

---

## 3. Cached Repository Decorator (`internal/repository/cached_task_repository.go`)

Using the **Decorator Pattern**, `CachedTaskRepository` wraps your existing `TaskRepository` without modifying its underlying SQL code:

```go
package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"go-task-manager/internal/models"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type CachedTaskRepository struct {
	postgresRepo TaskRepository
	redisClient  *redis.Client
	ttl          time.Duration
}

func NewCachedTaskRepository(postgresRepo TaskRepository, redisClient *redis.Client, ttl time.Duration) TaskRepository {
	return &CachedTaskRepository{
		postgresRepo: postgresRepo,
		redisClient:  redisClient,
		ttl:          ttl,
	}
}

// taskKey namespaces keys: "task:user:{userID}:id:{taskID}"
func (r *CachedTaskRepository) taskKey(userID, taskID uuid.UUID) string {
	return fmt.Sprintf("task:user:%s:id:%s", userID.String(), taskID.String())
}

func (r *CachedTaskRepository) GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*models.Task, error) {
	key := r.taskKey(userID, id)

	// 1. Check Redis (Cache Hit)
	val, err := r.redisClient.Get(ctx, key).Result()
	if err == nil {
		var task models.Task
		if jsonErr := json.Unmarshal([]byte(val), &task); jsonErr == nil {
			return &task, nil
		}
	}

	// 2. Cache Miss: Query PostgreSQL
	task, err := r.postgresRepo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	// 3. Populate cache asynchronously with TTL
	go func() {
		data, err := json.Marshal(task)
		if err == nil {
			_ = r.redisClient.Set(context.Background(), key, data, r.ttl).Err()
		}
	}()

	return task, nil
}

func (r *CachedTaskRepository) Create(ctx context.Context, task *models.Task) error {
	return r.postgresRepo.Create(ctx, task)
}

func (r *CachedTaskRepository) Update(ctx context.Context, task *models.Task) error {
	if err := r.postgresRepo.Update(ctx, task); err != nil {
		return err
	}

	// Invalidate stale cache
	key := r.taskKey(task.UserID, task.ID)
	_ = r.redisClient.Del(ctx, key).Err()
	return nil
}

func (r *CachedTaskRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	if err := r.postgresRepo.Delete(ctx, id, userID); err != nil {
		return err
	}

	// Invalidate stale cache
	key := r.taskKey(userID, id)
	_ = r.redisClient.Del(ctx, key).Err()
	return nil
}

func (r *CachedTaskRepository) List(ctx context.Context, filter TaskFilter) ([]models.Task, int64, int, error) {
	// Complex filtering and pagination always delegate to PostgreSQL
	return r.postgresRepo.List(ctx, filter)
}
```

---

## 4. Inspecting Cache in Redis Commander

With your Docker Compose stack running (`make docker-up`), open **`http://localhost:8082`** in your browser to:
- View all active cache keys.
- Inspect JSON payloads stored in Redis.
- Watch TTL countdowns in real time.
