package database

import (
    "fmt"
    "log"
    
    "SafeQly/internal/models"
)


func Migrate() error {
    log.Println("Running database migrations...")
    
    log.Printf("Attempting to migrate database models: User and PendingUser")
    err := DB.AutoMigrate(
        &models.User{},
        &models.PendingUser{},
        &models.Escrow{},
		&models.Dispute{},
		&models.Transaction{},  
		&models.BankAccount{},
    )
    
    if err != nil {
        log.Printf("Error migrating database: %v", err)
        return fmt.Errorf("failed to migrate database: %w", err)
    }
    
    log.Println("Database migration completed successfully")
    return nil
}