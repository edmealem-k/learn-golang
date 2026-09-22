package models

import (
	"time"

	"gorm.io/gorm"
)

// User represents the users table in the database
// GORM uses struct tags to define column properties
type User struct {
	// gorm.Model embeds ID, CreatedAt, UpdatedAt, and DeletedAt fields
	gorm.Model

	// Username must be uniqe and cannot be null
	// The `gorm` tag defines database constraints
	Username string `gorm:"uniqueIndex;not null;size:50" json:"username"`
	Email    string `gorm:"uniqueIndex;not null;size:100" json:"email"`
	// Password is stored but never returned in json responses
	// the `json:"-"` tag hides this field from api responses
	Password string `gorm:"not null" json:"-"`
	// optional fields use pointers to allow null values
	Bio *string `gorm:"default:true" json:"bio,omitempty"`
	// active status with a default value
	IsActive bool `gorm:"default:true" json:"is_active"`
	// Custom timestampt fields
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}

// TableName overrides the default table name
// By default, GORM would use "users" (pluralized struct name)
func (User) TableName() string {
	return "users"
}
