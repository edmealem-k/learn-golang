# 07. Authentication & Security Architecture

A production authentication system must protect against credential theft, replay attacks, brute-force attempts, and unauthorized data access. This chapter details the design and implementation of:

1. **Bcrypt Password Hashing**.
2. **Dual-Token JWT Architecture** (Short-lived Access Token + Rotating Refresh Token).
3. **Forgot & Reset Password Lifecycle** (Cryptographic one-time tokens).
4. **Gin Auth Middleware** (Extracting authenticated user context).

---

## 1. Authentication Architecture Overview

```
                      ┌──────────────────────────────────────────────┐
                      │                 Client                       │
                      └──────┬───────────────────────────────▲───────┘
                             │                               │
        1. Login             │ POST /auth/login              │ Access Token (15m)
        (Email + Password)   ▼                               │ Refresh Token (7d)
                      ┌──────────────┐                       │
                      │ Auth Handler │───────────────────────┘
                      └──────┬───────┘
                             │ Check Bcrypt Password
                             ▼
                      ┌──────────────┐
                      │  PostgreSQL  │ (Stores User & Hashed Refresh Token)
                      └──────────────┘

     Subsequent Protected Requests:
     GET /tasks (Header: "Authorization: Bearer <access_token>")
     ├── AuthMiddleware extracts token & verifies signature.
     ├── Sets `userID` in `gin.Context`.
     └── Next handler retrieves `c.Get("userID")` without querying the DB!
```

---

## 2. Security Utilities (`pkg/utils`)

Create reusable, pure utility functions under `pkg/utils/`.

### A. Password & Token Hashing (`pkg/utils/hash.go`)

```go
package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a plain password with a cost of 12
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(bytes), nil
}

// CheckPasswordHash compares a plain password against its bcrypt hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateRandomHex generates a cryptographically secure random string of n bytes
func GenerateRandomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// HashToken computes a SHA-256 hash of a string (used for refresh & reset tokens)
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
```

> [!NOTE]
> **Why SHA-256 for tokens instead of Bcrypt?**
> Bcrypt is intentionally slow (to prevent brute-forcing short human passwords). Reset tokens and refresh tokens already possess 256 bits of high entropy generated via `crypto/rand`. A fast SHA-256 hash is safe and prevents high CPU usage while looking up tokens in PostgreSQL.

---

### B. JWT Utilities (`pkg/utils/jwt.go`)

```go
package utils

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTClaims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	jwt.RegisteredClaims
}

// GenerateToken generates an HMAC-SHA256 signed JWT
func GenerateToken(userID uuid.UUID, email string, secret string, duration time.Duration) (string, error) {
	claims := JWTClaims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "task-manager-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken verifies the signature and expiration of a JWT
func ValidateToken(tokenString string, secret string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&JWTClaims{},
		func(token *jwt.Token) (interface{}, error) {
			// Validate signing algorithm is HMAC
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
		},
	)

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid or expired token")
	}

	return claims, nil
}
```

---

## 3. The Forgot & Reset Password Lifecycle

Password reset is often implemented insecurely. Here is the production-grade flow:

### Step 1: Request Password Reset (`POST /api/v1/auth/forgot-password`)

1. User provides email: `{"email": "user@example.com"}`.
2. Find user by email.
   - _Security tip_: If user is **not found**, still return `200 OK` with `"If the email exists, a reset link has been sent"`. This prevents **User Enumeration Attacks** (attackers checking which emails are registered).
3. Generate a 32-byte raw cryptographic hex string: `rawToken, _ := utils.GenerateRandomHex(32)`.
4. Compute `hashedToken := utils.HashToken(rawToken)`.
5. Save `hashedToken` in `password_reset_tokens` table with:
   - `UserID = user.ID`
   - `ExpiresAt = time.Now().Add(15 * time.Minute)`
6. Send email to user with link: `https://frontend.com/reset-password?token=<rawToken>`.

### Step 2: Reset Password (`POST /api/v1/auth/reset-password`)

1. Client submits: `{"token": "<rawToken>", "new_password": "NewStrongPassword123!"}`.
2. Compute `hashedToken := utils.HashToken(rawToken)`.
3. Query DB:
   ```sql
   SELECT * FROM password_reset_tokens
   WHERE token_hash = ? AND used_at IS NULL AND expires_at > NOW();
   ```
4. If invalid or expired, return `400 Bad Request: Invalid or expired reset token`.
5. Hash the `new_password` with Bcrypt.
6. In a single database **transaction**:
   - Update `users.password_hash`.
   - Set `password_reset_tokens.used_at = NOW()`.
   - Invalidate all existing `refresh_tokens` for this user (`UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = ?`).
7. Return `200 OK: Password reset successfully`.

---

## 4. Gin Authentication Middleware

Create `internal/middleware/auth_middleware.go`. This middleware protects private routes:

```go
package middleware

import (
	"net/http"
	"strings"

	"go-task-manager/pkg/utils"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates the Bearer JWT from Authorization header
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Authorization header is required",
			})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Authorization header format must be Bearer <token>",
			})
			return
		}

		tokenString := parts[1]
		claims, err := utils.ValidateToken(tokenString, jwtSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid or expired token",
			})
			return
		}

		// Set user data in Gin's context for downstream handlers to read
		c.Set("userID", claims.UserID)
		c.Set("userEmail", claims.Email)

		c.Next()
	}
}

// GetUserID retrieves the authenticated user's ID from gin.Context
func GetUserID(c *gin.Context) (uuid.UUID, bool) {
	val, exists := c.Get("userID")
	if !exists {
		return uuid.Nil, false
	}
	id, ok := val.(uuid.UUID)
	return id, ok
}
```

---

## 5. Protecting Routes in Gin

Register public vs protected route groups in your router:

```go
func SetupRoutes(r *gin.Engine, authHandler *handler.AuthHandler, taskHandler *handler.TaskHandler, jwtSecret string) {
	api := r.Group("/api/v1")
	{
		// Public Auth Endpoints
		auth := api.Group("/auth")
		{
			auth.POST("/signup", authHandler.SignUp)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/forgot-password", authHandler.ForgotPassword)
			auth.POST("/reset-password", authHandler.ResetPassword)
		}

		// Protected Endpoints (Requires valid Bearer token)
		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware(jwtSecret))
		{
			// Tasks
			tasks := protected.Group("/tasks")
			{
				tasks.POST("", taskHandler.CreateTask)
				tasks.GET("", taskHandler.ListTasks)
				tasks.GET("/:id", taskHandler.GetTask)
				tasks.PUT("/:id", taskHandler.UpdateTask)
				tasks.DELETE("/:id", taskHandler.DeleteTask)
			}
		}
	}
}
```

---

Next, continue to [09. API Design & Business Logic](./09-business-logic-and-api-design.md) to implement the request DTOs, data validations, repository queries, and service layer logic.
