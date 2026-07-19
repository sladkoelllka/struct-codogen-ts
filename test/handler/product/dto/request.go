package dto

type UserURI struct {
	ID int `uri:"id"`
}

type UserFilter struct {
	Name           *string `form:"name"`
	OrganizationID *int    `form:"organization_id"`
	RoleID         *int    `form:"role_id"`
	Phone          *string `form:"phone"`
	Email          *string `form:"email"`
}

type CreateUserRequest struct {
	Name                  string `json:"name" binding:"required"`
	Phone                 string `json:"phone" binding:"required"`
	Email                 string `json:"email" binding:"required"`
	Password              string `json:"password" binding:"required"`
	OrganizationID        string `json:"organization_id" binding:"required"`
	RoleID                string `json:"role_id" binding:"required"`
	AdditionalPermissions []int  `json:"additional_permissions"`
}

type UpdateUserRequest struct {
	Name                  *string `json:"name"`
	Phone                 *string `json:"phone"`
	Email                 *string `json:"email"`
	OrganizationID        *string `json:"organization_id"`
	Password              *string `json:"password"`
	RoleID                *string `json:"role_id"`
	IsBlocked             *bool   `json:"is_blocked"`
	AdditionalPermissions []int   `json:"additional_permissions"`
}
