package mocks

import (
	"context"

	authv1 "github.com/siti-nabila/api-contracts/pb/auth/v1"
)

type PasswordClient struct {
	ChangeFunc func(context.Context, *authv1.ChangePasswordRequest) (*authv1.ChangePasswordResponse, error)
	Calls      int
}

func (client *PasswordClient) ChangePassword(ctx context.Context, request *authv1.ChangePasswordRequest) (*authv1.ChangePasswordResponse, error) {
	client.Calls++
	if client.ChangeFunc == nil {
		panic("unexpected ChangePassword call")
	}
	return client.ChangeFunc(ctx, request)
}
