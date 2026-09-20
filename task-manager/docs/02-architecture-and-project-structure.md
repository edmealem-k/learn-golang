# 02. Architecture & Project Layout

When developing applications in Go, project layout and architectural separation are critical. Unlike frameworks in other ecosystems (such as Django, Rails, or NestJS) that enforce a rigid folder structure, Go offers freedom. However, the Go community has converged on proven patterns that ensure scalability, maintainability, and testability.

---

## 1. The Standard Go Project Layout

The industry-standard directory structure for a production-grade backend service looks like this:

```
task-manager/
├── cmd/
│   └── api/
│       └── main.go                 # Application entrypoint (wires dependencies and starts HTTP server)
├── internal/                       # Private application code (compiler-enforced privacy)
│   ├── config/                     # Environment variable and config file loading
│   │   └── config.go
│   ├── database/                   # PostgreSQL connection & migrations
│   │   └── postgres.go
│   ├── models/                     # Domain models and GORM entities
│   │   ├── user.go
│   │   ├── task.go
│   │   ├── category.go
│   │   └── token.go
│   ├── repository/                 # Data access layer (SQL/GORM queries)
│   │   ├── user_repository.go
│   │   ├── task_repository.go
│   │   └── token_repository.go
│   ├── service/                    # Business logic layer
│   │   ├── auth_service.go
│   │   ├── task_service.go
│   │   └── email_service.go
│   ├── handler/                    # HTTP controllers / Gin route handlers
│   │   ├── auth_handler.go
│   │   ├── task_handler.go
│   │   └── response.go             # Standardized JSON response helpers
│   ├── middleware/                 # Gin HTTP middlewares (Auth, Logger, RateLimit, CORS)
│   │   ├── auth_middleware.go
│   │   └── cors.go
│   └── validator/                  # Custom request validation rules
├── pkg/                            # Optional: Public reusable packages that can be imported by other projects
│   └── utils/                      # Password hashing, token generators, etc.
│       ├── hash.go
│       └── jwt.go
├── docs/                           # Documentation and guides
├── docker-compose.yml              # Local PostgreSQL container definition
├── Dockerfile                      # Production container image build
├── .env.example                    # Template for environment variables
├── go.mod                          # Go module dependencies
└── go.sum                          # Dependency checksums
```

### Why is `internal/` special in Go?

In Go, any directory named `internal/` receives special compiler enforcement:

- Code inside `internal/` can **only** be imported by packages within the parent tree of that `internal/` directory.
- Third-party packages or other projects cannot import your internal business logic.
- This prevents unintentional tight-coupling when your codebase grows.

### Why `cmd/api/main.go`?

- The `cmd/` directory houses the `main` packages of your project.
- Having subdirectories like `cmd/api/` allows you to add other executable commands in the future without cluttering the root (e.g., `cmd/migrate/` for database migrations, `cmd/worker/` for background queue workers).
- Keep `main.go` **thin**. Its sole responsibility is to:
  1. Load configuration.
  2. Establish database connections.
  3. Instantiate repositories, services, and handlers (Dependency Injection).
  4. Register routes.
  5. Start the HTTP server with graceful shutdown handling.

---

## 2. Layered Architecture (Clean Architecture)

A production application isolates concerns across distinct layers. The core rule: **Dependencies flow in one direction only**.

```
   HTTP Requests
         │
         ▼
 ┌──────────────┐
 │   Handler    │  (Transport Layer: Gin HTTP contexts, binds JSON, sends HTTP status)
 └──────┬───────┘
        │ Calls Service Interface
        ▼
 ┌──────────────┐
 │   Service    │  (Business Logic: Validates business rules, orchestrates operations)
 └──────┬───────┘
        │ Calls Repository Interface
        ▼
 ┌──────────────┐
 │  Repository  │  (Data Access Layer: Runs GORM queries against PostgreSQL)
 └──────┬───────┘
        │ Reads / Writes
        ▼
 ┌──────────────┐
 │   Database   │  (PostgreSQL storage)
 └──────────────┘
```

### Breakdown of Layer Responsibilities

#### 1. Handler Layer (`internal/handler`)

- Knows **only** about HTTP requests and responses.
- Binds incoming JSON/query parameters using Gin's `c.ShouldBindJSON`.
- Passes pure Go structs or primitives to the Service layer.
- Converts results or errors into appropriate HTTP status codes (`200 OK`, `400 Bad Request`, `404 Not Found`, `500 Internal Server Error`).
- **Never** makes direct database queries or handles business calculations.

#### 2. Service Layer (`internal/service`)

- Houses the core business rules of your application.
- Example rules:
  - "A user can only view and update tasks that belong to them."
  - "When completing a task, verify it isn't already archived."
  - "Before creating a user, verify the email is not taken, and hash the password."
