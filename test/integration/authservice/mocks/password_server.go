package mocks

import (
	"context"

	authv1 "github.com/siti-nabila/api-contracts/pb/auth/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/rest-orc/test/shared/fixtures"
)

type PasswordServer struct {
	authv1.UnimplementedAuthServiceServer
}

func (*PasswordServer) ChangePassword(context.Context, *authv1.ChangePasswordRequest) (*authv1.ChangePasswordResponse, error) {
	return &authv1.ChangePasswordResponse{Changed: true, LoginRequired: true}, nil
}

func (*PasswordServer) RefreshSession(context.Context, *authv1.RefreshSessionRequest) (*userv1.UserTokenResponse, error) {
	r := fixtures.LoginTokenResponse()
	r.Token = "refreshed-token"
	r.RefreshToken = "rotated-refresh"
	return r, nil
}
func (*PasswordServer) Logout(context.Context, *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	return &authv1.LogoutResponse{Changed: false}, nil
}
