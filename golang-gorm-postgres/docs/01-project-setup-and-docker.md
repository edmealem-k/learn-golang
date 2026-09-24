# 01. Project Setup & Docker

In this first part, we will set up the project directory structure, initialize the Go module, configure a local PostgreSQL container using Docker Compose, and install the required Go packages.

---

## 1. Project Directory Structure

A clean, modular directory structure separates configuration, data models, business logic (controllers), routing, and utilities.

Here is the folder structure we will build:

```
golang-gorm-postgres/
├── controllers/          # HTTP request handlers (Auth, User, Post)
├── initializers/         # Config loading and database initialization
├── middleware/           # Gin middlewares (JWT deserialization & auth guard)
├── models/               # GORM database schemas & request DTOs
├── routes/               # Route groupings and controller wiring
├── utils/                # Password hashing and RS256 token utilities
├── app.env               # Environment variables (secrets, keys, database)
├── docker-compose.yml    # PostgreSQL container setup
├── go.mod                # Go module definition
├── go.sum                # Dependency checksums
├── main.go               # Server entry point
└── README.md
```

You can create these directories using:

```bash
mkdir -p controllers initializers middleware models routes utils
```

---

## 2. Initialize the Go Module

Run the following command in your terminal to initialize your Go module:

```bash
go mod init golang-gorm-postgres
```

---

## 3. Install Required Dependencies

Run these commands to install the necessary packages:

```bash
# Gin Web Framework & CORS middleware
go get github.com/gin-gonic/gin
go get github.com/gin-contrib/cors

# GORM ORM & PostgreSQL Driver
go get gorm.io/gorm
go get gorm.io/driver/postgres

# Modern JWT v5 (RS256 asymmetric token support)
go get github.com/golang-jwt/jwt/v5

# Password Hashing with Bcrypt
go get golang.org/x/crypto/bcrypt

# Google UUID package
go get github.com/google/uuid

# Viper Configuration Manager (parses .env files with durations)
go get github.com/spf13/viper
```

---

## 4. PostgreSQL Docker Container (`docker-compose.yml`)

Running PostgreSQL in a Docker container keeps your development environment isolated, reproducible, and easy to reset.

Create `docker-compose.yml` in the root of your project:

```yaml
version: "3.8"

services:
  postgres:
    image: postgres:16-alpine
    container_name: gorm_postgres
    restart: unless-stopped
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password123
      POSTGRES_DB: golang-gorm
    ports:
      - "6500:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres -d golang-gorm"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  postgres_data:
```

### Explanation of Docker Settings:

- **`ports: - "6500:5432"`**: We map the host port `6500` to PostgreSQL's container port `5432`. This avoids port conflicts if you already have another PostgreSQL instance running on your machine on port `5432`.
- **`volumes: - postgres_data:/var/lib/postgresql/data`**: Persists your database records between container restarts.

### Starting PostgreSQL:

```bash
docker compose up -d
```

Verify it is running:

```bash
docker compose ps
```

---

Next, continue to [02. Environment Config & RSA Keys](./02-environment-config-and-rsa-keys.md) to generate your RS256 RSA keys and configure Viper.
