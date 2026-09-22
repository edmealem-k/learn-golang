# Gin & GORM REST API

A RESTful API built with **Go**, the **Gin** web framework, and **GORM** (Object-Relational Mapping) connected to **PostgreSQL**, following the [OneUptime Gin + GORM Guide](https://oneuptime.com/blog/post/2026-01-26-gin-gorm).

---

## 📌 Project Architecture & Structure

The project follows a standard modular Go structure separating concerns between database configuration, data models, HTTP handlers, and routing:

```
gin-gorm-api/
├── config/
│   ├── database.go       # PostgreSQL connection, DSN formatting, and connection pooling
│   └── migrate.go        # GORM AutoMigrate schema runner
├── models/
│   ├── user.go           # User model with table constraints and JSON tags
│   └── post.go           # Post and Tag models with relationships
├── handlers/
│   ├── user_handler.go   # User HTTP handlers and request DTOs
│   └── post_handler.go   # Post HTTP handlers (CRUD & associations)
├── routes/
│   └── routes.go         # Gin route definitions and endpoint grouping
├── main.go               # Application entrypoint
├── go.mod                # Module dependencies
├── go.sum                # Dependency checksums
└── README.md
```

---

## 📊 Database Models & Relationships

The project models a blogging / discussion system demonstrating key relational database patterns in GORM:

1. **User (`users`)**:
   - Primary key: `ID uint` (via `gorm.Model`).
   - Fields: `Username` (unique, max 50), `Email` (unique, max 100), `Password` (`json:"-"` hidden from responses), `Bio` (optional string pointer), `IsActive` (boolean).
   - Custom table name via `TableName()`.

2. **Post (`posts`)**:
   - Primary key: `ID uint` (via `gorm.Model`).
   - Fields: `Title`, `Content`, `Published`.
   - **Belongs-To (`User`)**: `UserID uint` foreign key linking each post to its author.
   - **Self-Referencing (`ParentID *uint`)**: Enables nested comments, replies, and discussion threads (`Parent` and `Replies []Post`).
   - **Many-to-Many (`Tags []Tag`)**: Connected to `tags` through a `post_tags` join table.

3. **Tag (`tags`)**:
   - Primary key: `ID uint` (via `gorm.Model`).
   - Fields: `Name` (unique, max 50).
   - Many-to-Many association back to `Posts`.

---

## 🛠️ Environment Configuration

The application requires the following environment variables to connect to PostgreSQL:

```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=gin_gorm_db
```

---

## 🚀 Progress & Implementation Checklist

### ✅ Implemented So Far:

- [x] **Project initialization** and dependency setup (`gin-gonic/gin`, `gorm.io/gorm`, `gorm.io/driver/postgres`, `golang.org/x/crypto`).
- [x] **Database connection pooling** (`config/database.go`) setting `MaxIdleConns`, `MaxOpenConns`, and `ConnMaxLifetime`.
- [x] **Schema migrations** (`config/migrate.go`) for `User`, `Post`, and `Tag`.
- [x] **Relational models** (`models/user.go`, `models/post.go`) with Belongs-To, Self-referencing, and Many-to-Many definitions.
- [x] **User registration** (`CreateUser` in `handlers/user_handler.go`) with DTO input validation (`binding:"required,email"`) and `bcrypt` password hashing.

### ⏳ To Complete Next (When Resuming):

- [ ] **User CRUD Handlers** (`handlers/user_handler.go`):
  - `GetUsers`: List users with limit/offset pagination.
  - `GetUser`: Fetch single user by ID.
  - `UpdateUser`: Partial update using `Updates` with DTO validation.
  - `DeleteUser`: Soft-delete user via GORM.
- [ ] **Post & Association Handlers** (`handlers/post_handler.go`):
  - `CreatePost`: Create posts with tag associations.
  - `GetPost`: Fetch post preloading author and tags using `.Preload("User").Preload("Tags")`.
- [ ] **Routes Configuration** (`routes/routes.go`):
  - Setup route groups (`/api/v1/users`, `/api/v1/posts`).
- [ ] **Server Initialization** (`main.go`):
  - Connect database, run migrations, register routes, and call `router.Run()`.
- [ ] **Transaction Middleware / Examples** (optional advanced topic from the guide).

---

## 💡 Notes & Watchouts for When You Resume

When you resume coding:

1. **`handlers/user_handler.go`**: Make sure to add `return` after `c.JSON(http.StatusBadRequest, ...)` and `c.JSON(http.StatusInternalServerError, ...)` so execution stops on error rather than falling through to database insert.
2. **`models/post.go`**: Double-check the tags on `Title` (`gorm:"not null;size:200"`) and ensure the closing quote on `json:"parent_id,omitempty"` is present.
3. **`config/database.go`**: In the DSN string, PostgreSQL expects `sslmode=disable` (rather than `disabled`), and check the environment variable name `DB_PASSWORD`.

---

## 🔗 Reference

- Blog Tutorial: [How to Use Gin with GORM (OneUptime)](https://oneuptime.com/blog/post/2026-01-26-gin-gorm)
