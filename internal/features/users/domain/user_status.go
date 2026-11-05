package domain

import "context"

const UserStatusUpdatePermission = "iam.user.status.update"

type UserStatusActor struct {
	UserID             uint64
	MustChangePassword bool
	Permissions        map[string]bool
}

type ChangeUserStatusBody struct {
	Status *string `json:"status"`
	Reason *string `json:"reason"`
}

type ChangeUserStatusRequest struct {
	UserID         uint64
	Status         *string
	Reason         *string
	Version        uint64
	IdempotencyKey string
}

type ChangeUserStatusResponse struct {
	UserID  uint64 `json:"user_id"`
	Status  string `json:"status"`
	Version string `json:"version"`
	Changed bool   `json:"changed"`
}

type ChangeUserStatusGateway interface {
	ChangeUserStatus(context.Context, ChangeUserStatusRequest) (ChangeUserStatusResponse, error)
}
