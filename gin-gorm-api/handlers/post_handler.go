// handlers/post_handler
package handlers

import (
	"gin-gorm-api/config"
	"gin-gorm-api/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CreatePostInput defines the request body for creating posts
type CreatePostInput struct {
	Title   string   `json:"title" binding:"required,max=200"`
	Content string   `json:"content" binding:"required"`
	UserId  uint     `json:"user_id" binding:"reqquired"`
	Tags    []string `json:"tags"`
}

// CreatePost handles POST /posts
// Creates a post and associates it with tags
func CreatePost(c *gin.Context) {
	var input CreatePostInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// verify the user exists
	var user models.User
	if result := config.DB.First(&user, input.UserId); result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "User not found",
		})
		return
	}

	// Find or create tags
	var tags []models.Tag
	for _, tagName := range input.Tags {
		var tag models.Tag
		// FirstOrCreate finds existing tag or creates new one
		config.DB.FirstOrCreate(&tag, models.Tag{Name: tagName})
		tags = append(tags, tag)
	}

	// Create the post with associated tags
	post := models.Post{
		Title:   input.Title,
		Content: input.Content,
		UserID:  input.UserId,
		Tags:    tags,
	}

	// Create saves the post and creates the many-to-many associations
	result := config.DB.Create(&post)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Faild to create post",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Post Created successfully",
		"post":    post,
	})
}

// GetPost handles GET /posts/:id
// Returns a post with its author and tags preloaded
func GetPost(c *gin.Context) {
	id := c.Param("id")

	var post models.Post

	// Preload fetches related records in separate queries
	// This avoids the N+1 query problem
	result := config.DB.
		Preload("user").
		Preload("tags").
		Preload("Replies").
		Preload("Replies.User").
		First(&post, id)

	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "post not found",
		})
		return
	}

	c.JSON(http.StatusOK, post)
}

// ListPosts handles GET /posts
// Returns posts with filtering and eager loading
func ListPosts(c *gin.Context) {
	var posts []models.Post

	query := config.DB.Model(&models.Post{})

	// Filter by published status
	if published := c.Query("published"); published != "" {
		query = query.Where("published=?", published == "true")
	}
	// Filter by author
	if authorID := c.Query("author_id"); authorID != "" {
		query = query.Where("user_id=?", authorID)
	}
	// Filter by tag
	if tagName := c.Query("tag"); tagName != "" {
		// Use joins for many-to-many filtering
		query = query.Joins("JOIN post_tags ON post_tags.post_id=posts.id").
			Joins("JOIN tags ON tags.id=post_tags.tag_id").
			Where("tags.name=?", tagName)
	}

	// fetch with preloaded relationships
	result := query.Preload("User").Preload("Tags").Order("created_at DESC").Find(&posts)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Faild to fetch posts",
		})
		return
	}

	c.JSON(http.StatusOK, posts)
}
