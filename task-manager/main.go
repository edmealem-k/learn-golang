package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Task struct {
	gorm.Model
	ID          int
	Title       string
	Description string
	Status      string
}

type User struct {
	gorm.Model
	ID    string
	Name  string
	Tasks []Task
}

var users = make([]User, 0)

func main() {
	db, err := gorm.Open(sqlite.Open("task-manger.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect to database!")
	}
	
	// Migrate the schema
	db.AutoMigrate(&Task{}, &User{})
	
	// Create a Gin router with a default middleware (loggers and recovery)
	r := gin.Default()

	// define a simple GET endpoint
	r.GET("/ping", handlePing)

	// Start server on port 8080 (default)
	r.Run(":3005")
}

func handlePing(c *gin.Context) {
	fmt.Println("/ping endpoint")

	c.JSON(http.StatusOK, gin.H{
		"message": "PONG",
	})
}

// func handleGetuser(c *gin.Context) {
// 	name := c.Param("name")
// 	c.JSON(http.StatusOK, gin.H{
// 		"name": name,
// 	})
// }

// func handleWelcom(c *gin.Context) {
// 	firstname := c.DefaultQuery("firstname", "Guest")
// 	lastname := c.Query("lastname")
// 	c.JSON(http.StatusOK, gin.H{
// 		"firstname": firstname,
// 		"lastname":  lastname,
// 	})
// }
