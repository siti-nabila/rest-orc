package domain

import "context"

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type ChangePasswordResponse struct {
	Changed       bool `json:"changed"`
	LoginRequired bool `json:"login_required"`
}

type ChangePasswordGateway interface {
	ChangePassword(context.Context, ChangePasswordRequest) (ChangePasswordResponse, error)
}
