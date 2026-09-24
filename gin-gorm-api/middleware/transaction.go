// Package middleware/transaction.go
package middleware

import (
	"gin-gorm-api/config"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// TransactionMiddleware wraps the request in a database transaction
// on error or panic, the transaction is rolled back
func TransactionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// start a transaction
		tx := config.DB.Begin()

		if tx.Error != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "failed to start transaction",
			})
			return
		}

		// handle panics with rollback
		c.Set("tx", tx)

		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
				panic(r)
			}
		}()

		// process request
		c.Next()

		// Check for errors and rollback ro commit
		if c.Writer.Status() >= 400 {
			tx.Rollback()
		} else {
			if err := tx.Commit().Error; err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "Failed to commit transaction",
				})
			}
		}
	}
}

// GetTx retrieves the transaction from context
func GetTx(c *gin.Context) *gorm.DB {
	if tx, exists := c.Get("tx"); exists {
		return tx.(*gorm.DB)
	}
	return config.DB
}
