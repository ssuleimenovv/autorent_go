package database

import (
	"fmt"
	"log"

	"autorent-backend/shared/identity-service/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Migrate runs database migrations
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.SubjectType{},
		&models.ActorType{},
		&models.User{},
		&models.Role{},
		&models.Permission{},
		&models.RefreshToken{},
		&models.ActivationToken{},
		&models.UserProvisionRequest{},
		&models.UserRole{},
		&models.RolePermission{},
		&models.RoleInheritance{},
	)
}

// SeedDefaultData seeds initial data
func SeedDefaultData(db *gorm.DB) error {
	// Seed SubjectTypes
	subjectTypes := []models.SubjectType{
		{ID: uuid.New(), Name: "user", Description: "Individual user"},
		{ID: uuid.New(), Name: "service", Description: "Service account"},
		{ID: uuid.New(), Name: "api_key", Description: "API key"},
		{ID: uuid.New(), Name: "system", Description: "System account"},
	}

	for _, st := range subjectTypes {
		if err := db.Where(models.SubjectType{Name: st.Name}).Assign(models.SubjectType{ID: st.ID, Name: st.Name, Description: st.Description}).FirstOrCreate(&models.SubjectType{}).Error; err != nil {
			return fmt.Errorf("failed to seed subject types: %w", err)
		}
	}

	// Seed ActorTypes
	actorTypes := []models.ActorType{
		{ID: uuid.New(), Name: "client", Description: "Client user"},
		{ID: uuid.New(), Name: "partner", Description: "Partner user"},
		{ID: uuid.New(), Name: "admin", Description: "Administrator"},
		{ID: uuid.New(), Name: "internal", Description: "Internal service"},
	}

	for _, at := range actorTypes {
		if err := db.Where(models.ActorType{Name: at.Name}).Assign(models.ActorType{ID: at.ID, Name: at.Name, Description: at.Description}).FirstOrCreate(&models.ActorType{}).Error; err != nil {
			return fmt.Errorf("failed to seed actor types: %w", err)
		}
	}

	log.Println("Database seeded successfully")
	return nil
}
