package mocks

import (
	"context"
	sessionv1 "github.com/siti-nabila/api-contracts/pb/session/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type SelfSessionService struct {
	List   func(context.Context, *sessionv1.ListMySessionsRequest) (*sessionv1.ListSessionsResponse, error)
	Revoke func(context.Context, *sessionv1.RevokeMySessionRequest) (*sessionv1.RevokeSessionResponse, error)
	All    func(context.Context, *emptypb.Empty) (*sessionv1.RevokeAllMySessionsResponse, error)
}

func (s *SelfSessionService) ListMySessions(c context.Context, r *sessionv1.ListMySessionsRequest, _ ...grpc.CallOption) (*sessionv1.ListSessionsResponse, error) {
	return s.List(c, r)
}
func (s *SelfSessionService) RevokeMySession(c context.Context, r *sessionv1.RevokeMySessionRequest, _ ...grpc.CallOption) (*sessionv1.RevokeSessionResponse, error) {
	return s.Revoke(c, r)
}
func (s *SelfSessionService) RevokeAllMySessions(c context.Context, r *emptypb.Empty, _ ...grpc.CallOption) (*sessionv1.RevokeAllMySessionsResponse, error) {
	return s.All(c, r)
}
