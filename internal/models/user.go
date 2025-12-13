package models

import (
	"time"
	"gorm.io/gorm"
)

type User struct {
	ID                uint           `gorm:"primarykey" json:"id"`
	FullName          string         `gorm:"not null" json:"full_name"`
	Email             string         `gorm:"uniqueIndex;not null" json:"email"`
	Phone             string         `gorm:"not null" json:"phone"`
	Password          string         `gorm:"not null" json:"-"`
	UserTag           string         `gorm:"uniqueIndex;not null" json:"user_tag"`
	Avatar            string         `gorm:"type:text" json:"avatar,omitempty"`
	Balance           float64        `gorm:"default:0" json:"balance"`
	EscrowBalance     float64        `gorm:"default:0" json:"escrow_balance"` 
	IsEmailVerified   bool           `gorm:"default:false" json:"is_email_verified"`
	OTP               string         `gorm:"index" json:"-"`
	OTPExpiry         *time.Time     `json:"-"`
	ResetToken        string         `gorm:"index" json:"-"`
	ResetTokenExpiry  *time.Time     `json:"-"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string {
	return "users"
}

type PendingUser struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	FullName  string    `gorm:"not null" json:"full_name"`
	Email     string    `gorm:"uniqueIndex;not null" json:"email"`
	Phone     string    `gorm:"not null" json:"phone"`
	Password  string    `gorm:"not null" json:"-"`
	OTP       string    `gorm:"not null" json:"-"`
	OTPExpiry time.Time `gorm:"not null" json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

func (PendingUser) TableName() string {
	return "pending_users"
}