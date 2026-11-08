package mocks

import (
	"context"
	sessionv1 "github.com/siti-nabila/api-contracts/pb/session/v1"
	"github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"google.golang.org/protobuf/types/known/emptypb"
)

type SelfSessionServer struct {
	sessionv1.UnimplementedSessionServiceServer
}

func (*SelfSessionServer) ListMySessions(context.Context, *sessionv1.ListMySessionsRequest) (*sessionv1.ListSessionsResponse, error) {
	return fixtures.MySessionsResponse(17), nil
}

func (*SelfSessionServer) RevokeMySession(context.Context, *sessionv1.RevokeMySessionRequest) (*sessionv1.RevokeSessionResponse, error) {
	return &sessionv1.RevokeSessionResponse{}, nil
}

func (*SelfSessionServer) RevokeAllMySessions(context.Context, *emptypb.Empty) (*sessionv1.RevokeAllMySessionsResponse, error) {
	return &sessionv1.RevokeAllMySessionsResponse{Changed: true, LoginRequired: true}, nil
}

func (*SelfSessionServer) ListUserSessions(_ context.Context, r *sessionv1.ListUserSessionsRequest) (*sessionv1.ListSessionsResponse, error) {
	return fixtures.MySessionsResponse(r.UserId), nil
}

func (*SelfSessionServer) RevokeUserSession(context.Context, *sessionv1.RevokeUserSessionRequest) (*sessionv1.RevokeSessionResponse, error) {
	return &sessionv1.RevokeSessionResponse{}, nil
}

func (*SelfSessionServer) RevokeAllUserSessions(context.Context, *sessionv1.RevokeAllUserSessionsRequest) (*sessionv1.RevokeAllUserSessionsResponse, error) {
	return &sessionv1.RevokeAllUserSessionsResponse{Version: 9007199254740993}, nil
}
