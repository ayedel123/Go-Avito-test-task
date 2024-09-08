package main

import (
	"time"

	"github.com/google/uuid"
)

// Tender
type Tender struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid()"`
	Name        string    `json:"name" binding:"required"`
	Description string    `json:"description" binding:"required"`
	Status      string    `json:"status" binding:"required"`
	ServiceType string    `json:"service_type"`
	AuthorID    uuid.UUID `json:"author_id" binding:"required"`
	AuthorType  string    `json:"author_type" binding:"required"`
	Version     int       `json:"version" gorm:"default:1"`
	CreatedAt   time.Time `json:"created_at" gorm:"default:current_timestamp"`
}

// Bids
type Bid struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid()"`
	Name       string    `json:"name" binding:"required"`
	Status     string    `json:"status" binding:"required"`
	AuthorType string    `json:"author_type" binding:"required"`
	AuthorID   uuid.UUID `json:"author_id" binding:"required"`
	TenderID   uuid.UUID `json:"tender_id" binding:"required"`
	Version    int       `json:"version" gorm:"default:1"`
	CreatedAt  time.Time `json:"created_at" gorm:"default:current_timestamp"`
}

// Employee
type Employee struct {
	ID        int       `json:"id"`
	Username  string    `json:"username" db:"username"`
	FirstName string    `json:"first_name" db:"first_name"`
	LastName  string    `json:"last_name" db:"last_name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// OrganizationType
type OrganizationType string

const (
	IE  OrganizationType = "IE"
	LLC OrganizationType = "LLC"
	JSC OrganizationType = "JSC"
)

// Organization
type Organization struct {
	ID          int              `json:"id"`
	Name        string           `json:"name"`
	Description *string          `json:"description,omitempty"`
	Type        OrganizationType `json:"type"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// OrganizationResponsible
type OrganizationResponsible struct {
	ID             int `json:"id"`
	OrganizationID int `json:"organization_id" db:"organization_id"`
	UserID         int `json:"user_id" db:"user_id"`
}
