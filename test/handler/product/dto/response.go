package dto

import "time"

type UserResponse struct {
	ID                    int64        `json:"id"`
	Name                  string       `json:"name"`
	Phone                 string       `json:"phone"`
	Email                 string       `json:"email"`
	Organization          organization `json:"organization"`
	Role                  role         `json:"role"`
	AdditionalPermissions []string     `json:"additional_permissions"`
	CreatedAt             time.Time    `json:"created_at"`
	UpdatedAt             time.Time    `json:"updated_at"`
	payload
	pay Pay `json:"pay"`
}

type payload struct {
	OrganizationID int    `json:"organization_id"`
	Name           string `json:"name"`
}

type Pay struct {
	RoleID string `json:"role_id"`
	Name   string `json:"name"`
}
