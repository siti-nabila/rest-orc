package mocks

import (
	"context"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Service struct {
	MeFunc func(
		context.Context,
		*emptypb.Empty,
		...grpc.CallOption,
	) (*userv1.UserData, error)
	ListUsersFunc func(
		context.Context,
		*userv1.ListUsersRequest,
		...grpc.CallOption,
	) (*userv1.ListUsersResponse, error)
	RegisterFunc func(
		context.Context,
		*userv1.AuthRequest,
		...grpc.CallOption,
	) (*userv1.UserTokenResponse, error)
	LoginFunc func(
		context.Context,
		*userv1.AuthRequest,
		...grpc.CallOption,
	) (*userv1.UserTokenResponse, error)
}

func (service *Service) Register(
	ctx context.Context,
	request *userv1.AuthRequest,
	options ...grpc.CallOption,
) (*userv1.UserTokenResponse, error) {
	return service.RegisterFunc(ctx, request, options...)
}

func (service *Service) Login(
	ctx context.Context,
	request *userv1.AuthRequest,
	options ...grpc.CallOption,
) (*userv1.UserTokenResponse, error) {
	return service.LoginFunc(ctx, request, options...)
}

func (service *Service) Me(
	ctx context.Context,
	request *emptypb.Empty,
	options ...grpc.CallOption,
) (*userv1.UserData, error) {
	return service.MeFunc(ctx, request, options...)
}

func (service *Service) ListUsers(
	ctx context.Context,
	request *userv1.ListUsersRequest,
	options ...grpc.CallOption,
) (*userv1.ListUsersResponse, error) {
	return service.ListUsersFunc(ctx, request, options...)
}
