package dto

type organization struct {
	OrganizationID int    `json:"organization_id"`
	Name           string `json:"name"`
}

type role struct {
	RoleID string `json:"role_id"`
	Name   string `json:"name"`
}
