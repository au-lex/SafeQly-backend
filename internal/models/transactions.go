package models

import (
	"time"
	"gorm.io/gorm"
)

type TransactionType string
type TransactionStatus string

const (
	TransactionDeposit    TransactionType = "deposit"
	TransactionWithdrawal TransactionType = "withdrawal"
	TransactionEscrow     TransactionType = "escrow"
	TransactionRefund     TransactionType = "refund"
	TransactionRelease    TransactionType = "release"
)

const (
	TransactionPending   TransactionStatus = "pending"
	TransactionCompleted TransactionStatus = "completed"
	TransactionFailed    TransactionStatus = "failed"
	TransactionCancelled TransactionStatus = "cancelled"
)

type Transaction struct {
	ID              uint              `gorm:"primarykey" json:"id"`
	UserID          uint              `gorm:"not null;index" json:"user_id"`
	Type            TransactionType   `gorm:"type:varchar(20);not null" json:"type"`
	Amount          float64           `gorm:"not null" json:"amount"`
	Status          TransactionStatus `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
	Reference       string            `gorm:"uniqueIndex;not null" json:"reference"`
	Description     string            `gorm:"type:text" json:"description"`
	PaymentMethod   string            `gorm:"type:varchar(50)" json:"payment_method,omitempty"`
	PaymentProvider string            `gorm:"type:varchar(50)" json:"payment_provider,omitempty"`
	BankName        string            `json:"bank_name,omitempty"`
	AccountNumber   string            `json:"account_number,omitempty"`
	AccountName     string            `json:"account_name,omitempty"`
	CompletedAt     *time.Time        `json:"completed_at,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	DeletedAt       gorm.DeletedAt    `gorm:"index" json:"-"`
	
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (Transaction) TableName() string {
	return "transactions"
}