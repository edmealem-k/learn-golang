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
			users.GET("/:id/todos", handlers.GetUserTodos)
		}

		todos := v1.Group("/todos")
		{
			todos.POST("", handlers.CreateTodo)
			todos.GET("", handlers.GetAllTodos)
			todos.GET("/stats", handlers.GetTodoStats)
			todos.DELETE("/completed", handlers.ClearCompletedTodos)

			todos.GET("/:id", handlers.GetTodo)
			todos.PATCH("/:id", handlers.UpdateTodo)
			todos.PATCH("/:id/toggle", handlers.ToggleTodoStatus)
			todos.DELETE("/:id", handlers.DeleteTodo)
		}
	}
}
