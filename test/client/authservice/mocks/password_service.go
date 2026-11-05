package mocks

import (
	"context"

	authv1 "github.com/siti-nabila/api-contracts/pb/auth/v1"
	"google.golang.org/grpc"
)

type PasswordService struct {
	ChangeFunc func(context.Context, *authv1.ChangePasswordRequest, ...grpc.CallOption) (*authv1.ChangePasswordResponse, error)
	Calls      int
}

func (service *PasswordService) ChangePassword(ctx context.Context, request *authv1.ChangePasswordRequest, opts ...grpc.CallOption) (*authv1.ChangePasswordResponse, error) {
	service.Calls++
	if service.ChangeFunc == nil {
		panic("unexpected ChangePassword call")
	}
	return service.ChangeFunc(ctx, request, opts...)
}
