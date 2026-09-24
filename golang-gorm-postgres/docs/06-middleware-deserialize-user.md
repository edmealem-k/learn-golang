# 06. Middleware & Authentication Guards

In this section, we create the `DeserializeUser` middleware (`middleware/deserialize-user.go`). This middleware protects private endpoints, extracts credentials from either cookies or the `Authorization` header, verifies the RS256 signature, and injects the authenticated user into the request context.

---

## 1. Dual-Extraction Strategy (Cookie + Bearer Header)

To support both **Web Browsers** (which automatically send cookies) and **Mobile Apps / API Clients** (which send `Authorization: Bearer <token>` headers), our middleware supports dual-source extraction:

1. Check if the `access_token` cookie is present.
2. If missing from cookies, check the `Authorization` HTTP header.
3. If both are missing, reject the request with `401 Unauthorized`.

---

## 2. Implementing `middleware/deserialize-user.go`

Create `middleware/deserialize-user.go`:

```go
package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"golang-gorm-postgres/initializers"
	"golang-gorm-postgres/models"
	"golang-gorm-postgres/utils"
)

// DeserializeUser authenticates requests using RS256 access tokens
func DeserializeUser() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var accessToken string
		cookie, err := ctx.Cookie("access_token")

		authorizationHeader := ctx.Request.Header.Get("Authorization")
		fields := strings.Fields(authorizationHeader)

		// 1. Try extracting token from Authorization: Bearer <token>
		if len(fields) != 0 && fields[0] == "Bearer" {
			accessToken = fields[1]
		} else if err == nil {
			// 2. Fallback to HttpOnly cookie
			accessToken = cookie
		}

		if accessToken == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"status":  "fail",
				"message": "You are not logged in",
			})
			return
		}

		config, _ := initializers.LoadConfig(".")

		// 3. Validate token signature with RS256 Public Key
		sub, err := utils.ValidateToken(accessToken, config.AccessTokenPublicKey)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"status":  "fail",
				"message": err.Error(),
			})
			return
		}

		// 4. Verify user still exists in the database
		var user models.User
		result := initializers.DB.First(&user, "id = ?", fmt.Sprint(sub))
		if result.Error != nil {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"status":  "fail",
				"message": "The user belonging to this token no longer exists",
			})
			return
		}

		// 5. Store authenticated user in Gin Context for downstream handlers
		ctx.Set("currentUser", user)
		ctx.Next()
	}
}
```

---

## 3. How Downstream Handlers Retrieve the User

Any route wrapped with `DeserializeUser()` can access the authenticated user struct directly from the Gin context:

```go
currentUser := ctx.MustGet("currentUser").(models.User)
fmt.Println("Logged in user:", currentUser.Email)
```

- **`ctx.MustGet("currentUser")`**: Retrieves the interface value stored under the key.
- **`.(models.User)`**: Type assertion converting the `interface{}` to the concrete `models.User` struct.

---

## 4. Role-Based Access Control (RBAC) Guard

Want to restrict certain endpoints exclusively to admins? You can easily create an additional middleware guard:

```go
// RestrictTo ensures only users with matching roles can access the endpoint
func RestrictTo(roles ...string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentUser, exists := ctx.Get("currentUser")
		if !exists {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"status": "fail", "message": "Unauthorized"})
			return
		}

		user := currentUser.(models.User)
		roleAllowed := false
		for _, role := range roles {
			if user.Role == role {
				roleAllowed = true
				break
			}
		}

		if !roleAllowed {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"status":  "fail",
				"message": "You are not allowed to perform this action",
			})
			return
		}

		ctx.Next()
	}
}
```
