# 07. User & Post CRUD Controllers

In this section, we create the business logic handlers for managing authenticated user profiles and performing paginated CRUD operations on posts.

---

## 1. User Profile Handler (`controllers/user.controller.go`)

Create `controllers/user.controller.go` to provide the current logged-in user profile (`/api/users/me`).

```go
package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"golang-gorm-postgres/models"
)

type UserController struct {
	DB *gorm.DB
}

func NewUserController(DB *gorm.DB) UserController {
	return UserController{DB}
}

// GetMe returns the profile of the currently authenticated user
func (uc *UserController) GetMe(ctx *gin.Context) {
	currentUser := ctx.MustGet("currentUser").(models.User)

	userResponse := models.UserResponse{
		ID:        *currentUser.ID,
		Name:      currentUser.Name,
		Email:     currentUser.Email,
		Role:      currentUser.Role,
		Photo:     currentUser.Photo,
		Provider:  currentUser.Provider,
		CreatedAt: currentUser.CreatedAt,
		UpdatedAt: currentUser.UpdatedAt,
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "success", "data": gin.H{"user": userResponse}})
}
```

---

## 2. Post CRUD Controller (`controllers/post.controller.go`)

Create `controllers/post.controller.go` to implement full Create, Read, Update, Delete with pagination support.

```go
package controllers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"golang-gorm-postgres/models"
)

type PostController struct {
	DB *gorm.DB
}

func NewPostController(DB *gorm.DB) PostController {
	return PostController{DB}
}

// CreatePost creates a new post belonging to the authenticated user
func (pc *PostController) CreatePost(ctx *gin.Context) {
	currentUser := ctx.MustGet("currentUser").(models.User)
	var payload *models.CreatePostInput

	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, err.Error())
		return
	}

	now := time.Now()
	newPost := models.Post{
		Title:     payload.Title,
		Content:   payload.Content,
		Image:     payload.Image,
		User:      *currentUser.ID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	result := pc.DB.Create(&newPost)
	if result.Error != nil {
		if strings.Contains(result.Error.Error(), "duplicate key value violates unique constraint") {
			ctx.JSON(http.StatusConflict, gin.H{"status": "fail", "message": "Post with that title already exists"})
			return
		}
		ctx.JSON(http.StatusBadGateway, gin.H{"status": "error", "message": result.Error.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"status": "success", "data": newPost})
}

// FindPosts returns a paginated list of posts
func (pc *PostController) FindPosts(ctx *gin.Context) {
	var page = ctx.DefaultQuery("page", "1")
	var limit = ctx.DefaultQuery("limit", "10")

	intPage, _ := strconv.Atoi(page)
	intLimit, _ := strconv.Atoi(limit)
	offset := (intPage - 1) * intLimit

	var posts []models.Post
	results := pc.DB.Limit(intLimit).Offset(offset).Find(&posts)
	if results.Error != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"status": "error", "message": results.Error.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "success", "results": len(posts), "data": posts})
}

// FindPostById retrieves a single post by its UUID
func (pc *PostController) FindPostById(ctx *gin.Context) {
	postId := ctx.Param("postId")

	var post models.Post
	result := pc.DB.First(&post, "id = ?", postId)
	if result.Error != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"status": "fail", "message": "No post with that title exists"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "success", "data": post})
}

// UpdatePost modifies an existing post
func (pc *PostController) UpdatePost(ctx *gin.Context) {
	postId := ctx.Param("postId")
	var payload *models.UpdatePostInput

	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"status": "fail", "message": err.Error()})
		return
	}

	var post models.Post
	result := pc.DB.First(&post, "id = ?", postId)
	if result.Error != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"status": "fail", "message": "No post with that id exists"})
		return
	}

	postToUpdate := models.Post{
		Title:     payload.Title,
		Content:   payload.Content,
		Image:     payload.Image,
		User:      post.User,
		CreatedAt: post.CreatedAt,
		UpdatedAt: time.Now(),
	}

	pc.DB.Model(&post).Updates(postToUpdate)

	ctx.JSON(http.StatusOK, gin.H{"status": "success", "data": post})
}

// DeletePost removes a post by ID
func (pc *PostController) DeletePost(ctx *gin.Context) {
	postId := ctx.Param("postId")

	result := pc.DB.Delete(&models.Post{}, "id = ?", postId)
	if result.Error != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"status": "error", "message": result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		ctx.JSON(http.StatusNotFound, gin.H{"status": "fail", "message": "No post with that id exists"})
		return
	}

	ctx.JSON(http.StatusNoContent, nil)
}
```

---

## 3. Explaining Pagination Logic

```go
intPage, _ := strconv.Atoi(page)
intLimit, _ := strconv.Atoi(limit)
offset := (intPage - 1) * intLimit
pc.DB.Limit(intLimit).Offset(offset).Find(&posts)
```

- When `page = 1` and `limit = 10`: `offset = (1-1)*10 = 0` (fetches rows 1–10).
- When `page = 2` and `limit = 10`: `offset = (2-1)*10 = 10` (fetches rows 11–20).
- GORM generates `SELECT * FROM posts LIMIT 10 OFFSET 10;`.
