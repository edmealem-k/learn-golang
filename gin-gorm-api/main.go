package main

import (
	"gin-gorm-api/config"
	"gin-gorm-api/routes"
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	// Set Gin mode based on environment
	// Production mode disables debug output
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Connect to database
	config.ConnectDatabase()

	// Run migrations
	config.RunMigrations()

	// create gin router with default middleware
	// Default includes Logger and recovery middleware
	router := gin.Default()

	// Add cors middleware for frontend access
	router.Use(corsMiddleware())

	// Setup application routes
	routes.SetupRoutes(router)

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
		})
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE,OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