- **Never** imports `gin.Context`. The service layer must remain transport-agnostic (you could reuse it for a gRPC server, CLI command, or background worker without altering a single line).

#### 3. Repository Layer (`internal/repository`)

- Encapsulates all database access code using GORM.
- Contains methods such as `Create(task *models.Task)`, `GetByID(id uint) (*models.Task, error)`, `List(userID uint, filter TaskFilter) ([]models.Task, error)`.
- If you later decide to switch ORMs or optimize queries with raw SQL, only this layer changes.

#### 4. Model Layer (`internal/models`)

- Pure Go structs defining the database tables, GORM mappings, and JSON representations.
- Free from dependencies on handlers, services, or repositories.

---

## 3. Dependency Injection & Decoupling with Interfaces

In Go, interfaces are satisfied **implicitly**. You do not write `implements TaskRepository`. If a struct implements the methods defined in an interface, Go treats it as satisfying that interface.

### The Golden Rule of Go Interfaces:

> _"Accept interfaces, return structs."_
> Define interfaces where they are **consumed**, not where they are implemented.

### Example: Decoupling Service from Repository

In `internal/service/task_service.go`, define what the service needs from the repository:

```go
package service

import (
	"context"
	"go-task-manager/internal/models"
)

// TaskRepository defines the contracts required by TaskService.
// This allows us to inject a MockTaskRepository during unit tests!
type TaskRepository interface {
	Create(ctx context.Context, task *models.Task) error
	GetByID(ctx context.Context, id uint, userID uint) (*models.Task, error)
	List(ctx context.Context, userID uint) ([]models.Task, error)
	Update(ctx context.Context, task *models.Task) error
	Delete(ctx context.Context, id uint, userID uint) error
}

type taskService struct {
	repo TaskRepository
}

// NewTaskService constructor accepts the interface.
func NewTaskService(repo TaskRepository) *taskService {
	return &taskService{repo: repo}
}
```

And in `internal/repository/task_repository.go`, implement the concrete struct:

```go
package repository

import (
	"context"
	"go-task-manager/internal/models"
	"gorm.io/gorm"
)

type PostgresTaskRepository struct {
	db *gorm.DB
}

func NewPostgresTaskRepository(db *gorm.DB) *PostgresTaskRepository {
	return &PostgresTaskRepository{db: db}
}

func (r *PostgresTaskRepository) Create(ctx context.Context, task *models.Task) error {
	return r.db.WithContext(ctx).Create(task).Error
}

// ... other methods ...
```

Notice how `PostgresTaskRepository` automatically satisfies `TaskRepository` without importing `task_service.go`! This completely prevents circular dependencies.

---

## 4. Recommended Production Packages

Here are the industry-standard libraries to install in your `go.mod`:

| Package                                                | Purpose              | Why This Package?                                                                                      |
| ------------------------------------------------------ | -------------------- | ------------------------------------------------------------------------------------------------------ |
| `github.com/gin-gonic/gin`                             | HTTP Web Framework   | High speed, proven reliability, robust middleware chaining, built-in validation support.               |
| `gorm.io/gorm`                                         | ORM Core             | Ergonomic query building, transaction support, hooks, and relationships.                               |
| `gorm.io/driver/postgres`                              | GORM Postgres Driver | Official PostgreSQL driver built on top of `pgx`.                                                      |
| `github.com/golang-jwt/jwt/v5`                         | JWT Handling         | The most audited, modern, and active JWT library in Go.                                                |
| `golang.org/x/crypto/bcrypt`                           | Password Hashing     | Adaptive one-way hashing with automatic salting to securely store passwords.                           |
| `github.com/spf13/viper` or `github.com/joho/godotenv` | Configuration        | Loading `.env` and environment variables cleanly.                                                      |
| `github.com/go-playground/validator/v10`               | Input Validation     | Gin's default validator. Provides rich tags (`email`, `min=8`, `oneof=pending in_progress completed`). |
| `github.com/stretchr/testify`                          | Testing Suite        | Provides `assert`, `require`, and `mock` for clean, expressive unit tests.                             |
| `github.com/google/uuid`                               | UUID Generation      | Cryptographically secure UUIDs (v4/v7) for tokens and identifiers.                                     |

To add these packages to your project:

```bash
go get github.com/gin-gonic/gin
go get gorm.io/gorm
go get gorm.io/driver/postgres
go get github.com/golang-jwt/jwt/v5
go get golang.org/x/crypto/bcrypt
go get github.com/joho/godotenv
go get github.com/google/uuid
go get github.com/stretchr/testify
```

---

Next, continue to [05. Database Design & PostgreSQL Models](./05-database-design-and-models.md) to explore relational schema design, GORM tags, relationships, and connection pooling.
