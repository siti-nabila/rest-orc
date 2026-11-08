package domain

import "context"

const UserReadPermission = "iam.user.read"

type UserDetailActor struct {
	UserID             uint64
	MustChangePassword bool
	Permissions        map[string]bool
}

type UserDetail struct {
	ID                 uint64                        `json:"id"`
	Email              string                        `json:"email"`
	Name               string                        `json:"name"`
	Address            string                        `json:"address"`
	Phone              string                        `json:"phone"`
	CountryCode        string                        `json:"country_code"`
	Status             string                        `json:"status"`
	Version            string                        `json:"version"`
	CreatedAt          string                        `json:"created_at"`
	MustChangePassword bool                          `json:"must_change_password"`
	Roles              []UserDetailRole              `json:"roles"`
	ApplicationAccess  []UserDetailApplicationAccess `json:"application_access"`
}

type UserDetailRole struct {
	RoleID     uint64 `json:"role_id"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	RoleStatus string `json:"role_status"`
}

type UserDetailApplicationAccess struct {
	ApplicationID     uint64 `json:"application_id"`
	Code              string `json:"code"`
	Name              string `json:"name"`
	Status            string `json:"status"`
	ApplicationStatus string `json:"application_status"`
}

type GetUserGateway interface {
	GetUser(context.Context, uint64) (UserDetail, error)
}
