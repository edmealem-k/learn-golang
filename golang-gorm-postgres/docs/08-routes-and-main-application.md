# 08. Routing, CORS & Main Application Wiring

In this final section, we assemble the pieces:

1. Define modular route groups for authentication, user profiles, and posts.
2. Configure **CORS (Cross-Origin Resource Sharing)** with `AllowCredentials: true` so frontend browsers can send and receive cookies.
3. Wire everything together in `main.go`.
4. Run and test the complete API workflow using `curl`.

---

## 1. Defining Route Groups

### Auth Routes (`routes/auth.routes.go`)

Create `routes/auth.routes.go`:

```go
package routes

import (
	"github.com/gin-gonic/gin"

	"golang-gorm-postgres/controllers"
	"golang-gorm-postgres/middleware"
)

type AuthRouteController struct {
	authController controllers.AuthController
}

func NewAuthRouteController(authController controllers.AuthController) AuthRouteController {
	return AuthRouteController{authController}
}

func (rc *AuthRouteController) AuthRoute(rg *gin.RouterGroup) {
	router := rg.Group("/auth")

	router.POST("/register", rc.authController.SignUpUser)
	router.POST("/login", rc.authController.SignInUser)
	router.GET("/refresh", rc.authController.RefreshAccessToken)
	router.GET("/logout", middleware.DeserializeUser(), rc.authController.LogoutUser)
}
```

---

### User Routes (`routes/user.routes.go`)

Create `routes/user.routes.go`:

```go
package routes

import (
	"github.com/gin-gonic/gin"

	"golang-gorm-postgres/controllers"
	"golang-gorm-postgres/middleware"
)

type UserRouteController struct {
	userController controllers.UserController
}

func NewRouteUserController(userController controllers.UserController) UserRouteController {
	return UserRouteController{userController}
}

func (uc *UserRouteController) UserRoute(rg *gin.RouterGroup) {
	router := rg.Group("/users")

	// Protected: requires valid JWT cookie or Bearer token
	router.GET("/me", middleware.DeserializeUser(), uc.userController.GetMe)
}
```

---

### Post Routes (`routes/post.routes.go`)

Create `routes/post.routes.go`:

```go
package routes

import (
	"github.com/gin-gonic/gin"

	"golang-gorm-postgres/controllers"
	"golang-gorm-postgres/middleware"
)

type PostRouteController struct {
	postController controllers.PostController
}

func NewPostRouteController(postController controllers.PostController) PostRouteController {
	return PostRouteController{postController}
}

func (pc *PostRouteController) PostRoute(rg *gin.RouterGroup) {
	router := rg.Group("/posts")

	// Apply DeserializeUser guard to all post routes
	router.Use(middleware.DeserializeUser())

	router.POST("/", pc.postController.CreatePost)
	router.GET("/", pc.postController.FindPosts)
	router.GET("/:postId", pc.postController.FindPostById)
	router.PUT("/:postId", pc.postController.UpdatePost)
	router.DELETE("/:postId", pc.postController.DeletePost)
}
```

---

## 2. Server Entry Point (`main.go`)

Create `main.go` in your project root:

```go
package main

import (
	"log"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"golang-gorm-postgres/controllers"
	"golang-gorm-postgres/initializers"
	"golang-gorm-postgres/routes"
)

var (
	server              *gin.Engine
	AuthController      controllers.AuthController
	AuthRouteController routes.AuthRouteController

	UserController      controllers.UserController
	UserRouteController routes.UserRouteController

	PostController      controllers.PostController
	PostRouteController routes.PostRouteController
)

func init() {
	config, err := initializers.LoadConfig(".")
	if err != nil {
		log.Fatal(" Could not load environment variables: ", err)
	}

	initializers.ConnectDB(&config)

	// Initialize Controllers
	AuthController = controllers.NewAuthController(initializers.DB)
	AuthRouteController = routes.NewAuthRouteController(AuthController)

	UserController = controllers.NewUserController(initializers.DB)
	UserRouteController = routes.NewRouteUserController(UserController)

	PostController = controllers.NewPostController(initializers.DB)
	PostRouteController = routes.NewPostRouteController(PostController)

	server = gin.Default()
}

func main() {
	config, err := initializers.LoadConfig(".")
	if err != nil {
		log.Fatal(" Could not load environment variables: ", err)
	}

	// 1. Configure CORS
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"http://localhost:3000", config.ClientOrigin}
	corsConfig.AllowCredentials = true
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	server.Use(cors.New(corsConfig))

	// 2. Health Checker Endpoint
	router := server.Group("/api")
	router.GET("/healthchecker", func(ctx *gin.Context) {
		message := "Welcome to Golang with GORM and Postgres"
		ctx.JSON(http.StatusOK, gin.H{"status": "success", "message": message})
	})

	// 3. Mount Routes
	AuthRouteController.AuthRoute(router)
	UserRouteController.UserRoute(router)
	PostRouteController.PostRoute(router)

	// 4. Start HTTP Server
	log.Printf(" Server running on port %s", config.ServerPort)
	log.Fatal(server.Run(":" + config.ServerPort))
}
```

### Why `AllowCredentials = true` in CORS?

When a browser frontend (e.g. Next.js/React at `http://localhost:3000`) communicates with your backend at `http://localhost:8000`:

- Browsers will **refuse** to store or send `HttpOnly` cookies unless `AllowCredentials: true` is explicitly configured in CORS headers on the server!
- With this enabled, cookies work seamlessly across domains.

---

## 3. Testing the Complete Flow with `curl`

### 1. Register a User

```bash
curl -X POST http://localhost:8000/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Jane Doe",
    "email": "jane@example.com",
    "password": "Password123!",
    "passwordConfirm": "Password123!"
  }'
```

### 2. Login & Save Cookies to a Jar

```bash
curl -X POST http://localhost:8000/api/auth/login \
  -c cookies.txt \
  -H "Content-Type: application/json" \
  -d '{
    "email": "jane@example.com",
    "password": "Password123!"
  }'
```

_(Notice the `access_token` and `refresh_token` stored inside `cookies.txt`)_

### 3. Access Protected Profile (`/api/users/me`) using Cookies

```bash
curl -X GET http://localhost:8000/api/users/me -b cookies.txt
```

### 4. Create a Post

```bash
curl -X POST http://localhost:8000/api/posts/ \
  -b cookies.txt \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Mastering Go and GORM",
    "content": "Building secure production REST APIs in Go is fun!",
    "image": "https://example.com/cover.png"
  }'
```

### 5. Fetch Paginated Posts

```bash
curl -X GET "http://localhost:8000/api/posts/?page=1&limit=5" -b cookies.txt
```

### 6. Refresh Expired Access Token

```bash
curl -X GET http://localhost:8000/api/auth/refresh -b cookies.txt -c cookies.txt
```

### 7. Logout

```bash
curl -X GET http://localhost:8000/api/auth/logout -b cookies.txt -c cookies.txt
```
