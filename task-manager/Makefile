APP_NAME=task-manager
MAIN_FILE=./cmd/api/main.go
DB_URL=postgres://postgres:postgres_secret_password@localhost:5432/taskmanager_db?sslmode=disable

.PHONY: help dev build run test test-coverage lint docker-up docker-down clean swagger

help: ## Show help for each Makefile target
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-18s\033[0m %s\n", $$1, $$2}'

dev: ## Run server with Air hot-reloading
	air

run: ## Run server directly with go run
	go run $(MAIN_FILE)

build: ## Compile production binary
	@echo "Building production binary..."
	CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o ./bin/$(APP_NAME) $(MAIN_FILE)

test: ## Run all tests with race detector
	go test -v -race ./...

test-coverage: ## Run tests with coverage and open HTML report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

lint: ## Run golangci-lint
	golangci-lint run ./...

docker-up: ## Start PostgreSQL and Adminer containers
	docker compose up -d

docker-down: ## Stop Docker containers
	docker compose down

docker-logs: ## View PostgreSQL Docker logs
	docker compose logs -f postgres

clean: ## Clean build binaries and temporary files
	rm -rf ./tmp ./bin coverage.out air.log

swagger: ## Generate Swagger API documentation
	swag init -g $(MAIN_FILE) -o ./docs/swagger

