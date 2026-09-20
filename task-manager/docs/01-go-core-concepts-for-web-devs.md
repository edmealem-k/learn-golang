# 01. Go Core Concepts Explained for Web Developers

If you are coming from languages like JavaScript/TypeScript, Python, PHP, or Java, Go's design philosophy can feel very different. Go does not have classes, inheritance, exceptions (`try/catch`), or magical ORM reflection without explicit struct tags.

This guide explains the foundational Go concepts you must understand to write, debug, and reason about web APIs.

---

## 1. Pointers vs. Values (`*` and `&`)

In Go, variables are stored either directly in memory (by value) or referenced by memory address (by pointer).

### The Basics:

- `&variable`: "Address of" operator. Yields a memory address (a pointer).
- `*Type`: Declares a pointer type (e.g., `*models.Task` means "a pointer to a `Task`").
- `*pointer`: Dereferences the pointer. Accesses the actual value at the memory address.

### Why Do We Use Pointers in Web APIs?

#### Reason 1: Mutating Structs in Functions / Methods

In Go, **arguments are passed by value (copied)**. If you pass a struct to a function and modify it, you are modifying a copy!

```go
// ❌ WRONG: Modifies a copy; the original task remains unchanged!
func (s *taskService) CompleteTask(t models.Task) {
    t.Status = models.StatusCompleted
}

// ✅ RIGHT: Modifies the original task in memory
func (s *taskService) CompleteTask(t *models.Task) {
    t.Status = models.StatusCompleted
}
```

#### Reason 2: Avoiding Expensive Memory Copies

If a struct contains large fields (strings, slices, nested structs), passing `*Task` copies only an 8-byte memory address, rather than the entire struct.

#### Reason 3: Representing `NULL` in JSON and PostgreSQL

Primitive types in Go have default zero-values:

- `string` defaults to `""`
- `int` defaults to `0`
- `bool` defaults to `false`
- `time.Time` defaults to `0001-01-01 00:00:00 UTC`

If a database column or JSON payload allows `NULL` (such as `due_date` or `category_id`), a plain `time.Time` or `uint` cannot represent `nil`. It would mistakenly default to `0` or `0001-01-01`.
By using a pointer:

```go
DueDate    *time.Time `json:"due_date"`    // Can be nil (translates to NULL in SQL and null in JSON)
CategoryID *uint      `json:"category_id"` // Can be nil
```

#### Reason 4: DTO Updates (Detecting Omitted Fields vs Zero-Values)

In a `PUT` or `PATCH` request:

```go
type UpdateTaskRequest struct {
    Title  *string `json:"title"`
    Status *string `json:"status"`
}
```

- If client sends: `{"status": "completed"}`
  - `Title` is `nil` $\rightarrow$ do not update `title`.
  - `Status` is `&"completed"` $\rightarrow$ update `status`.
- If client sends: `{"title": ""}` (explicit empty title)
  - `Title` is `&""` $\rightarrow$ client explicitly passed an empty string, which validation can reject.

---

## 2. Interfaces & Implicit Implementation ("Duck Typing")

In Java, C#, or TypeScript, a class must explicitly declare that it implements an interface:

```typescript
class TaskRepository implements ITaskRepository { ... }
```

In Go, interfaces are implemented **implicitly**. If a struct has all the methods defined by an interface with matching signatures, it automatically satisfies that interface.

```go
// 1. Define interface
type Notifier interface {
    SendNotification(userID uint, message string) error
}

// 2. Concrete struct
type EmailService struct {
    smtpServer string
}

// 3. EmailService satisfies Notifier simply by having this method
func (e *EmailService) SendNotification(userID uint, message string) error {
    fmt.Printf("Emailing user %d: %s\n", userID, message)
    return nil
}

// 4. Any function expecting a Notifier can accept *EmailService!
func NotifyUser(n Notifier, userID uint, msg string) {
    n.SendNotification(userID, msg)
}
```

### Why This Matters for Production Code:

- **Zero Circular Dependencies**: Services can define what they need (`TaskRepository` interface) in their own package without knowing what database library or driver is being used.
- **Trivial Unit Testing**: In your tests, you can create a `MockTaskRepository` that satisfies the interface and returns fake data without touching PostgreSQL.

---

## 3. Error Handling Without Exceptions

Go deliberately does **not** have exceptions (`try / catch / throw`). Errors in Go are ordinary values that implement the `error` interface:

