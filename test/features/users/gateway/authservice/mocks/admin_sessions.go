package mocks

import (
	"context"

	sessionv1 "github.com/siti-nabila/api-contracts/pb/session/v1"
)

type AdminSessionClient struct {
	List   func(context.Context, *sessionv1.ListUserSessionsRequest) (*sessionv1.ListSessionsResponse, error)
	Revoke func(context.Context, *sessionv1.RevokeUserSessionRequest) (*sessionv1.RevokeSessionResponse, error)
	All    func(context.Context, *sessionv1.RevokeAllUserSessionsRequest) (*sessionv1.RevokeAllUserSessionsResponse, error)
}

func (s *AdminSessionClient) ListUserSessions(c context.Context, r *sessionv1.ListUserSessionsRequest) (*sessionv1.ListSessionsResponse, error) {
	return s.List(c, r)
}

func (s *AdminSessionClient) RevokeUserSession(c context.Context, r *sessionv1.RevokeUserSessionRequest) (*sessionv1.RevokeSessionResponse, error) {
	return s.Revoke(c, r)
}

func (s *AdminSessionClient) RevokeAllUserSessions(c context.Context, r *sessionv1.RevokeAllUserSessionsRequest) (*sessionv1.RevokeAllUserSessionsResponse, error) {
	return s.All(c, r)
}
