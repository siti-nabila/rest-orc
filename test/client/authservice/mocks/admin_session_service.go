package mocks

import (
	"context"

	sessionv1 "github.com/siti-nabila/api-contracts/pb/session/v1"
	"google.golang.org/grpc"
)

type AdminSessionService struct {
	List   func(context.Context, *sessionv1.ListUserSessionsRequest) (*sessionv1.ListSessionsResponse, error)
	Revoke func(context.Context, *sessionv1.RevokeUserSessionRequest) (*sessionv1.RevokeSessionResponse, error)
	All    func(context.Context, *sessionv1.RevokeAllUserSessionsRequest) (*sessionv1.RevokeAllUserSessionsResponse, error)
}

func (s *AdminSessionService) ListUserSessions(c context.Context, r *sessionv1.ListUserSessionsRequest, _ ...grpc.CallOption) (*sessionv1.ListSessionsResponse, error) {
	return s.List(c, r)
}

func (s *AdminSessionService) RevokeUserSession(c context.Context, r *sessionv1.RevokeUserSessionRequest, _ ...grpc.CallOption) (*sessionv1.RevokeSessionResponse, error) {
	return s.Revoke(c, r)
}

func (s *AdminSessionService) RevokeAllUserSessions(c context.Context, r *sessionv1.RevokeAllUserSessionsRequest, _ ...grpc.CallOption) (*sessionv1.RevokeAllUserSessionsResponse, error) {
	return s.All(c, r)
}
