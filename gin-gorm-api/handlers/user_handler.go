// handlers/user_handler.go
package handlers

import (
	"gin-gorm-api/config"
	"gin-gorm-api/models"
	"net/http"

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
