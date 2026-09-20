# 04. Configuration & Environment Setup

Production Go services adhere to the **12-Factor App methodology**: configuration must be strictly separated from code and read from environment variables. Hardcoding database credentials, secrets, or ports is a severe anti-pattern.

---

## 1. The Production Configuration Strategy

A robust configuration system should:

1. **Fail Fast**: If a critical secret (like `JWT_SECRET`) is missing, the application should crash immediately upon startup with an explicit error, rather than failing in the middle of a user request.
2. **Provide Sensible Defaults**: For non-critical variables (like `PORT=8080`, `DB_MAX_OPEN_CONNS=25`), sensible defaults should be used if not explicitly overridden.
3. **Be Strongly Typed**: Parsed into integers, booleans, and durations (e.g., `time.Duration`) rather than passing raw strings throughout the application.

---

## 2. Implementing the Configuration Loader

Create `internal/config/config.go`:

```go
package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server
	AppEnv string
	Port   string

	// Database
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	DBSSLMode      string
	DBMaxOpenConns int
	DBMaxIdleConns int

	// Authentication & JWT
	JWTSecret            string
	JWTAccessExpiration  time.Duration
	JWTRefreshExpiration time.Duration

	// Email (for password reset)
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	FromEmail    string
	FrontendURL  string
}

func LoadConfig() (*Config, error) {
	// In development, load .env file if it exists.
	// In production (Docker/Kubernetes), env vars are injected directly by the runtime.
	if err := godotenv.Load(); err != nil {
		log.Println("Note: .env file not found, reading from system environment variables")
	}

	cfg := &Config{
		AppEnv:               getEnv("APP_ENV", "development"),
		Port:                 getEnv("PORT", "8080"),
		DBHost:               getEnv("DB_HOST", "localhost"),
		DBPort:               getEnv("DB_PORT", "5432"),
		DBUser:               getEnv("DB_USER", "postgres"),
		DBPassword:           getEnv("DB_PASSWORD", ""),
		DBName:               getEnv("DB_NAME", "taskmanager_db"),
		DBSSLMode:            getEnv("DB_SSLMODE", "disable"),
		DBMaxOpenConns:       getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:       getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
		JWTSecret:            getEnv("JWT_SECRET", ""),
		JWTAccessExpiration:  getEnvAsDuration("JWT_ACCESS_EXPIRATION", 15*time.Minute),
		JWTRefreshExpiration: getEnvAsDuration("JWT_REFRESH_EXPIRATION", 7*24*time.Hour),
		SMTPHost:             getEnv("SMTP_HOST", "sandbox.smtp.mailtrap.io"),
		SMTPPort:             getEnvAsInt("SMTP_PORT", 2525),
		SMTPUser:             getEnv("SMTP_USER", ""),
		SMTPPassword:         getEnv("SMTP_PASSWORD", ""),
		FromEmail:            getEnv("FROM_EMAIL", "no-reply@taskmanager.local"),
		FrontendURL:          getEnv("FRONTEND_URL", "http://localhost:3000"),
	}

	// Validate required variables (Fail-Fast principle)
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required and cannot be empty")
	}
	if cfg.DBPassword == "" && cfg.AppEnv == "production" {
		return nil, fmt.Errorf("DB_PASSWORD must be provided in production")
	}

	return cfg, nil
}

// Helper functions to retrieve env values with fallbacks
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	strValue := getEnv(key, "")
	if value, err := strconv.Atoi(strValue); err == nil {
		return value
	}
	return fallback
}

func getEnvAsDuration(key string, fallback time.Duration) time.Duration {
	strValue := getEnv(key, "")
	if value, err := time.ParseDuration(strValue); err == nil {
		return value
	}
	return fallback
}
```

---

## 3. Environment Variables Template (`.env.example`)

Create `.env.example` in the root of your project. Developers can copy it to `.env`:

```ini
# Application
APP_ENV=development
PORT=8080
FRONTEND_URL=http://localhost:3000

# PostgreSQL Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres_secret_password
DB_NAME=taskmanager_db
DB_SSLMODE=disable
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=10

# Security & Tokens
# Generate a strong 32+ character key: `openssl rand -base64 32`
JWT_SECRET=super_secret_jwt_signing_key_change_me_in_prod
JWT_ACCESS_EXPIRATION=15m
JWT_REFRESH_EXPIRATION=168h

# Email / SMTP (Use Mailtrap or Mailhog for local testing)
SMTP_HOST=sandbox.smtp.mailtrap.io
SMTP_PORT=2525
SMTP_USER=your_smtp_user
SMTP_PASSWORD=your_smtp_password
FROM_EMAIL=no-reply@taskmanager.local
```

> [!CAUTION]
> Always add `.env` to your `.gitignore`. Never commit live secrets or passwords to Git. Only `.env.example` should be committed.

---

## 4. Local PostgreSQL with Docker Compose

To run PostgreSQL locally without installing it directly on your machine, create a `docker-compose.yml` in your project root:

```yaml
version: "3.8"

services:
  postgres:
    image: postgres:16-alpine
    container_name: taskmanager_postgres
    restart: unless-stopped
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres_secret_password
      POSTGRES_DB: taskmanager_db
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres -d taskmanager_db"]
      interval: 5s
      timeout: 5s
      retries: 5

  # Optional: Web UI to browse PostgreSQL tables
  adminer:
    image: adminer:latest
    container_name: taskmanager_adminer
    restart: unless-stopped
    ports:
      - "8081:8080"
    depends_on:
      - postgres

volumes:
  postgres_data:
```

### Running the Database:

Start PostgreSQL in the background:

```bash
docker compose up -d postgres
```

Check health:

```bash
docker compose ps
```

You can also navigate to `http://localhost:8081` in your browser to view your PostgreSQL tables via Adminer.

---

Next, continue to [07. Authentication & Security](./07-authentication-and-security.md) to implement password hashing, JWTs, forgot/reset password flows, and Gin middleware.
