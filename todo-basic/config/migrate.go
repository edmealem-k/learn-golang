// config/migrate.go
package config

import (
	"log"
	"todo-basic/models"
)

func RunMigration() {
	err := DB.AutoMigrate(
		&models.User{},
		&models.Todo{},
	)
	if err != nil {
		log.Fatal("Migration failed:", err)
	}

	log.Println("Database migration completed")
}
