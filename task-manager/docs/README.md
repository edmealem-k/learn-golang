# Production-Ready Go Task Manager API — Complete Master Guide

Welcome to the comprehensive implementation guide for building a production-grade **Task Manager RESTful API** in Go. This guide is organized in a logical, step-by-step order from foundational Go principles to production deployment.

---

## 📚 Table of Contents (Sequential Order)

1. [01. Go Core Concepts Explained for Web Developers](./01-go-core-concepts-for-web-devs.md)
   - Pointers vs. Values (`*` and `&`), why pointers are used in models/DTOs/methods.
   - Interfaces and implicit implementation ("duck typing") for decoupled architecture.
   - Go error handling philosophy, error wrapping (`%w`), and `errors.Is`.
   - `context.Context` mechanics (request cancellation propagation, DB timeouts).
   - Struct tags reflection mechanics (`json:`, `gorm:`, `binding:`).
   - Goroutines and concurrency safety in web servers.

2. [02. Architecture & Project Layout](./02-architecture-and-project-structure.md)
   - Clean / Layered Architecture principles (Handler $\rightarrow$ Service $\rightarrow$ Repository $\rightarrow$ Database).
   - Standard Go Project Layout (`cmd/`, `internal/`, `pkg/`).
   - Compiler-enforced privacy of the `internal/` directory.
   - Dependency Injection without heavy frameworks.

3. [03. Developer Tooling, Hot-Reloading & Productivity](./03-developer-tooling-and-setup.md)
   - Live auto-reloading server setup with **Air** (`.air.toml`).
   - Standard production **Makefile** (`make dev`, `make test`, `make build`, `make docker-up`).
   - Static analysis and code quality checks with **golangci-lint** (`.golangci.yml`).
   - Interactive API documentation using **Swagger / OpenAPI** (`swaggo`).
   - Interactive step-through debugging with **Delve** (`dlv`).

4. [04. Configuration & Environment Setup](./04-configuration-and-environment.md)
   - 12-Factor application principles.
   - Typed `Config` struct with fail-fast startup validation.
   - Loading environment variables with `godotenv`.
   - Local development with Docker Compose (PostgreSQL, Redis, Adminer, Redis Commander).

5. [05. Database Design & PostgreSQL Models](./05-database-design-and-models.md)
   - Relational schema design (Users, Tasks, Categories, Refresh Tokens, Password Reset Tokens).
   - GORM struct definitions, custom types, constraints, and indexes (`idx_user_status`).
   - Safe foreign keys, cascade deletes, and soft deletion (`gorm.DeletedAt`).
   - Production PostgreSQL connection pooling (`SetMaxOpenConns`, `SetMaxIdleConns`).

6. [06. Production Middleware & Observability](./06-production-middleware-and-observability.md)
   - Structured JSON logging using modern standard library `log/slog`.
   - Request ID tracing middleware (`X-Request-ID`).
   - Custom Panic Recovery middleware (prevents server crashes, returns clean 500 JSON).
   - Production CORS middleware.
   - Token-bucket IP Rate Limiting middleware (`golang.org/x/time/rate`).

7. [07. Authentication & Security Architecture](./07-authentication-and-security.md)
   - Password hashing with `bcrypt`.
   - Dual-token JWT architecture (Short-lived Access Tokens + Long-lived Refresh Tokens).
   - Secure Forgot & Reset Password lifecycle (cryptographic one-time tokens, SHA-256 storage).
   - Gin Auth Middleware (`Bearer <token>` extraction and validation).

8. [08. Complete Authentication Implementation Guide](./08-complete-auth-implementation-guide.md)
   - Complete, unabridged code for `UserRepository` and `TokenRepository`.
   - `EmailService` interface with local console mock and SMTP email sending.
   - Full `AuthService` (SignUp, Login, Token Refresh with rotation, Forgot & Reset password with ACID transactions).
   - Full `AuthHandler` with complete request DTOs and Gin bindings.

