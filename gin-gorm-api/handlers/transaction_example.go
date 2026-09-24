// handlers/transaction_example.go
package handlers

import (
	"gin-gorm-api/config"
	"gin-gorm-api/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// TransferPostInput defines the request for transfering post ownership
type TransferPostInput struct {
	PostID    uint `json:"post_id" binding:"required"`
	NewUserID uint `json:"new_user_id" binding:"required"`
}

// TransferPost handles POSt /posts/trasfer
// Transfers ownership of a post to another user atomically
func TransferPost(c *gin.Context) {
	var input TransferPostInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Use a transaction to ensure all operations succeed or fail together
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		// All database operations inside this function use the transaction

		// Find the post
		var post models.Post
		if err := tx.First(&post, input.PostID).Error; err != nil {
			return err
		}

		// Verify the new owner exists
		var newOwner models.User
		if err := tx.First(&newOwner, input.NewUserID).Error; err != nil {
			return err
		}

		// update the post ownership
		if err := tx.Model(&post).Update("user_id", input.NewUserID).Error; err != nil {
			return err
		}

		// Return nil to commit the transaction
		// any error returned will cause a rollback
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Transaction failed" + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Post transferred successfully",
	})
}
