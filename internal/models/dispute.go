package models

import (
	"time"
	"gorm.io/gorm"
)

type DisputeStatus string
type DisputeReason string

const (
	DisputeOpen       DisputeStatus = "open"
	DisputeInProgress DisputeStatus = "in_progress"
	DisputeResolved   DisputeStatus = "resolved"
	DisputeClosed     DisputeStatus = "closed"
)

const (
	ReasonNotReceived    DisputeReason = "item_not_received"
	ReasonNotAsDescribed DisputeReason = "item_significantly_not_as_described"
	ReasonDamaged        DisputeReason = "item_arrived_damaged"
	ReasonIncorrectItem  DisputeReason = "incorrect_item_received"
	ReasonOther          DisputeReason = "other"
)

type Dispute struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	EscrowID    uint           `gorm:"not null;index" json:"escrow_id"`
	RaisedBy    uint           `gorm:"not null;index" json:"raised_by"`
	Reason      DisputeReason  `gorm:"type:varchar(50);not null" json:"reason"`
	Description string         `gorm:"type:text;not null" json:"description"`
	Evidence    string         `gorm:"type:text" json:"evidence,omitempty"`
	EvidencePublicID  string `json:"evidence_public_id,omitempty"`
  EvidenceFileName  string `json:"evidence_file_name,omitempty"` 
	Status      DisputeStatus  `gorm:"type:varchar(20);not null;default:'open'" json:"status"`
	Resolution  string         `gorm:"type:text" json:"resolution,omitempty"`
	ResolvedBy  *uint          `gorm:"index" json:"resolved_by,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	ResolvedAt  *time.Time     `json:"resolved_at,omitempty"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	
	Escrow Escrow `gorm:"foreignKey:EscrowID" json:"escrow,omitempty"`
	User   User   `gorm:"foreignKey:RaisedBy" json:"user,omitempty"`
}

func (Dispute) TableName() string {
	return "disputes"
}