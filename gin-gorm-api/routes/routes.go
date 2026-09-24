// Package routes/routes.go
package routes

import (
	"gin-gorm-api/handlers"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all application routes
func SetupRoutes(router *gin.Engine) {
	/// API version group
	v1 := router.Group("/api/v1")
	{
		// User routes
		users := v1.Group("/users")
		{
			users.POST("", handlers.CreatePost)
			users.GET("", handlers.ListUsers)
			users.GET("/:id", handlers.GetUser)
			users.PATCH("/:id", handlers.UpdateUser)
			users.DELETE("/:id", handlers.DeleteUser)
			users.POST("/:id/restore", handlers.RestoreUser)
		}

		// Post routes
		posts := v1.Group("/posts/")
		{
			posts.POST("", handlers.CreatePost)
			posts.GET("", handlers.ListPosts)
			posts.GET("/:id", handlers.GetPost)
			posts.POST("/transfer", handlers.TransferPost)
		}
	}
}
