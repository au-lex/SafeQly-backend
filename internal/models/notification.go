package models

import (
	"time"
	"gorm.io/gorm"
)

type NotificationType string

const (
	NotificationEscrowCreated   NotificationType = "escrow_created"
	NotificationEscrowAccepted  NotificationType = "escrow_accepted"
	NotificationEscrowRejected  NotificationType = "escrow_rejected"
	NotificationEscrowCompleted NotificationType = "escrow_completed"
	NotificationEscrowReleased  NotificationType = "escrow_released"
	NotificationDisputeRaised   NotificationType = "dispute_raised"
	NotificationDisputeResolved NotificationType = "dispute_resolved"
	NotificationDepositSuccess  NotificationType = "deposit_success"
	NotificationWithdrawalSuccess NotificationType = "withdrawal_success"
	NotificationWithdrawalFailed  NotificationType = "withdrawal_failed"
)

type Notification struct {
	ID        uint             `json:"id" gorm:"primaryKey"`
	UserID    uint             `json:"user_id" gorm:"not null;index"`
	Type      NotificationType `json:"type" gorm:"type:varchar(50);not null"`
	Title     string           `json:"title" gorm:"type:varchar(255);not null"`
	Message   string           `json:"message" gorm:"type:text;not null"`
	IsRead    bool             `json:"is_read" gorm:"default:false;index"`
	Data      string           `json:"data" gorm:"type:json"`
	CreatedAt time.Time        `json:"created_at"`
	ReadAt    *time.Time       `json:"read_at"`
	
	// Relationships
	User User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

func (Notification) TableName() string {
	return "notifications"
}

// BeforeCreate hook
func (n *Notification) BeforeCreate(tx *gorm.DB) error {
	n.CreatedAt = time.Now()
	return nil
}