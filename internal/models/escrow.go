package models

import (
	"time"
	"gorm.io/gorm"
)

type EscrowStatus string

const (
	EscrowPending   EscrowStatus = "pending"
	EscrowAccepted  EscrowStatus = "accepted"
	EscrowRejected  EscrowStatus = "rejected"
	EscrowCompleted EscrowStatus = "completed"
	EscrowReleased  EscrowStatus = "released"
	EscrowDisputed  EscrowStatus = "disputed"
	EscrowCancelled EscrowStatus = "cancelled"
)

type Escrow struct {
	ID              uint           `gorm:"primarykey" json:"id"`
	BuyerID         uint           `gorm:"not null;index" json:"buyer_id"`
	SellerID        uint           `gorm:"not null;index" json:"seller_id"`
	Items           string         `gorm:"type:text;not null" json:"items"`
	Amount          float64        `gorm:"not null" json:"amount"`
	DeliveryDate    string         `gorm:"not null" json:"delivery_date"`
	AttachedFile    string         `gorm:"type:text" json:"attached_file,omitempty"`
	Status          EscrowStatus   `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
	RejectionReason string         `gorm:"type:text" json:"rejection_reason,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	AcceptedAt      *time.Time     `json:"accepted_at,omitempty"`
	CompletedAt     *time.Time     `json:"completed_at,omitempty"`
	ReleasedAt      *time.Time     `json:"released_at,omitempty"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
	
	Buyer  User `gorm:"foreignKey:BuyerID" json:"buyer,omitempty"`
	Seller User `gorm:"foreignKey:SellerID" json:"seller,omitempty"`
}

func (Escrow) TableName() string {
	return "escrows"
}