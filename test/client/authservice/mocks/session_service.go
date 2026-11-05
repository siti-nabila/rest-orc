package mocks

import (
	"context"

	authv1 "github.com/siti-nabila/api-contracts/pb/auth/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"google.golang.org/grpc"
)

type SessionService struct {
	RefreshFunc func(context.Context, *authv1.RefreshSessionRequest, ...grpc.CallOption) (*userv1.UserTokenResponse, error)
	LogoutFunc  func(context.Context, *authv1.LogoutRequest, ...grpc.CallOption) (*authv1.LogoutResponse, error)
}

func (s *SessionService) RefreshSession(ctx context.Context, r *authv1.RefreshSessionRequest, o ...grpc.CallOption) (*userv1.UserTokenResponse, error) {
	return s.RefreshFunc(ctx, r, o...)
}
func (s *SessionService) Logout(ctx context.Context, r *authv1.LogoutRequest, o ...grpc.CallOption) (*authv1.LogoutResponse, error) {
	return s.LogoutFunc(ctx, r, o...)
}
