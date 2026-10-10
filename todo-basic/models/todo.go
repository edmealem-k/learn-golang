// models/todo.go
package models

import "gorm.io/gorm"

type Todo struct {
	gorm.Model

	Title       string `gorm:"not null;size:200" json:"title"`
	Description string `gorm:"type:text" json:"description"`
	IsCompleted bool   `gorm:"default:false" json:"is_completed"`

	UserID uint `gorm:"not null" json:"user_id"`
	User   User `gorm:"foreignKey:UserID" json:"author,omitempty"`
}
