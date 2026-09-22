// config/migrate.go
package config

import (
	"gin-gorm-api/models"
	"log"
)

// RunMigrations creates or updates database tables
// based on the model struct definition
func RunMigrations() {
	// AutoMigrate creates tables, missing columns, and missing indexes
	// It will not delete unused columns ro protect your data
	err := DB.AutoMigrate(
		&models.User{},
		&models.Post{},
		&models.Tag{},
	)
	if err != nil {
		log.Fatal("Migration failed:", err)
	}

	log.Println("Database migration completed")
}
