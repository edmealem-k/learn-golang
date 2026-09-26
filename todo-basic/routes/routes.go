package routes

import (
	"todo-basic/handlers"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(router *gin.Engine) {
	v1 := router.Group("/api/v1")
	{
		users := v1.Group("/users")
		{
			users.POST("", handlers.CreateUser)
		}

		todos := v1.Group("/todos")
		{
			todos.POST("", handlers.CreateTodo)
			todos.GET("", handlers.GetAllTodos)
			todos.GET("/:id", handlers.GetTodo)
			todos.PATCH("/:id", handlers.UpdateTodo)
			todos.DELETE("/:id", handlers.DeleteTodo)
		}
	}
}
