// handlers/user_handler.go
package handlers

import (
	"gin-gorm-api/config"
	"gin-gorm-api/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// CreateUserInput defines the expected request body
// Using a separate struct allows for validation without exposing all models fields
type CreateUserInput struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Bio      string `json:"bio" binding:"max=500"`
}

// UpdateUserInput defines fields that can be updated
// All fields are optional for partial updates
type UpdateUserInput struct {
	Username *string `json:"username" binding:"omitempty,min=3,max=50"`
	Email    *string `json:"email" binding:"omitempty,email"`
	Bio      *string `json:"bio" binding:"omitempty,max=500"`
	IsActive *bool   `json:"is_active"`
}

// CreateUser handler POST /users
// It creates a new user with a hashed password
func CreateUser(c *gin.Context) {
	var input CreateUserInput

	// Bind and validate the json req body
	// if validation fails, return a 400 repsonse with the error details
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": err.Error(),
		})
		return
	}

	// Hash the password before storing
	// Never store plain text passwords in the database
	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(input.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process password",
		})
		return
	}

	// Create the user model
	user := models.User{
		Username: input.Username,
		Email:    input.Email,
		Password: string(hashedPassword),
	}

	// Handle optional bio field
	if input.Bio != "" {
		user.Bio = &input.Bio
	}

	// Insert into database
	// GORM handles the sql insert statement
	result := config.DB.Create(&user)
	if result.Error != nil {
		// Check for unique constraint violations
		c.JSON(http.StatusConflict, gin.H{
			"error": "username or email already exists",
		})
		return
	}

	// Return the created user (password is hidden via json:"-" tag)
	c.JSON(http.StatusCreated, gin.H{
		"message": "user created successfully",
		"user":    user,
	})
}

// GetUser handles GET /users/:id
// Retrieves a single user by ID
func GetUser(c *gin.Context) {
	id := c.Param("id")

	var user models.User

	result := config.DB.First(&user, id)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, user)
}

// ListUsers handles GET /users
// Returns a paginated list of users with optional filtering
func ListUsers(c *gin.Context) {
	var users []models.User

	// Parse query parameters for pagination
	// Default: page 1, 10 items per page
	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "10")

	// convert to integers for gorm and keep values in a safe range
	pageNum, err := strconv.Atoi(page)
	if err != nil || pageNum < 1 {
		pageNum = 1
	}

	limitNum, err := strconv.Atoi(limit)
	if err != nil || limitNum < 1 || limitNum > 100 {
		limitNum = 10
	}

	// calculate offset for pagination
	offset := (pageNum - 1) * limitNum

	// Build query with optional filters
	query := config.DB.Model(&models.User{})

	// filter by active status if provided
	if active := c.Query("active"); active != "" {
		query = query.Where("is_active=?", active == "true")
	}

	// search by username if provided
	if search := c.Query("search"); search != "" {
		// use ILIKE for case-insesitive search (postgresql)
		query = query.Where("username ILIKE ?", "%"+search+"%")
	}

	// get total count for pagination metadata
	var total int64
	query.Count(&total)

	// fetch paginated results
	result := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(limitNum).
		Find(&users)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch users",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"pagination": gin.H{
			"page":        pageNum,
			"limit":       limitNum,
			"total":       total,
			"total_pages": (total + int64(limitNum) - 1) / int64(limitNum),
		},
	})
}

// UpdateUser handles PATCH /users/:id
// supports partial updates - only provided fields are changed
func UpdateUser(c *gin.Context) {
	id := c.Param("id")

	// first, find the existing user
	var user models.User
	if result := config.DB.First(&user, id); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "user not found",
		})
		return
	}

	// bind the update input
	var input UpdateUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation failed",
			"details": err.Error(),
		})
		return
	}

	// build a map of fields to update
	// This allows for partial updates without overwriting with zero values
	updates := make(map[string]any)

	if input.Username != nil {
		updates["username"] = *input.Username
	}
	if input.Email != nil {
		updates["email"] = *input.Email
	}
	if input.Bio != nil {
		updates["bio"] = *input.Bio
	}
	if input.IsActive != nil {
		updates["is_active"] = *input.IsActive
	}

	// perform the update
	// updates only modifies the special fields
	result := config.DB.Model(&user).Updates(updates)
	if result.Error != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "update failed - username or email may already exist",
		})
		return
	}

	// refresh the user data to return updated values
	config.DB.First(&user, id)

	c.JSON(http.StatusOK, gin.H{
		"message": "user updated successfully",
		"user":    user,
	})
}

// DeleteUser handles delete /user/:id
// performs a soft delete by default (sets deleted_at timestamp)
func DeleteUser(c *gin.Context) {
	id := c.Param("id")

	// check if user exists
	var user models.User
	if result := config.DB.First(&user, id); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "user not found",
		})
		return
	}

	// check for hard delete flag
	hardDelete := c.Query("hard") == "true"

	if hardDelete {
		// permanently delete the record
		// use unscoped to bypass soft delete behavior
		result := config.DB.Unscoped().Delete(&user)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to delete user",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "User permanently deleted",
		})
	} else {
		// soft delete - sets deleted_at timestamp
		// record remains in database but is exluded from queries
		result := config.DB.Delete(&user)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to delete user",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "user deleted successfully",
		})
	}
}

// RestoreUser handles post /user/:id/restore
// restores a soft-deleted user
func RestoreUser(c *gin.Context) {
	id := c.Param("id")

	// use unscoped to find soft-deleted records
	var user models.User
	result := config.DB.Unscoped().First(&user, id)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "user not found",
		})
		return
	}

	// check if the user was actually deleted
	if !user.DeletedAt.Valid {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "user is not deleted",
		})
		return
	}

	// clear the deleted_at field to restore
	config.DB.Unscoped().Model(&user).Update("deleted_at", nil)

	c.JSON(http.StatusOK, gin.H{
		"message": "user restored successfully",
		"user":    user,
	})
}
