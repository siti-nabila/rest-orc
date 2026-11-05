package domain

import "context"

const UserUpdatePermission = "iam.user.update"

type UserProfileActor struct {
	UserID             uint64
	MustChangePassword bool
	Permissions        map[string]bool
}

type UpdateUserProfileRequest struct {
	UserID         uint64
	Version        uint64
	IdempotencyKey *string
	Profile        UpdateProfileRequest
}

type UpdateUserProfileResponse struct {
	Changed bool        `json:"changed"`
	User    *UserDetail `json:"user,omitempty"`
}

type UpdateUserProfileGateway interface {
	UpdateUserProfile(context.Context, UpdateUserProfileRequest) (UpdateUserProfileResponse, error)
}
