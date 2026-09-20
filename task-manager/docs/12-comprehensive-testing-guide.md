# 12. Comprehensive Testing Guide

In production engineering, automated tests are your safety net. They ensure new features don't break existing functionality and allow you to refactor with confidence.

Go has first-class built-in testing tools. This guide will walk you through the concepts from the ground up, teaching you how to test a layered Go application.

---

## 1. Go Testing Fundamentals

### How Go Discovers Tests:

- Any file ending in `_test.go` (e.g., `task_service_test.go`) is recognized as a test file and is compiled **only** when running `go test`.
- Test functions must start with `Test` and accept a single parameter `t *testing.T`:
  ```go
  func TestCalculateDueDate(t *testing.T) { ... }
  ```
- Subtests are organized using `t.Run("subtest name", func(t *testing.T) { ... })`.

### The "Table-Driven Test" Pattern

Idiomatic Go code heavily relies on **table-driven tests**. Instead of copying and pasting test functions for each case, you define a slice of test cases (inputs and expected outputs) and iterate over them.

```go
func TestCheckPasswordHash(t *testing.T) {
	hashedPassword, _ := utils.HashPassword("MySecret123!")

	// Define the test table
	tests := []struct {
		name          string
		plainPassword string
		hash          string
		expectedMatch bool
	}{
		{
			name:          "Valid password matches",
			plainPassword: "MySecret123!",
			hash:          hashedPassword,
			expectedMatch: true,
		},
		{
			name:          "Wrong password fails",
			plainPassword: "WrongPassword",
			hash:          hashedPassword,
			expectedMatch: false,
		},
		{
			name:          "Empty password fails",
			plainPassword: "",
			hash:          hashedPassword,
			expectedMatch: false,
		},
	}

	// Iterate and run each case
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := utils.CheckPasswordHash(tc.plainPassword, tc.hash)
			if result != tc.expectedMatch {
				t.Errorf("expected match %v, got %v", tc.expectedMatch, result)
			}
		})
	}
}
```

---

## 2. The Testing Pyramid in Go

```
        /  End-to-End (E2E)  \       (Fewest: Slowest, spins up all servers)
       /──────────────────────\
      /   Integration Tests    \     (Medium: Real DB transactions or Testcontainers)
     /──────────────────────────\
    /     HTTP Handler Tests     \   (Fast: httptest, verifies JSON & status codes)
   /──────────────────────────────\
  /       Unit Tests (Mocks)       \ (Most: Instant, tests pure business logic)
```

---

## 3. Unit Testing with Mocks (`testify/mock`)

Unit tests test business logic (the **Service Layer**) in complete isolation. We do **not** connect to a real PostgreSQL database. Instead, we use a **Mock Repository** that simulates database responses.

### A. Creating the Mock Repository (`internal/service/mocks/task_repository_mock.go`)

Install testify: `go get github.com/stretchr/testify`

```go
package mocks

import (
	"context"

	"go-task-manager/internal/models"
	"go-task-manager/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockTaskRepository struct {
	mock.Mock
}

func (m *MockTaskRepository) Create(ctx context.Context, task *models.Task) error {
	args := m.Called(ctx, task)
	return args.Error(0)
}

func (m *MockTaskRepository) GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*models.Task, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Task), args.Error(1)
}

func (m *MockTaskRepository) List(ctx context.Context, filter repository.TaskFilter) ([]models.Task, int64, int, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]models.Task), args.Get(1).(int64), args.Get(2).(int), args.Error(3)
}

func (m *MockTaskRepository) Update(ctx context.Context, task *models.Task) error {
	args := m.Called(ctx, task)
	return args.Error(0)
}

func (m *MockTaskRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}
```

### B. Writing the Service Unit Test (`internal/service/task_service_test.go`)

Here we test the business rule: _"When a task status is changed to 'completed', the `CompletedAt` timestamp must be automatically populated."_

```go
package service_test

import (
	"context"
	"testing"
	"time"

	"go-task-manager/internal/models"
	"github.com/google/uuid"
	"go-task-manager/internal/service"
	"go-task-manager/internal/service/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUpdateTask_StatusCompletedSetsTimestamp(t *testing.T) {
	// 1. Arrange (Setup mock and service)
	mockRepo := new(mocks.MockTaskRepository)
	taskSvc := service.NewTaskService(mockRepo)

	taskID := uuid.New()
	userID := uuid.New()

	existingTask := &models.Task{
		Base:        models.Base{ID: taskID},
		UserID:      userID,
		Title:       "Buy milk",
		Status:      models.StatusPending,
		CompletedAt: nil,
	}

	// Expect GetByID(ctx, taskID, userID) to return existingTask
	mockRepo.On("GetByID", mock.Anything, taskID, userID).Return(existingTask, nil)

	// Expect Update(ctx, task) where CompletedAt is not nil
	mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(task *models.Task) bool {
		return task.Status == models.StatusCompleted && task.CompletedAt != nil
	})).Return(nil)

	// 2. Act
	updates := map[string]interface{}{
		"status": models.StatusCompleted,
	}
	updatedTask, err := taskSvc.UpdateTask(context.Background(), taskID, userID, updates)

	// 3. Assert
	assert.NoError(t, err)
	assert.NotNil(t, updatedTask.CompletedAt)
	assert.Equal(t, models.StatusCompleted, updatedTask.Status)

	// Verify all expected mock calls were actually executed
	mockRepo.AssertExpectations(t)
}
```

