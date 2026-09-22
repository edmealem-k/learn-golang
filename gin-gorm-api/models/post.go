// models/post.go
package models

import "gorm.io/gorm"

type Post struct {
	gorm.Model

	Title     string `gorm:"not null;size:200" json:"title"`
	Content   string `gorm:"type:text" json:"content"`
	Published bool   `gorm:"default:false" json:"published"`

	// Belongs to relaitonship - GORM will populate this
	// when you use preload to fetch related data
	UserID uint `gorm:"not null" json:"user_id"`
	User   User `gorm:"foreignKey:UserID" json:"author,omitempty"`

	// Self-referencing relationship to reply/thread functionality
	ParentID *uint  `json:"parent_id,omitempty"`
	Parent   *Post  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Replies  []Post `gorm:"foreignKey:ParentID" json:"replies,omitempty"`

	// Many-to-Many relationship through a join table
	Tags []Tag `gorm:"many2many:post_tags;" json:"tags,omitempty"`
}

type Tag struct {
	gorm.Model
	Name  string `gorm:"not null;uniqueIndex;size:50" json:"name"`
	Posts []Post `gorm:"many2many:post_tags;" json:"posts,omitempty"`
}
