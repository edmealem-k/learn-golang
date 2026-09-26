package models

import "gorm.io/gorm"

type User struct {
	gorm.Model

	Username string `gorm:"uniqueIndex;not null;size:50" json:"username"`
	Email    string `gorm:"uniqueIndex;not null;size:100" json:"email"`
	Password string `gorm:"not null" json:"-"`
}
