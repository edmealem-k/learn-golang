# 09. API Design & Business Logic

A professional API is predictable, provides consistent JSON responses, handles errors gracefully, and enforces business rules cleanly.

---

## 1. RESTful API Specification

| Method   | Endpoint                       | Auth | Description                                    |   Status Code    |
| -------- | ------------------------------ | :--: | ---------------------------------------------- | :--------------: |
| `POST`   | `/api/v1/auth/signup`          |  No  | Register new user account                      |  `201 Created`   |
| `POST`   | `/api/v1/auth/login`           |  No  | Login and receive Access & Refresh tokens      |     `200 OK`     |
| `POST`   | `/api/v1/auth/refresh`         |  No  | Exchange Refresh token for new Access token    |     `200 OK`     |
| `POST`   | `/api/v1/auth/forgot-password` |  No  | Request password reset email                   |     `200 OK`     |
| `POST`   | `/api/v1/auth/reset-password`  |  No  | Reset password using reset token               |     `200 OK`     |
| `GET`    | `/api/v1/tasks`                | Yes  | List tasks with filtering, search & pagination |     `200 OK`     |
| `POST`   | `/api/v1/tasks`                | Yes  | Create a new task                              |  `201 Created`   |
| `GET`    | `/api/v1/tasks/:id`            | Yes  | Get a single task by ID                        |     `200 OK`     |
| `PUT`    | `/api/v1/tasks/:id`            | Yes  | Update an existing task                        |     `200 OK`     |
| `DELETE` | `/api/v1/tasks/:id`            | Yes  | Soft-delete a task                             | `204 No Content` |
| `GET`    | `/api/v1/categories`           | Yes  | List user's custom categories                  |     `200 OK`     |
| `POST`   | `/api/v1/categories`           | Yes  | Create a new category                          |  `201 Created`   |

---

## 2. Standardized API Responses & Error Handling

Never return raw errors or ad-hoc JSON shapes across different endpoints. Maintain a uniform response envelope.

Create `internal/handler/response.go`:

```go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response standard JSON structure
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Meta    interface{} `json:"meta,omitempty"` // For pagination metadata
	Error   string      `json:"error,omitempty"`
}

// PaginationMeta pagination metadata
type PaginationMeta struct {
	CurrentPage int   `json:"current_page"`
	PageSize    int   `json:"page_size"`
	TotalItems  int64 `json:"total_items"`
	TotalPages  int   `json:"total_pages"`
}

func SendSuccess(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func SendPaginated(c *gin.Context, data interface{}, meta PaginationMeta) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

func SendError(c *gin.Context, statusCode int, errMsg string) {
	c.JSON(statusCode, Response{
		Success: false,
		Error:   errMsg,
	})
}
```

---

## 3. Data Transfer Objects (DTOs) & Input Validation

Never bind request JSON directly to database models (`models.Task`). Use **DTOs** (Data Transfer Objects) with validation tags.

Create `internal/handler/dto.go`:

```go
package handler

import (
	"time"
	"github.com/google/uuid"
)

// Auth DTOs
type SignUpRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=100"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=72"`
}

// Task DTOs
type CreateTaskRequest struct {
	Title       string     `json:"title" binding:"required,min=1,max=255"`
	Description string     `json:"description"`
	Status      string     `json:"status" binding:"omitempty,oneof=pending in_progress completed archived"`
	Priority    string     `json:"priority" binding:"omitempty,oneof=low medium high urgent"`
	DueDate     *time.Time `json:"due_date"`
	CategoryID  *uuid.UUID `json:"category_id"`
}

type UpdateTaskRequest struct {
	Title       *string    `json:"title" binding:"omitempty,min=1,max=255"`
	Description *string    `json:"description"`
	Status      *string    `json:"status" binding:"omitempty,oneof=pending in_progress completed archived"`
	Priority    *string    `json:"priority" binding:"omitempty,oneof=low medium high urgent"`
	DueDate     *time.Time `json:"due_date"`
	CategoryID  *uuid.UUID `json:"category_id"`
}

