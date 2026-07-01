package models

import (
	"time"

	"github.com/google/uuid"
)

// ticket Status
type TicketStatus string

const (
	TicketStatusPending TicketStatus = "pending"
	TicketStatusApproved TicketStatus = "approved"
	TicketStatusRejected TicketStatus = "rejected"
	TicketStatusInReview TicketStatus = "in_review"
)

// ticket Type
type TicketType string

const (
	TicketTypeClientOnboarding TicketType = "client_onboarding"
	TicketTypePartnerOnboarding TicketType = "partner_onboarding"
	TicketTypePartnerCarApproval TicketType = "partner_car_approval"
)

type Ticket struct {
	ID          uuid.UUID    `gorm:"type:uuid;primaryKey" json:"id"`
	Type        TicketType   `gorm:"type:text;not null" json:"type"`
	Status      TicketStatus `gorm:"type:text;not null" json:"status"`
	Title       string       `gorm:"not null" json:"title"`
	Description string       `gorm:"type:text" json:"description"`
	RequesterID string       `gorm:"not null" json:"requester_id"`
	ReferenceID string       `gorm:"not null" json:"reference_id"`
	Metadata    string       `gorm:"type:text" json:"metadata"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

func (Ticket) TableName() string {
	return "tickets"
}