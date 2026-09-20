# 13. Step-by-Step Master Implementation Roadmap

This is your master execution blueprint. Follow these 9 sequential phases to build, verify, and master your production-ready Go Task Manager API from scratch.

---

## 🗺️ The 9 Implementation Phases

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Phase 1    │ ──> │   Phase 2    │ ──> │   Phase 3    │
│ Setup & Tool │     │ DB & Models  │     │ Security Pkg │
└──────────────┘     └──────────────┘     └──────────────┘
        │
        ▼
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Phase 4    │ ──> │   Phase 5    │ ──> │   Phase 6    │
│  Middleware  │     │  Full Auth   │     │ Worker Pools │
└──────────────┘     └──────────────┘     └──────────────┘
        │
        ▼
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Phase 7    │ ──> │   Phase 8    │ ──> │   Phase 9    │
│ Tasks+Redis  │     │ Write Tests  │     │ Prod & Docs  │
└──────────────┘     └──────────────┘     └──────────────┘
```

---

### Phase 1: Environment, Dependencies & Tooling

**Objective**: Establish a frictionless developer workflow with hot-reloading and containerized databases.

- [ ] **Install Dev CLI Tools**:
  ```bash
  go install github.com/air-verse/air@latest
  go install github.com/swaggo/swag/cmd/swag@latest
  ```
- [ ] **Install Core Dependencies**:
  ```bash
  go get github.com/gin-gonic/gin
  go get gorm.io/gorm
  go get gorm.io/driver/postgres
  go get github.com/redis/go-redis/v9
  go get github.com/golang-jwt/jwt/v5
  go get golang.org/x/crypto/bcrypt
  go get github.com/joho/godotenv
  go get github.com/google/uuid
  go get golang.org/x/time/rate
  go get github.com/stretchr/testify
  ```
- [ ] **Launch Infrastructure with Docker Compose**:
  ```bash
  make docker-up
  ```
  *Verify*:
  - PostgreSQL running on port `5432`
  - Adminer accessible at `http://localhost:8081`
  - Redis running on port `6379`
  - Redis Commander accessible at `http://localhost:8082`
- [ ] **Configure Environment**:
  - Copy `.env.example` to `.env`.
  - Add your `MAILTRAP_API_TOKEN` and a generated `JWT_SECRET` (`openssl rand -base64 32`).
- [ ] **Implement Typed Config**:
  - Code `internal/config/config.go` (refer to [Guide 04](./04-configuration-and-environment.md)).

---

### Phase 2: Database Layer & Domain Models

**Objective**: Define PostgreSQL tables with strict constraints, relations, and indexes.

- [ ] **Implement Models** in `internal/models/` (refer to [Guide 05](./05-database-design-and-models.md)):
  - `user.go`: `User` with `gorm.DeletedAt` and `json:"-"` on `PasswordHash`.
  - `task.go`: `Task`, `Category`, `TaskStatus` and `TaskPriority` enums.
  - `token.go`: `RefreshToken` and `PasswordResetToken`.
- [ ] **Database Connection & Auto-Migration**:
  - Implement `internal/database/postgres.go`.
  - Configure connection pool: `SetMaxOpenConns(25)`, `SetMaxIdleConns(10)`.
  - Call `db.AutoMigrate(...)`.
- [ ] **Verify in Adminer**:
  - Open `http://localhost:8081`, login with `postgres` / `postgres_secret_password` / DB `taskmanager_db`.
  - Check that tables, foreign keys, and indexes (`idx_user_status`, `idx_user_due`) were created correctly.

---

### Phase 3: Security & Utility Packages

**Objective**: Create reusable cryptographic and token utilities before touching business logic.

- [ ] **Password & Token Hashing** (`pkg/utils/hash.go`):
  - `HashPassword` and `CheckPasswordHash` with `bcrypt`.
  - `GenerateRandomHex` with `crypto/rand`.
  - `HashToken` with `crypto/sha256`.
- [ ] **JWT Handling** (`pkg/utils/jwt.go`):
  - `GenerateToken` and `ValidateToken` using `golang-jwt/jwt/v5`.
- [ ] **Write Utility Unit Tests**:
  - Create `pkg/utils/hash_test.go` and `pkg/utils/jwt_test.go` to verify hashing and token expiration.
  - Run: `go test -v ./pkg/utils/...`

---

### Phase 4: Production Middleware & Observability

**Objective**: Equip the API with structured logging, distributed tracing, and crash prevention.

- [ ] **Implement Middleware** in `internal/middleware/` (refer to [Guide 06](./06-production-middleware-and-observability.md)):
  - `logger.go`: Structured JSON logging using `log/slog`.
  - `request_id.go`: Attaches `X-Request-ID` UUID to every request and log entry.
  - `recovery.go`: Catches panics and returns clean 500 JSON without leaking stack traces.
  - `cors.go`: Configurable CORS handler.
  - `ratelimit.go`: Per-IP token bucket rate limiting using `golang.org/x/time/rate`.

---

### Phase 5: Complete Authentication Subsystem

**Objective**: Build full user registration, login, token refresh, and password recovery.

- [ ] **Repositories**:
  - `internal/repository/user_repository.go`
  - `internal/repository/token_repository.go`