9. [09. API Design & Business Logic](./09-business-logic-and-api-design.md)
   - RESTful endpoint conventions and status codes.
   - Request DTOs & validation tags (`go-playground/validator`).
   - Standardized API response envelopes (`SendSuccess`, `SendPaginated`, `SendError`).
   - Repository & Service layers implementation for Tasks and Categories.
   - Advanced features: Pagination, dynamic filtering, sorting, and full-text search.

10. [10. Redis In-Memory Caching](./10-redis-caching.md)
    - Redis connection client (`github.com/redis/go-redis/v9`) with connection pooling.
    - Implementing the **Cache-Aside Pattern** with `CachedTaskRepository`.
    - Dynamic cache invalidation when tasks are created, updated, or deleted.
    - Inspecting cached keys and TTLs using **Redis Commander** GUI (`http://localhost:8082`).

11. [11. Concurrency: Goroutines, Channels & Background Workers](./11-concurrency-goroutines-channels-and-workers.md)
    - Concurrency primitives (`go`, `chan`, `select`, `sync.WaitGroup`, `time.Ticker`).
    - Non-blocking Email Worker Pool using buffered channels and worker goroutines.
    - Periodic background cleanup cron job for expired tokens using `time.Ticker`.
    - Coordinated, leak-free graceful shutdown using `context.Context` and `sync.WaitGroup`.

12. [12. Comprehensive Testing Guide](./12-comprehensive-testing-guide.md)
    - Go testing fundamentals (`go test`, table-driven tests).
    - Unit testing with mocks (`testify/mock`).
    - Handler testing with `net/http/httptest` and Gin test mode.
    - Database integration testing using the **Transaction Rollback Pattern** (`db.Begin()` / `defer tx.Rollback()`).
    - Test coverage and visual HTML reporting (`make test-coverage`).

13. [13. Step-by-Step Master Implementation Roadmap](./13-step-by-step-implementation-roadmap.md)
    - Sequential 9-phase execution checklist from an empty repo to production.
    - Graceful server shutdown pattern in `cmd/api/main.go` with `os.Signal` / `SIGTERM`.
    - Common Go and GORM pitfalls to avoid.
    - Final verification checklist.

---

## 🛠️ Technology Stack Overview

| Area | Choice | Reason |
|---|---|---|
| **Language** | Go (1.22+) | High performance, strict typing, great concurrency, simple deployment. |
| **HTTP Framework** | [Gin](https://github.com/gin-gonic/gin) | Fast HTTP router, rich middleware ecosystem, idiomatic request binding. |
| **ORM** | [GORM](https://gorm.io) | Feature-rich ORM, auto-migrations, hooks, relation preloading. |
| **Database** | PostgreSQL 16 | Robust ACID compliance, native JSON/array support, rich indexing capabilities. |
| **In-Memory Cache** | [Redis 7](https://github.com/redis/go-redis) | Sub-millisecond reads, cache-aside pattern, and query caching. |
| **Email Service** | Standard `net/smtp` (Go Standard Library) | Provider-agnostic: Mailtrap sandbox in dev, AWS SES / SendGrid in prod. Zero external SDKs. |
| **Auth** | [golang-jwt/jwt/v5](https://github.com/golang-jwt/jwt) & [bcrypt](https://pkg.go.dev/golang.org/x/crypto/bcrypt) | Industry-standard JWT parsing and secure password hashing. |
| **Config** | [Godotenv](https://github.com/joho/godotenv) | Flexible 12-factor configuration from environment variables. |
| **Testing** | Standard `testing`, [Testify](https://github.com/stretchr/testify) | Assertions, mocking, and real containerized database integration tests. |
| **Live Reload** | [Air](https://github.com/air-verse/air) | Instant server recompilation on code save. |

---

Begin your journey with [01. Go Core Concepts Explained for Web Developers](./01-go-core-concepts-for-web-devs.md)!
