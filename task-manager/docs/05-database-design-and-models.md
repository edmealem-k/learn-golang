# 05. Database Design & PostgreSQL Models (with UUIDs)

In a modern production application, data design dictates performance, data integrity, and API security. We use **UUIDs (Universally Unique Identifiers)** as primary and foreign keys.

### Why UUIDs over Sequential Integers?
1. **Prevents ID Enumeration / Scraping**: Attackers cannot guess sequential IDs (`/tasks/1`, `/tasks/2`).
2. **Distributed ID Generation**: The backend or frontend can generate unique IDs (`uuid.New()`) before saving to PostgreSQL.
3. **Multi-tenant Security**: Client-facing URLs never leak the total count of users or tasks in your database.

We use Go's industry-standard package:
```bash
go get github.com/google/uuid
```

---

## 1. Relational Schema & Entity Relationships

```
 ┌──────────────────────────────────────┐
 │               Users                  │
 │ (ID: UUID, Name, Email, PasswordHash)│
 └──────────────────┬───────────────────┘
                    │ 1:N (Foreign Key: user_id UUID)
         ┌──────────┴───────────────────────────┐
         │ 1:N                                  │ 1:N
         ▼                                      ▼
 ┌───────────────────────────┐          ┌───────────────────────────────┐
 │        Categories         │          │             Tasks             │
 │ (ID: UUID, UserID, Name)  │          │ (ID: UUID, UserID, CatID, ...)│
 └─────────────┬─────────────┘          └───────────────────────────────┘
               │ 1:N (Nullable)                         ▲
               └────────────────────────────────────────┘
         
 Additional Security Tables (1:N with Users):
 ├── Refresh Tokens (ID: UUID, UserID: UUID, TokenHash)
 └── Password Reset Tokens (ID: UUID, UserID: UUID, TokenHash)
```

---

## 2. GORM Models Implementation with UUIDs

Create these models inside `internal/models/`.

### A. Base Model Helper (`internal/models/base.go`)

To avoid repeating `ID`, `CreatedAt`, `UpdatedAt`, and `DeletedAt` on every struct, we define a reusable `Base` model:

```go
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Base replaces gorm.Model with a UUID primary key
type Base struct {
	ID        uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate GORM hook ensures a UUID is generated if not already provided
func (b *Base) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}
```

---

### B. User Model (`internal/models/user.go`)

```go
package models

import (
	"github.com/google/uuid"
)

type User struct {
	Base
	Name         string     `gorm:"type:varchar(100);not null" json:"name"`
	Email        string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash string     `gorm:"type:varchar(255);not null" json:"-"` // "-" prevents leaking password in JSON
	IsActive     bool       `gorm:"default:true;not null" json:"is_active"`

	// Relationships
	Tasks        []Task     `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"tasks,omitempty"`
	Categories   []Category `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"categories,omitempty"`
}
```

---

### C. Task & Category Models (`internal/models/task.go`)

```go
package models

import (
	"time"

	"github.com/google/uuid"
)

type TaskStatus string

const (
	StatusPending    TaskStatus = "pending"
	StatusInProgress TaskStatus = "in_progress"
	StatusCompleted  TaskStatus = "completed"
	StatusArchived   TaskStatus = "archived"
)

type TaskPriority string

const (
	PriorityLow    TaskPriority = "low"
	PriorityMedium TaskPriority = "medium"
	PriorityHigh   TaskPriority = "high"
	PriorityUrgent TaskPriority = "urgent"
)

type Category struct {
	Base
	UserID uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Name   string    `gorm:"type:varchar(50);not null" json:"name"`
	Color  string    `gorm:"type:varchar(7);default:'#6B7280'" json:"color"` // Hex e.g. #3B82F6
}

type Task struct {
	Base
	UserID      uuid.UUID     `gorm:"type:uuid;not null;index:idx_user_status;index:idx_user_due" json:"user_id"`
	CategoryID  *uuid.UUID    `gorm:"type:uuid;index" json:"category_id"` // Pointer allows NULL value
	Category    *Category     `gorm:"foreignKey:CategoryID;constraint:OnDelete:SET NULL" json:"category,omitempty"`

	Title       string        `gorm:"type:varchar(255);not null" json:"title"`
	Description string        `gorm:"type:text" json:"description"`
	Status      TaskStatus    `gorm:"type:varchar(20);default:'pending';not null;index:idx_user_status" json:"status"`
	Priority    TaskPriority  `gorm:"type:varchar(20);default:'medium';not null" json:"priority"`

	DueDate     *time.Time    `gorm:"index:idx_user_due" json:"due_date"`
	CompletedAt *time.Time    `json:"completed_at"` // Populated automatically when status becomes 'completed'
}
```

---

### D. Authentication Tokens (`internal/models/token.go`)

```go
package models

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID        uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	TokenHash string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"-"`
	UserAgent string     `gorm:"type:varchar(255)" json:"user_agent"`
	ClientIP  string     `gorm:"type:varchar(45)" json:"client_ip"`
	ExpiresAt time.Time  `gorm:"not null;index" json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at"`
	CreatedAt time.Time  `json:"created_at"`
}

type PasswordResetToken struct {
	ID        uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	TokenHash string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"-"`
	ExpiresAt time.Time  `gorm:"not null;index" json:"expires_at"`
	UsedAt    *time.Time `json:"used_at"`
	CreatedAt time.Time  `json:"created_at"`
}
```

---

## 3. Database Connection & Connection Pooling

GORM runs on top of Go's standard `database/sql` driver. Create `internal/database/postgres.go`:

```go
package database

import (
	"fmt"
	"log"
	"time"

	"go-task-manager/internal/config"
	"go-task-manager/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort, cfg.DBSSLMode,
	)

	var gormLogLevel logger.LogLevel
	if cfg.AppEnv == "production" {
		gormLogLevel = logger.Error
	} else {
		gormLogLevel = logger.Info
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(gormLogLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	// Connection Pool Settings
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.DBMaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.DBMaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(15 * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Successfully connected to PostgreSQL database")
	return db, nil
}

func Migrate(db *gorm.DB) error {
	log.Println("Running database migrations with UUID support...")
	return db.AutoMigrate(
		&models.User{},
		&models.Category{},
		&models.Task{},
		&models.RefreshToken{},
		&models.PasswordResetToken{},
	)
}
```

---

Next, continue to [06. Production Middleware & Observability](./06-production-middleware-and-observability.md).
