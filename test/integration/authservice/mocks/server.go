package mocks

import (
	"context"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	usersfixtures "github.com/siti-nabila/rest-orc/test/integration/users/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Server struct {
	userv1.UnimplementedUserServiceServer
}

func (*Server) Register(context.Context, *userv1.AuthRequest) (*userv1.UserTokenResponse, error) {
	return fixtures.RegistrationResponse(), nil
}

func (*Server) Login(context.Context, *userv1.AuthRequest) (*userv1.UserTokenResponse, error) {
	return fixtures.LoginTokenResponse(), nil
}

func (*Server) Me(context.Context, *emptypb.Empty) (*userv1.UserData, error) {
	return usersfixtures.MeResponse(), nil
}

func (*Server) ListUsers(context.Context, *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error) {
	return usersfixtures.ListUsersResponse(), nil
}
