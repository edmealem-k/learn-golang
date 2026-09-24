# Gin & GORM REST API

A fully functional, modular RESTful API built with **Go**, the **Gin** web framework, and **GORM** (Object-Relational Mapping) connected to **PostgreSQL**, following the [OneUptime Gin + GORM Guide](https://oneuptime.com/blog/post/2026-01-26-gin-gorm).

---

## 📌 Project Architecture & Directory Structure

The project separates concerns cleanly between database configuration, data models, business HTTP handlers, middleware, and routing:

```
gin-gorm-api/
├── config/
│   ├── database.go            # PostgreSQL connection, DSN formatting, and connection pool settings
│   └── migrate.go             # GORM AutoMigrate schema runner
├── models/
│   ├── user.go                # User model with table constraints and JSON tags
│   └── post.go                # Post and Tag models with relationships
├── handlers/
│   ├── user_handler.go        # User HTTP handlers (CRUD, pagination, search, restore)
│   ├── post_handler.go        # Post HTTP handlers (Preload, tags association, filtering)
│   └── transaction_example.go # Atomic transaction example (transferring post ownership)
├── middleware/
│   └── transaction.go         # Request-scoped database transaction middleware
├── routes/
│   └── routes.go              # Gin route definitions and endpoint grouping (/api/v1)
├── main.go                    # Application entry point with CORS and graceful startup
├── go.mod                     # Module dependencies
├── go.sum                     # Dependency checksums
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

## 📡 API Endpoints Reference

All API routes are prefixed under `/api/v1`:

### 👤 Users

| Method   | Endpoint                    | Description                                                                                                                            |
| -------- | --------------------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| `POST`   | `/api/v1/users`             | Create a new user with bcrypt password hashing                                                                                         |
| `GET`    | `/api/v1/users`             | List users with pagination (`?page=1&limit=10`), active filter (`?active=true`), and case-insensitive username search (`?search=john`) |
| `GET`    | `/api/v1/users/:id`         | Get user by ID                                                                                                                         |
| `PATCH`  | `/api/v1/users/:id`         | Partial update user details (username, email, bio, is_active)                                                                          |
| `DELETE` | `/api/v1/users/:id`         | Soft-delete a user (`?hard=true` for permanent delete)                                                                                 |
| `POST`   | `/api/v1/users/:id/restore` | Restore a soft-deleted user                                                                                                            |

### 📝 Posts & Associations

| Method | Endpoint                 | Description                                                                                                                  |
| ------ | ------------------------ | ---------------------------------------------------------------------------------------------------------------------------- |
| `POST` | `/api/v1/posts`          | Create a post and automatically associate/create tags (`FirstOrCreate`)                                                      |
| `GET`  | `/api/v1/posts`          | List posts with eager-loaded Author and Tags, filtered by `?published=true`, `?author_id=1`, or `?tag=golang` (via SQL join) |
| `GET`  | `/api/v1/posts/:id`      | Get post with preloaded `User`, `Tags`, `Replies`, and `Replies.User`                                                        |
| `POST` | `/api/v1/posts/transfer` | Atomically transfer post ownership to another user using a GORM database transaction                                         |

### 🏥 System

| Method | Endpoint  | Description                  |
| ------ | --------- | ---------------------------- |
| `GET`  | `/health` | Server health check endpoint |

---

## 🛠️ Environment Configuration

Set the following environment variables (e.g. in your environment or `.env` file):

```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=gin_gorm_db
PORT=8080
GIN_MODE=debug # set to 'release' in production
```

---

## 🚀 Running the Application

```bash
# 1. Download dependencies
go mod tidy

# 2. Run the server
go run main.go
```

---

## 💡 Notes & Watchouts

1. **Route Mapping in `routes/routes.go`**:
   - In `routes/routes.go`: line 18 maps `users.POST("", handlers.CreatePost)`. Make sure to update it to `handlers.CreateUser` so user registration points to the user handler!
2. **DTO Binding in `handlers/post_handler.go`**:
   - In `CreatePostInput`, check the tag on `UserId`: `binding:"reqquired"` has a small typo (`reqquired` $\rightarrow$ `required`).

---

## 🔗 Reference

- Blog Tutorial: [How to Use Gin with GORM (OneUptime)](https://oneuptime.com/blog/post/2026-01-26-gin-gorm)
