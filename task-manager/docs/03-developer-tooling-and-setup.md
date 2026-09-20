# 03. Developer Tooling, Hot-Reloading & Productivity

In production Go development, professional tooling makes a massive difference in developer speed, code quality, and consistency. Unlike interpreted languages (Node.js/Python), Go is a compiled language, so tools like live-reload servers, linters, and build automation are essential.

---

## 1. Hot-Reloading / Auto-Refresh with Air

When developing a web API, stopping the server and running `go run cmd/api/main.go` every time you edit a file is tedious. The Go community's de-facto standard hot-reloader is **[Air](https://github.com/air-verse/air)**.

### How Air Works:
Air watches your Go source files for changes (`.go`, `.env`, `.yaml`). When a file is saved, Air automatically:
1. Interrupts the running process safely.
2. Re-compiles the binary into a temporary directory (`./tmp/main`).
3. Starts the new binary within milliseconds.

### Installing Air:
```bash
# Install the latest version of air
go install github.com/air-verse/air@latest
```
*(Make sure `$HOME/go/bin` is in your system `$PATH`)*

### Air Configuration File (`.air.toml`):
Create a `.air.toml` in your project root to customize build arguments, watched extensions, and ignored directories:

```toml
root = "."
tmp_dir = "tmp"

[build]
# Command to build the binary
cmd = "go build -o ./tmp/main ./cmd/api/main.go"
# Binary to execute
bin = "./tmp/main"
# Send an interrupt signal (SIGINT) to allow graceful shutdown
send_interrupt = true
kill_delay = "500ms"
# File extensions to watch
include_ext = ["go", "tpl", "tmpl", "html", "env"]
# Directories to ignore
exclude_dir = ["assets", "tmp", "vendor", "testdata", "docs"]
# Exclude specific files
exclude_file = []
# Exclude regex
exclude_regex = ["_test.go"]
# Delay before rebuilding after file changes (in milliseconds)
delay = 1000
# Stop running if build fails
stop_on_error = true
# Log file
log = "air.log"

[color]
main = "magenta"
watcher = "cyan"
build = "yellow"
runner = "green"

[misc]
clean_on_exit = true
```

### Running Your Server with Air:
Simply run:
```bash
air
```
Now, whenever you save any file in `internal/` or `cmd/`, your terminal will display:
```
watching .
!build rebuilding...
running...
Server listening on port 8080 in development mode
```

---

## 2. Professional `Makefile` for Automation

In Go projects, a `Makefile` serves as the central command runner for building, testing, linting, and running database migrations.

Create a `Makefile` in the root of your project:

```makefile
# Variables
APP_NAME=task-manager
MAIN_FILE=./cmd/api/main.go
MIGRATIONS_DIR=./migrations
DB_URL=postgres://postgres:postgres_secret_password@localhost:5432/taskmanager_db?sslmode=disable

.PHONY: help dev build run test test-coverage lint docker-up docker-down clean swagger

help: ## Show help for each of the Makefile recipes
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-18s\033[0m %s\n", $$1, $$2}'

dev: ## Run server with Air hot-reloading
	air

run: ## Run server directly with go run
	go run $(MAIN_FILE)

build: ## Compile production binary
	@echo "Building production binary..."
	CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o ./bin/$(APP_NAME) $(MAIN_FILE)

test: ## Run all tests
	go test -v -race ./...

test-coverage: ## Run tests with coverage and open HTML report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

lint: ## Run golangci-lint
	golangci-lint run ./...

docker-up: ## Start PostgreSQL and Adminer via Docker Compose
	docker compose up -d

docker-down: ## Stop Docker containers
	docker compose down

docker-logs: ## View PostgreSQL Docker logs
	docker compose logs -f postgres

clean: ## Clean build binaries and temporary files
	rm -rf ./tmp ./bin coverage.out

swagger: ## Generate Swagger/OpenAPI documentation
	swag init -g $(MAIN_FILE) -o ./docs/swagger
```

### Why Developers Love This:
Now, you never have to remember long terminal flags:
- Start local DB: `make docker-up`
- Start live-reload dev server: `make dev`
- Run test suite with race detection: `make test`
- View visual test coverage: `make test-coverage`
- Build optimized Linux binary: `make build`