- [ ] **Email Service**:
  - `internal/service/email_service.go` using standard Go `net/smtp` (refer to [Guide 08](./08-complete-auth-implementation-guide.md)).
- [ ] **Auth Service**:
  - `internal/service/auth_service.go` (refer to [Guide 08](./08-complete-auth-implementation-guide.md)).
  - Implement `SignUp`, `Login`, `RefreshToken`, `ForgotPassword`, `ResetPassword`.
  - Wrap password reset in an ACID `db.Transaction` (updates password, consumes token, revokes sessions).
- [ ] **Auth Middleware**:
  - `internal/middleware/auth_middleware.go` (validates `Bearer <token>` and extracts `userID`).
- [ ] **Auth Handlers & Routing**:
  - `internal/handler/auth_handler.go`.
- [ ] **Interactive Verification**:
  - Open `requests.http` in your editor and test:
    1. `POST /auth/signup`
    2. `POST /auth/login` (grab JWT access token)
    3. `POST /auth/forgot-password` (check Mailtrap inbox for reset link!)
    4. `POST /auth/reset-password`

---

### Phase 6: Concurrency & Worker Pools

**Objective**: Make email sending and maintenance jobs non-blocking.

- [ ] **Email Worker Pool** (`internal/worker/email_worker.go` - refer to [Guide 11](./11-concurrency-goroutines-channels-and-workers.md)):
  - Create buffered job channel: `make(chan EmailJob, 100)`.
  - Spawn 3 background worker goroutines.
  - Make `ForgotPassword` push jobs to the channel in microseconds instead of awaiting SMTP.
- [ ] **Background Scheduler** (`internal/worker/scheduler.go`):
  - Setup `time.NewTicker(1 * time.Hour)` to periodically purge expired tokens from PostgreSQL.

---

### Phase 7: Task Domain & Redis Caching

**Objective**: Implement core task management with sub-millisecond Redis reads.

- [ ] **Task Repository**:
  - `internal/repository/task_repository.go` (PostgreSQL queries, pagination, dynamic status/priority filtering, ILIKE search).
- [ ] **Redis Connection**:
  - `internal/database/redis.go` (connect to Redis on port `6379`).
- [ ] **Cached Task Repository Decorator**:
  - `internal/repository/cached_task_repository.go` (refer to [Guide 10](./10-redis-caching.md)).
  - Cache hit on `GetByID` returns in `< 1ms`.
  - Create/Update/Delete invalidates stale keys in Redis.
- [ ] **Task Service & Handler**:
  - `internal/service/task_service.go` (validates user ownership, auto-sets `CompletedAt`).
  - `internal/handler/task_handler.go`.
- [ ] **Verify in Redis Commander**:
  - Open `http://localhost:8082` to watch keys appear and expire.

---

### Phase 8: Writing Automated Tests

**Objective**: Achieve production-grade test coverage (> 75%) with zero mocks for PostgreSQL.

- [ ] **Service Unit Tests with Mocks** (`internal/service/task_service_test.go` - refer to [Guide 12](./12-comprehensive-testing-guide.md)):
  - Use `testify/mock` to test business rules without touching the database.
- [ ] **HTTP Handler Tests** (`internal/handler/task_handler_test.go`):
  - Use `net/http/httptest` to test request binding, status codes, and error formatting in memory.
- [ ] **Database Integration Tests** (`internal/repository/task_repository_test.go`):
  - Use the **Transaction Rollback Pattern** (`tx := db.Begin()`, `defer tx.Rollback()`) to test real SQL queries without leaving test artifacts in PostgreSQL.
- [ ] **Run Full Test Suite**:
  ```bash
  make test
  make test-coverage
  ```

---

### Phase 9: Production Polish, Swagger & Graceful Shutdown

**Objective**: Prepare the application for real-world deployment.

- [ ] **Generate Interactive Swagger UI**:
  - Annotate `main.go` and handlers with swag doc-comments.
  - Run `make swagger`.
  - Visit `http://localhost:8080/swagger/index.html` in your browser.
- [ ] **Implement Graceful Shutdown in `cmd/api/main.go`**:
  - Listen for `SIGINT` / `SIGTERM`.
  - Stop HTTP server (`srv.Shutdown(ctx)` with 10s timeout).
  - Stop worker pool (`emailPool.Stop()`).
  - Close PostgreSQL and Redis connection pools.
- [ ] **Build Production Binary**:
  ```bash
  make build
  ```

---

## 🎯 Verification Checklist

Before considering your project complete, run through this final checklist:

| Verification Item | Command / Location | Expected Result |
|---|---|---|
| **Hot Reloading** | `make dev` | Air reloads instantly on file changes. |
| **All Containers Healthy** | `docker compose ps` | PostgreSQL and Redis status `healthy`. |
| **Lint Checks** | `make lint` | Zero lint errors from `golangci-lint`. |
| **Test Suite** | `make test` | All unit and integration tests pass with race detector enabled. |
| **Test Coverage** | `make test-coverage` | Visual HTML coverage report shows > 75% coverage. |
| **API Endpoints** | `requests.http` | All auth and task requests succeed with expected JSON envelopes. |
| **Graceful Shutdown** | Press `Ctrl+C` while `make run` | Server logs clean shutdown of HTTP, workers, and DB connections. |
