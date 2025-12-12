package models

import (
	"time"
	"gorm.io/gorm"
)

type BankAccount struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	UserID        uint           `gorm:"not null;index" json:"user_id"`
	BankName      string         `gorm:"not null" json:"bank_name"`
	AccountNumber string         `gorm:"not null" json:"account_number"`
	AccountName   string         `gorm:"not null" json:"account_name"`
	BankCode      string         `json:"bank_code,omitempty"`
	RecipientCode string         `json:"recipient_code"` 
	IsDefault     bool           `gorm:"default:false" json:"is_default"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	
	// Relationships
	User          User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (BankAccount) TableName() string {
	return "bank_accounts"
}