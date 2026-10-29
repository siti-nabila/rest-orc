package mocks

import (
	"context"

	authv1 "github.com/siti-nabila/api-contracts/pb/auth/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
)

type SessionClient struct {
	RefreshFunc func(context.Context, *authv1.RefreshSessionRequest) (*userv1.UserTokenResponse, error)
	LogoutFunc  func(context.Context, *authv1.LogoutRequest) (*authv1.LogoutResponse, error)
}

func (s *SessionClient) RefreshSession(ctx context.Context, r *authv1.RefreshSessionRequest) (*userv1.UserTokenResponse, error) {
	return s.RefreshFunc(ctx, r)
}
func (s *SessionClient) Logout(ctx context.Context, r *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	return s.LogoutFunc(ctx, r)
}