---

## 4. HTTP Handler Testing (`net/http/httptest`)

To test Gin handlers, you don't need to boot an HTTP server on a port. Go provides `net/http/httptest` to execute mock HTTP requests in memory.

### Writing the Handler Test (`internal/handler/task_handler_test.go`):

```go
package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-task-manager/internal/handler"
	"go-task-manager/internal/models"
	"go-task-manager/internal/service/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTaskHandler_CreateTask(t *testing.T) {
	// Set Gin to test mode to silence debug logs
	gin.SetMode(gin.TestMode)

	t.Run("Success - 201 Created", func(t *testing.T) {
		mockService := new(mocks.MockTaskService)
		taskHandler := handler.NewTaskHandler(mockService)

		router := gin.New()
		// Simulate auth middleware setting userID = 5
		router.Use(func(c *gin.Context) {
			c.Set("userID", uint(5))
			c.Next()
		})
		router.POST("/tasks", taskHandler.CreateTask)

		// Mock expectations
		taskID := uuid.New()
		expectedTask := &models.Task{
			Base:   models.Base{ID: taskID},
			UserID: userID,
			Title:  "Clean room",
			Status: models.StatusPending,
		}
		mockService.On("CreateTask", mock.Anything, userID, mock.AnythingOfType("*models.Task")).
			Return(expectedTask, nil)

		// Create fake request payload
		body := handler.CreateTaskRequest{
			Title: "Clean room",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest(http.MethodPost, "/tasks", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		// ResponseRecorder acts as the client's browser receiving response
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert HTTP Status & Response Body
		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "Clean room")
	})

	t.Run("Validation Failure - 400 Bad Request", func(t *testing.T) {
		mockService := new(mocks.MockTaskService)
		taskHandler := handler.NewTaskHandler(mockService)

		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("userID", uint(5))
		})
		router.POST("/tasks", taskHandler.CreateTask)

		// Send empty title (which violates `binding:"required"`)
		req, _ := http.NewRequest(http.MethodPost, "/tasks", bytes.NewBufferString(`{"title": ""}`))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Error")
	})
}
```

---

## 5. Database Integration Testing (Transaction Rollback Pattern)

When testing the **Repository Layer**, you want to verify that actual SQL queries, GORM hooks, foreign keys, and indexes work as intended against a real PostgreSQL instance.

### The Secret to Fast Database Tests: Transaction Rollback

Instead of dropping and recreating tables for every test (which is very slow), use this pattern:

1. Open a transaction at the start of the test: `tx := testDB.Begin()`.
2. Pass `tx` to the repository.
3. At the end of the test, rollback the transaction: `defer tx.Rollback()`.
4. **Result**: Your test database stays pristine, and tests execute in milliseconds without interfering with each other!

```go
package repository_test

import (
	"context"
	"testing"

	"go-task-manager/internal/models"
	"go-task-manager/internal/repository"
	"go-task-manager/testutils" // helper package connecting to test database

	"github.com/stretchr/testify/assert"
)

func TestPostgresTaskRepository_CreateAndGet(t *testing.T) {
	// Connect to test database once
	db := testutils.GetTestDB()

	// Begin isolated transaction
	tx := db.Begin()
	defer tx.Rollback() // Rollback guarantees no junk data is left in the DB

	repo := repository.NewPostgresTaskRepository(tx)

	// Create dummy user
	user := &models.User{
		Name:         "Alice",
		Email:        "alice@example.com",
		PasswordHash: "dummyhash",
	}
	tx.Create(user)

	// Test task creation
	task := &models.Task{
		UserID: user.ID,
		Title:  "Write integration tests",
		Status: models.StatusPending,
	}

	err := repo.Create(context.Background(), task)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, task.ID) // Database should assign valid UUID

	// Test retrieval
	found, err := repo.GetByID(context.Background(), task.ID, user.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Write integration tests", found.Title)
}
```

---

## 6. How to Run Tests & Inspect Coverage

Run all tests across the entire project:

```bash
go test -v ./...
```

Run tests with the Go race detector enabled (vital for catching concurrency bugs):

```bash
go test -race ./...
```

Generate test coverage report:

```bash
# Output coverage percentages
go test -cover ./...

# Generate an interactive visual HTML coverage map
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

_(This opens a browser tab highlighting in green every line of code tested, and in red any untested code branch!)_

---

Next, continue to [13. Step-by-Step Implementation Roadmap](./13-step-by-step-implementation-roadmap.md) for a checklist to build out your project.
