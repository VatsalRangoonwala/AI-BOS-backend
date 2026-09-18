package businesses

import (
	"errors"
	"time"
)

var (
	ErrBusinessNotFound    = errors.New("business not found or inaccessible")
	ErrNotBusinessMember   = errors.New("you are not an active member of this business")
	ErrUnauthorizedAction  = errors.New("you do not have permission to perform this action in this business")
	ErrMemberAlreadyExists = errors.New("user is already a member or invited to this business")
	ErrMemberNotFound      = errors.New("member not found in this business")
	ErrCannotRemoveOwner   = errors.New("the business owner cannot be removed")
)

type Business struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	LegalName       *string   `json:"legalName"`
	Phone           *string   `json:"phone"`
	Address         *string   `json:"address"`
	Timezone        string    `json:"timezone"`
	Currency        string    `json:"currency"`
	InvoicePrefix   string    `json:"invoicePrefix"`
	InvoiceSequence int       `json:"invoiceSequence"`
	Status          string    `json:"status"` // "active", "suspended"
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type Membership struct {
	ID         string     `json:"id"`
	BusinessID string     `json:"businessId"`
	UserID     string     `json:"userId"`
	Role       string     `json:"role"`   // "owner", "staff", "admin"
	Status     string     `json:"status"` // "active", "invited", "suspended"
	InvitedAt  *time.Time `json:"invitedAt"`
	AcceptedAt *time.Time `json:"acceptedAt"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

type UserSummary struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	FullName string `json:"fullName"`
	Status   string `json:"status"`
}

type MemberDetail struct {
	ID         string      `json:"id"`
	BusinessID string      `json:"businessId"`
	UserID     string      `json:"userId"`
	Role       string      `json:"role"`
	Status     string      `json:"status"`
	User       UserSummary `json:"user"`
	CreatedAt  time.Time   `json:"createdAt"`
	UpdatedAt  time.Time   `json:"updatedAt"`
}

type UserBusinessSummary struct {
	BusinessID   string `json:"businessId"`
	BusinessName string `json:"businessName"`
	Role         string `json:"role"`
	Status       string `json:"status"`
}