// TaskFilterQuery for query params: /tasks?page=1&page_size=10&status=pending&search=groceries
type TaskFilterQuery struct {
	Page       int    `form:"page,default=1" binding:"min=1"`
	PageSize   int    `form:"page_size,default=10" binding:"min=1,max=100"`
	Status     string `form:"status" binding:"omitempty,oneof=pending in_progress completed archived"`
	Priority   string `form:"priority" binding:"omitempty,oneof=low medium high urgent"`
	CategoryID *uuid.UUID `form:"category_id"`
	Search     string `form:"search"`
	SortBy     string `form:"sort_by,default=created_at" binding:"omitempty,oneof=created_at due_date priority title"`
	Order      string `form:"order,default=desc" binding:"omitempty,oneof=asc desc"`
}
```

> [!TIP]
> **Why Pointers in `UpdateTaskRequest`?**
> If `Title` were a plain `string`, an empty string `""` could mean "the client didn't send a title" OR "the client wants to clear the title". With `*string`, if `Title == nil`, the client omitted the field (do not update). If `Title != nil`, update it to the dereferenced value.

---

## 4. Repository Implementation with Dynamic Filtering

Create `internal/repository/task_repository.go`:

```go
package repository

import (
	"context"
	"fmt"
	"math"

	"go-task-manager/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TaskFilter struct {
	UserID     uuid.UUID
	Status     string
	Priority   string
	CategoryID *uuid.UUID
	Search     string
	SortBy     string
	Order      string
	Page       int
	PageSize   int
}

type TaskRepository interface {
	Create(ctx context.Context, task *models.Task) error
	GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*models.Task, error)
	List(ctx context.Context, filter TaskFilter) ([]models.Task, int64, int, error)
	Update(ctx context.Context, task *models.Task) error
	Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}

type postgresTaskRepository struct {
	db *gorm.DB
}

func NewPostgresTaskRepository(db *gorm.DB) TaskRepository {
	return &postgresTaskRepository{db: db}
}

func (r *postgresTaskRepository) Create(ctx context.Context, task *models.Task) error {
	return r.db.WithContext(ctx).Create(task).Error
}

func (r *postgresTaskRepository) GetByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*models.Task, error) {
	var task models.Task
	// Enforce userID ownership in every query!
	err := r.db.WithContext(ctx).
		Preload("Category").
		Where("id = ? AND user_id = ?", id, userID).
		First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *postgresTaskRepository) List(ctx context.Context, filter TaskFilter) ([]models.Task, int64, int, error) {
	var tasks []models.Task
	var totalItems int64

	query := r.db.WithContext(ctx).Model(&models.Task{}).Where("user_id = ?", filter.UserID)

	// Dynamic filters
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Priority != "" {
		query = query.Where("priority = ?", filter.Priority)
	}
	if filter.CategoryID != nil {
		query = query.Where("category_id = ?", *filter.CategoryID)
	}
	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		query = query.Where("title ILIKE ? OR description ILIKE ?", searchPattern, searchPattern)
	}

	// Count matching items
	if err := query.Count(&totalItems).Error; err != nil {
		return nil, 0, 0, err
	}

	// Calculate pagination
	totalPages := int(math.Ceil(float64(totalItems) / float64(filter.PageSize)))
	offset := (filter.Page - 1) * filter.PageSize

	// Apply Sorting and Pagination
	orderClause := fmt.Sprintf("%s %s", filter.SortBy, filter.Order)
	err := query.Order(orderClause).
		Limit(filter.PageSize).
		Offset(offset).
		Preload("Category").
		Find(&tasks).Error

	return tasks, totalItems, totalPages, err
}

func (r *postgresTaskRepository) Update(ctx context.Context, task *models.Task) error {
	return r.db.WithContext(ctx).Save(task).Error
}