---

## 3. Static Analysis & Linting (`golangci-lint`)

Go's built-in compiler catches syntax errors, but **[golangci-lint](https://golangci-lint.run/)** catches logic bugs, resource leaks, race conditions, missed error checks, and styling issues.

### Installing `golangci-lint`:
```bash
# On Linux / macOS using curl
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.60.3
```

### Configuration (`.golangci.yml`):
Create `.golangci.yml` in your project root:

```yaml
run:
  timeout: 5m
  tests: true

linters:
  enable:
    - errcheck      # Checks for unhandled errors
    - gosimple      # Suggests code simplifications
    - govet         # Standard Go vet tool
    - ineffassign   # Detects unused variable assignments
    - staticcheck   # Comprehensive static analysis
    - unused        # Checks for unused constants, variables, functions, and types
    - gocritic      # Opinionated stylistic and performance checks
    - gocyclo       # Computes function cyclomatic complexity
    - gosec         # Security auditing (detects SQL injection, weak crypto)

linters-settings:
  gocyclo:
    min-complexity: 15
  errcheck:
    check-blank: false
```

Run it anytime with:
```bash
make lint
```

---

## 4. Interactive API Documentation with Swagger (`swaggo`)

In production, frontend teams and third-party consumers need interactive documentation. **[Swaggo (swag)](https://github.com/swaggo/swag)** generates interactive Swagger UI documentation directly from Go code comments.

### 1. Install Swaggo:
```bash
go install github.com/swaggo/swag/cmd/swag@latest
go get github.com/swaggo/gin-swagger
go get github.com/swaggo/files
```

### 2. Annotate `cmd/api/main.go`:
```go
// @title           Task Manager API
// @version         1.0
// @description     Production-ready RESTful Task Manager API with full authentication.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@taskmanager.local

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and the JWT token.
```

### 3. Annotate Your Handlers (e.g. `CreateTask` in `internal/handler/task_handler.go`):
```go
// CreateTask godoc
// @Summary      Create a new task
// @Description  Creates a new task for the authenticated user
// @Tags         tasks
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        task  body      handler.CreateTaskRequest  true  "Task creation payload"
// @Success      201   {object}  handler.Response{data=models.Task}
// @Failure      400   {object}  handler.Response
// @Failure      401   {object}  handler.Response
// @Router       /tasks [post]
func (h *TaskHandler) CreateTask(c *gin.Context) { ... }
```

### 4. Register Swagger Route in Gin:
```go
import (
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "go-task-manager/docs/swagger" // generated docs
)

// Inside route setup:
router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
```

Generate docs with:
```bash
swag init -g ./cmd/api/main.go -o ./docs/swagger
```
Now visit `http://localhost:8080/swagger/index.html` in your browser to test your API interactively with a full UI!

---

## 5. Delve Debugger (`dlv`)

When you encounter a stubborn bug, printing with `fmt.Println` can be slow. **[Delve](https://github.com/go-delve/delve)** is the official debugger for Go.

### Install Delve:
```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

### Debugging with VS Code / Cursor / GoLand:
In `.vscode/launch.json`:
```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug Task Manager API",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}/cmd/api/main.go",
            "envFile": "${workspaceFolder}/.env"
        }
    ]
}
```
You can now set breakpoints, step through code line by line, inspect pointers, and watch goroutines execute in real time.

---

## 6. Summary of Developer Tooling

| Tool | Purpose | Command |
|---|---|---|
| **Air** | Live Auto-Reload Server | `air` |
| **Makefile** | Build & Command Automation | `make dev`, `make test`, `make build` |
| **golangci-lint** | Linting & Static Security Analysis | `make lint` |
| **Swaggo** | Auto-Generated Swagger API Docs | `make swagger` |
| **Delve** | Interactive Step-Through Debugger | `dlv debug ./cmd/api/main.go` |
| **Docker Compose** | One-command PostgreSQL & Adminer | `make docker-up` |

With these tools integrated, your development workflow will be as fast and pleasant as any modern web framework!