```go
type error interface {
    Error() string
}
```

### The Idiomatic Go Error Pattern:

Functions that can fail return the error as their last return value:

```go
user, err := userRepo.FindByEmail(ctx, email)
if err != nil {
    // 1. Handle or log the error
    // 2. Return early (Guard Clause pattern)
    return nil, fmt.Errorf("failed to find user by email: %w", err)
}
// 3. Proceed with happy path
```

### Error Wrapping (`%w`) & `errors.Is`:

When passing errors up through layers (Repository $\rightarrow$ Service $\rightarrow$ Handler), wrap the error with context using `fmt.Errorf("...: %w", err)`:

```go
// In repository:
if err := r.db.First(&task, id).Error; err != nil {
    return nil, fmt.Errorf("taskRepo.GetByID: %w", err)
}

// In handler: Check if the root cause was "Record Not Found"
if errors.Is(err, gorm.ErrRecordNotFound) {
    SendError(c, http.StatusNotFound, "Task does not exist")
    return
}
```

The `%w` verb wraps the original error like an onion. `errors.Is(err, target)` traverses through the layers to inspect if the root error matches `target`.

---

## 4. Understanding `context.Context`

Every HTTP request handled by Gin creates a Go `context.Context`. You access it via:

```go
ctx := c.Request.Context()
```

### What Does `context.Context` Do?

1. **Cancellation Propagation**:
   If a user clicks a button in their browser to load tasks, and then immediately closes the browser tab or cancels the request, Gin cancels the request's `context.Context`.
2. **Database Query Abort**:
   When you write:
   ```go
   r.db.WithContext(ctx).Find(&tasks)
   ```
   PostgreSQL is notified immediately if the client disconnects, and PostgreSQL cancels the running query! This prevents wasted CPU and memory on your database server.
3. **Timeouts**:
   You can attach a maximum execution time to critical operations:
   ```go
   ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
   defer cancel()
   ```
   If the database takes longer than 3 seconds, the query is automatically aborted with `context.DeadlineExceeded`.

---

## 5. Struct Tags and How Gin & GORM Use Them

Struct tags are metadata strings enclosed in backticks placed after field definitions:

```go
type CreateTaskRequest struct {
    Title       string     `json:"title" binding:"required,min=3,max=100"`
    Description string     `json:"description"`
    DueDate     *time.Time `json:"due_date" time_format:"2006-01-02T15:04:05Z07:00"`
}
```

### Tag Explanations:

1. `json:"title"`:
   - Controls how Go's `encoding/json` marshals (struct $\rightarrow$ JSON) and unmarshals (JSON $\rightarrow$ struct) this field.
   - `json:"-"` means: **Never serialize this field** (vital for `PasswordHash`).
   - `json:"tasks,omitempty"` means: If the slice is empty or `nil`, omit the `"tasks"` key from the output JSON.
2. `binding:"required,min=3"`:
   - Used by Gin's validator package (`go-playground/validator`).
   - When calling `c.ShouldBindJSON(&req)`, Gin inspects these tags. If `title` is missing or less than 3 characters, Gin automatically returns a descriptive validation error.
3. `gorm:"..."`:
   - Used by GORM when creating tables and running queries.
   - `gorm:"primaryKey"`: Sets column as primary key.
   - `gorm:"uniqueIndex"`: Creates a unique index in PostgreSQL.
   - `gorm:"type:varchar(100);not null"`: Explicit SQL column definition.

---

## 6. Concurrency in Web Servers & Goroutines

In Gin, **every incoming HTTP request is automatically executed in its own separate goroutine** (a lightweight thread managed by the Go runtime).

If 500 users send requests simultaneously, Gin runs 500 goroutines concurrently.

### Concurrency Rules for Web Handlers:

1. **Never mutate shared global variables across requests**:
   ```go
   // ❌ DANGEROUS: Concurrent requests modifying this slice will cause a fatal race condition crash!
   var globalTasks = []Task{}
   ```
2. **Each request should work on its own scoped data**:
   Pass data through the handler, service, and repository as local variables or return values.
3. **Database connections are thread-safe**:
   `gorm.DB` and the underlying `sql.DB` connection pool are completely safe to be shared across goroutines.

---

Next, proceed to [06. Production Middleware & Observability](./06-production-middleware-and-observability.md) to learn how production APIs log requests, catch panics, handle CORS, and apply rate limiting.