func (r *postgresTaskRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.Task{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
```

---

## 5. Service Layer (Business Rules)

Create `internal/service/task_service.go`:

```go
package service

import (
	"context"
	"errors"
	"time"

	"go-task-manager/internal/models"
	"go-task-manager/internal/repository"
	"github.com/google/uuid"
)

type TaskService interface {
	CreateTask(ctx context.Context, userID uuid.UUID, req *models.Task) (*models.Task, error)
	GetTask(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*models.Task, error)
	ListTasks(ctx context.Context, filter repository.TaskFilter) ([]models.Task, int64, int, error)
	UpdateTask(ctx context.Context, id uuid.UUID, userID uuid.UUID, updates map[string]interface{}) (*models.Task, error)
	DeleteTask(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}

type taskService struct {
	repo repository.TaskRepository
}

func NewTaskService(repo repository.TaskRepository) TaskService {
	return &taskService{repo: repo}
}

func (s *taskService) CreateTask(ctx context.Context, userID uuid.UUID, task *models.Task) (*models.Task, error) {
	task.UserID = userID
	if task.Status == "" {
		task.Status = models.StatusPending
	}
	if task.Priority == "" {
		task.Priority = models.PriorityMedium
	}

	if err := s.repo.Create(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *taskService) GetTask(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*models.Task, error) {
	return s.repo.GetByID(ctx, id, userID)
}

func (s *taskService) ListTasks(ctx context.Context, filter repository.TaskFilter) ([]models.Task, int64, int, error) {
	return s.repo.List(ctx, filter)
}

func (s *taskService) UpdateTask(ctx context.Context, id uuid.UUID, userID uuid.UUID, updates map[string]interface{}) (*models.Task, error) {
	task, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	// Business rule: if status is changed to "completed", stamp CompletedAt
	if newStatus, ok := updates["status"].(models.TaskStatus); ok {
		if newStatus == models.StatusCompleted && task.Status != models.StatusCompleted {
			now := time.Now()
			task.CompletedAt = &now
		} else if newStatus != models.StatusCompleted {
			task.CompletedAt = nil
		}
		task.Status = newStatus
	}

	if title, ok := updates["title"].(string); ok {
		task.Title = title
	}
	if desc, ok := updates["description"].(string); ok {
		task.Description = desc
	}
	if priority, ok := updates["priority"].(models.TaskPriority); ok {
		task.Priority = priority
	}
	if dueDate, ok := updates["due_date"].(*time.Time); ok {
		task.DueDate = dueDate
	}

	if err := s.repo.Update(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *taskService) DeleteTask(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return s.repo.Delete(ctx, id, userID)
}
```

---

## 6. HTTP Handler Implementation

Create `internal/handler/task_handler.go`:

```go
package handler

import (
	"errors"
	"net/http"
	"github.com/google/uuid"

	"go-task-manager/internal/middleware"
	"go-task-manager/internal/models"
	"go-task-manager/internal/repository"
	"go-task-manager/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TaskHandler struct {
	taskService service.TaskService
}

func NewTaskHandler(taskService service.TaskService) *TaskHandler {
	return &TaskHandler{taskService: taskService}
}

func (h *TaskHandler) CreateTask(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		SendError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		SendError(c, http.StatusBadRequest, err.Error())
		return
	}

	task := &models.Task{
		Title:       req.Title,
		Description: req.Description,
		Status:      models.TaskStatus(req.Status),
		Priority:    models.TaskPriority(req.Priority),
		DueDate:     req.DueDate,
		CategoryID:  req.CategoryID,
	}

	createdTask, err := h.taskService.CreateTask(c.Request.Context(), userID, task)
	if err != nil {
		SendError(c, http.StatusInternalServerError, "Failed to create task")
		return
	}

	SendSuccess(c, http.StatusCreated, "Task created successfully", createdTask)
}

func (h *TaskHandler) ListTasks(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		SendError(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var query TaskFilterQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		SendError(c, http.StatusBadRequest, err.Error())
		return
	}

	filter := repository.TaskFilter{
		UserID:     userID,
		Status:     query.Status,
		Priority:   query.Priority,
		CategoryID: query.CategoryID,
		Search:     query.Search,
		SortBy:     query.SortBy,
		Order:      query.Order,
		Page:       query.Page,
		PageSize:   query.PageSize,
	}

	tasks, totalItems, totalPages, err := h.taskService.ListTasks(c.Request.Context(), filter)
	if err != nil {
		SendError(c, http.StatusInternalServerError, "Failed to fetch tasks")
		return
	}

	meta := PaginationMeta{
		CurrentPage: query.Page,
		PageSize:    query.PageSize,
		TotalItems:  totalItems,
		TotalPages:  totalPages,
	}

	SendPaginated(c, tasks, meta)
}

func (h *TaskHandler) GetTask(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		SendError(c, http.StatusBadRequest, "Invalid task UUID")
		return
	}

	task, err := h.taskService.GetTask(c.Request.Context(), taskID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			SendError(c, http.StatusNotFound, "Task not found")
			return
		}
		SendError(c, http.StatusInternalServerError, "Failed to get task")
		return
	}

	SendSuccess(c, http.StatusOK, "Task retrieved", task)
}

func (h *TaskHandler) DeleteTask(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		SendError(c, http.StatusBadRequest, "Invalid task UUID")
		return
	}

	if err := h.taskService.DeleteTask(c.Request.Context(), taskID, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			SendError(c, http.StatusNotFound, "Task not found")
			return
		}
		SendError(c, http.StatusInternalServerError, "Failed to delete task")
		return
	}

	c.Status(http.StatusNoContent)
}
```

---

Next, proceed to [12. Comprehensive Testing Guide](./12-comprehensive-testing-guide.md) to learn how to write unit tests with mocks, HTTP integration tests, and database tests.
