package mocks

import (
	"context"
	sessionv1 "github.com/siti-nabila/api-contracts/pb/session/v1"
)

type SelfSessionClient struct {
	List   func(context.Context, *sessionv1.ListMySessionsRequest) (*sessionv1.ListSessionsResponse, error)
	Revoke func(context.Context, *sessionv1.RevokeMySessionRequest) (*sessionv1.RevokeSessionResponse, error)
	All    func(context.Context) (*sessionv1.RevokeAllMySessionsResponse, error)
}

func (s *SelfSessionClient) ListMySessions(c context.Context, r *sessionv1.ListMySessionsRequest) (*sessionv1.ListSessionsResponse, error) {
	return s.List(c, r)
}
func (s *SelfSessionClient) RevokeMySession(c context.Context, r *sessionv1.RevokeMySessionRequest) (*sessionv1.RevokeSessionResponse, error) {
	return s.Revoke(c, r)
}
func (s *SelfSessionClient) RevokeAllMySessions(c context.Context) (*sessionv1.RevokeAllMySessionsResponse, error) {
	return s.All(c)
}
