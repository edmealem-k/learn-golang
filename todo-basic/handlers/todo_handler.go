package handlers

import (
	"net/http"
	"strconv"
	"todo-basic/config"
	"todo-basic/models"

	"github.com/gin-gonic/gin"
)

type CreateTodoInput struct {
	Title       string `json:"title" binding:"required,max=200"`
	Description string `json:"description" binding:"required"`
	UserID      uint   `json:"user_id" binding:"required"`
}

type UpdateTodoInput struct {
	Title       *string `json:"title" binding:"omitempty,max=200"`
	Description *string `json:"description"`
	IsCompleted *bool   `json:"is_completed"`
}

func CreateTodo(c *gin.Context) {
	var input CreateTodoInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	var user models.User
	if result := config.DB.First(&user, input.UserID); result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "User not found",
		})
		return
	}

	todo := models.Todo{
		Title:       input.Title,
		Description: input.Description,
		UserID:      input.UserID,
	}

	if result := config.DB.Create(&todo); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create todo",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Todo Created successfully",
		"todo":    todo,
	})
}

func GetAllTodos(c *gin.Context) {
	var todos []models.Todo
	query := config.DB.Model(&models.Todo{})

	if isCompleted := c.Query("is_completed"); isCompleted != "" {
		query = query.Where("is_completed", isCompleted == "true")
	}

	if search := c.Query("search"); search != "" {
		query = query.Where("title ILIKE ?", "%"+search+"%")
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	var total int64
	query.Count(&total)

	result := query.Preload("User").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&todos)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch todos",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"page":    page,
		"limit":   limit,
		"total":   total,
		"results": len(todos),
		"todos":   todos,
	})
}

func GetTodo(c *gin.Context) {
	id := c.Param("id")

	var todo models.Todo

	result := config.DB.Preload("User").First(&todo, id)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "todo not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"todo": todo,
	})
}

func UpdateTodo(c *gin.Context) {
	id := c.Param("id")

	var todo models.Todo
	if result := config.DB.First(&todo, id); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "todo not found",
		})
		return
	}

	var input UpdateTodoInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": err.Error(),
		})
		return
	}

	// Build map or update struct so zero-values (like is_completed=false) are updated properly
	updates := make(map[string]interface{})
	if input.Title != nil {
		updates["title"] = *input.Title
	}
	if input.Description != nil {
		updates["description"] = *input.Description
	}
	if input.IsCompleted != nil {
		updates["is_completed"] = *input.IsCompleted
	}

	if result := config.DB.Model(&todo).Updates(updates); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update todo",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Todo updated successfully",
		"todo":    todo,
	})
}

func DeleteTodo(c *gin.Context) {
	id := c.Param("id")

	result := config.DB.Delete(&models.Todo{}, id)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete todo",
		})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Todo not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Todo deleted successfully",
	})
}

func ToggleTodoStatus(c *gin.Context) {
	id := c.Param("id")

	var todo models.Todo
	if result := config.DB.First(&todo, id); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Todo not found",
		})
		return
	}

	todo.IsCompleted = !todo.IsCompleted
	if result := config.DB.Model(&todo).Update("is_completed", todo.IsCompleted); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to toggle status",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Todo status updated",
		"id":           todo.ID,
		"is_completed": todo.IsCompleted,
	})
}

func GetUserTodos(c *gin.Context) {
	userId := c.Param("id")

	var user models.User
	if result := config.DB.First(&user, userId); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	var todos []models.Todo
	query := config.DB.Where("user_id = ?", userId)

	if status := c.Query("is_completed"); status != "" {
		query = query.Where("is_completed = ?", status == "true")
	}

	result := query.Order("created_at DESC").Find(&todos)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch user todos",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":    user.Username,
		"results": len(todos),
		"todos":   todos,
	})
}

func GetTodoStats(c *gin.Context) {
	var totalCount int64
	var completedCount int64
	var pendingCount int64

	config.DB.Model(&models.Todo{}).Count(&totalCount)
	config.DB.Model(&models.Todo{}).Where("is_completed = ?", true).Count(&completedCount)
	config.DB.Model(&models.Todo{}).Where("is_completed = ?", false).Count(&pendingCount)

	c.JSON(http.StatusOK, gin.H{
		"total":     totalCount,
		"completed": completedCount,
		"pending":   pendingCount,
	})
}

func ClearCompletedTodos(c *gin.Context) {
	result := config.DB.Where("is_completed = ?", true).Delete(&models.Todo{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear completed todos"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Cleared completed todos",
		"deleted": result.RowsAffected,
	})
}
