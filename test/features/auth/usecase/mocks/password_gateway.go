package mocks

import (
	"context"

	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
)

type PasswordGateway struct {
	ChangeFunc func(context.Context, domain.ChangePasswordRequest) (domain.ChangePasswordResponse, error)
	Calls      int
}

func (gateway *PasswordGateway) ChangePassword(ctx context.Context, request domain.ChangePasswordRequest) (domain.ChangePasswordResponse, error) {
	gateway.Calls++
	if gateway.ChangeFunc == nil {
		panic("unexpected ChangePassword call")
	}
	return gateway.ChangeFunc(ctx, request)
}
