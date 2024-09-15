package dbhelp

import (
	"time"
)

// Tender

// Bids

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
