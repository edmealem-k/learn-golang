# 06. Production Middleware & Observability

In production, APIs must be **observable** (giving you visibility into what happened when a request failed) and **resilient** (protected against abuse, unexpected panics, and cross-origin security issues).

This guide provides drop-in, production-grade middleware for Gin.

---

## 1. Structured Logging with Go's `log/slog`

Starting with Go 1.21, the standard library includes `log/slog`, a fast, structured JSON logger. In production, logs should always be emitted as structured JSON so log aggregators (Datadog, Grafana Loki, CloudWatch) can parse and filter by fields like `request_id`, `status_code`, and `duration_ms`.

Create `internal/middleware/logger.go`:

```go
package middleware

import (
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// InitLogger initializes the global slog logger (JSON in production, text in dev)
func InitLogger(isProduction bool) *slog.Logger {
	var handler slog.Handler
	if isProduction {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

// StructuredLogger logs HTTP requests with latency, status, IP, and Request ID
func StructuredLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery

		// Process request
		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		requestID := c.GetString("RequestID")

		if rawQuery != "" {
			path = path + "?" + rawQuery
		}

		// Choose log level based on status code
		var logLevel slog.Level
		switch {
		case statusCode >= 500:
			logLevel = slog.LevelError
		case statusCode >= 400:
			logLevel = slog.LevelWarn
		default:
			logLevel = slog.LevelInfo
		}

		slog.Log(c.Request.Context(), logLevel, "HTTP Request",
			slog.String("request_id", requestID),
			slog.String("method", method),
			slog.String("path", path),
			slog.Int("status", statusCode),
			slog.Duration("latency", latency),
			slog.String("client_ip", clientIP),
		)
	}
}
```

---

## 2. Request ID Middleware (`X-Request-ID`)

Every incoming request should receive a unique UUID. This ID is:

1. Returned in the response header `X-Request-ID` to the client.
2. Attached to all log statements.
   If a user contacts support with an error, they can provide the `X-Request-ID`, and you can instantly locate all related database queries and log traces in your server logs.

Create `internal/middleware/request_id.go`:

```go
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const RequestIDHeader = "X-Request-ID"

// RequestID attaches a unique identifier to every incoming request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Use existing ID if client sent one (useful for microservice tracing), or generate a new UUID
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Set in Gin context (for handlers/loggers) and response header (for the client)
		c.Set("RequestID", requestID)
		c.Header(RequestIDHeader, requestID)

		c.Next()
	}
}
```

---

## 3. Panic Recovery Middleware (Never Crash in Production)

If a developer accidentally references a `nil` pointer (e.g., `user.Tasks[0]` when `Tasks` is empty), Go will trigger a `panic`. If unhandled, this can crash the server or hang the connection.

Create `internal/middleware/recovery.go`:

```go
package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// CustomRecovery recovers from any panics and writes a 500 JSON response
func CustomRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				requestID := c.GetString("RequestID")
				stack := string(debug.Stack())

				slog.Error("CRITICAL: Server panic recovered",
					slog.String("request_id", requestID),
					slog.String("error", fmt.Sprintf("%v", err)),
					slog.String("stack_trace", stack),
				)

				// Never leak stack traces to the public API client!
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"success":    false,
					"error":      "An unexpected internal error occurred. Please try again later.",
					"request_id": requestID,
				})
			}
		}()

		c.Next()
	}
}
```

---

## 4. CORS (Cross-Origin Resource Sharing) Middleware

When your frontend (e.g. Next.js, React running on `http://localhost:3000`) communicates with your backend API (`http://localhost:8080`), web browsers block requests unless CORS headers are explicitly provided.

Create `internal/middleware/cors.go`:

```go
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func CORSMiddleware(allowedOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Request-ID")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		// Handle HTTP preflight requests
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
```

---

## 5. Rate Limiting Middleware (Preventing Brute-Force Attacks)

Authentication endpoints like `/auth/login` and `/auth/forgot-password` are prime targets for credential-stuffing and brute-force attacks.

We can apply a token-bucket rate limiter per client IP using the official Go rate library:

```bash
go get golang.org/x/time/rate
```

Create `internal/middleware/ratelimit.go`:

```go
package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// clientLimiter holds a rate limiter and its last seen timestamp
type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type IPRateLimiter struct {
	sync.Mutex
	clients map[string]*clientLimiter
	rate    rate.Limit
	burst   int
}

func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	limiter := &IPRateLimiter{
		clients: make(map[string]*clientLimiter),
		rate:    r,
		burst:   b,
	}

	// Clean up stale IP records every 5 minutes to prevent memory leak
	go limiter.cleanupStaleEntries()
	return limiter
}

func (i *IPRateLimiter) getClient(ip string) *rate.Limiter {
	i.Lock()
	defer i.Unlock()

	c, exists := i.clients[ip]
	if !exists {
		limiter := rate.NewLimiter(i.rate, i.burst)
		i.clients[ip] = &clientLimiter{limiter: limiter, lastSeen: time.Now()}
		return limiter
	}

	c.lastSeen = time.Now()
	return c.limiter
}

func (i *IPRateLimiter) cleanupStaleEntries() {
	for {
		time.Sleep(5 * time.Minute)
		i.Lock()
		for ip, client := range i.clients {
			if time.Since(client.lastSeen) > 10*time.Minute {
				delete(i.clients, ip)
			}
		}
		i.Unlock()
	}
}

// RateLimitMiddleware enforces rate limits (e.g. 5 requests per second, burst of 10)
func RateLimitMiddleware(limiter *IPRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		clientLimiter := limiter.getClient(ip)

		if !clientLimiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error":   "Too many requests. Please slow down and try again later.",
			})
			return
		}

		c.Next()
	}
}
```

---

## 6. Wiring It All Together in Gin

In your router setup:

```go
func SetupRouter(cfg *config.Config) *gin.Engine {
	// Initialize logger
	isProd := cfg.AppEnv == "production"
	middleware.InitLogger(isProd)

	// Create blank Gin engine (we use our own custom middlewares instead of gin.Default())
	router := gin.New()

	// Global Middlewares (Applied to every request in order)
	router.Use(middleware.RequestID())
	router.Use(middleware.StructuredLogger())
	router.Use(middleware.CustomRecovery())
	router.Use(middleware.CORSMiddleware(cfg.FrontendURL))

	// Strict rate limiter for auth (max 5 requests per second, burst of 10)
	authLimiter := middleware.NewIPRateLimiter(rate.Every(time.Second/5), 10)

	authGroup := router.Group("/api/v1/auth")
	authGroup.Use(middleware.RateLimitMiddleware(authLimiter))
	{
		// authGroup.POST("/login", ...)
		// authGroup.POST("/forgot-password", ...)
	}

	return router
}
```

---

Next, continue to [08. Complete Authentication Implementation Guide](./08-complete-auth-implementation-guide.md) to see the full service, repository, and email notification code for the entire auth system.
