package authservice

import (
	"context"

	sessionv1 "github.com/siti-nabila/api-contracts/pb/session/v1"
	"github.com/siti-nabila/rest-orc/internal/client/grpcclient"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

// SelfSessionService is session.v1.SessionService, separate from AuthService lifecycle.
type SelfSessionService interface {
	ListMySessions(context.Context, *sessionv1.ListMySessionsRequest, ...grpc.CallOption) (*sessionv1.ListSessionsResponse, error)
	RevokeMySession(context.Context, *sessionv1.RevokeMySessionRequest, ...grpc.CallOption) (*sessionv1.RevokeSessionResponse, error)
	RevokeAllMySessions(context.Context, *emptypb.Empty, ...grpc.CallOption) (*sessionv1.RevokeAllMySessionsResponse, error)
}

func NewWithSelfSessionService(transport *grpcclient.Client, service Service, profile ProfileService, password PasswordService, session SessionService, self SelfSessionService) (*Client, error) {
	if transport == nil {
		return nil, dictionary.ErrGRPCAuthTransportRequired
	}
	if service == nil {
		return nil, dictionary.ErrGRPCAuthServiceRequired
	}
	if profile == nil {
		return nil, dictionary.ErrGRPCProfileServiceRequired
	}
	if password == nil {
		return nil, dictionary.ErrGRPCPasswordServiceRequired
	}
	if session == nil || self == nil {
		return nil, dictionary.ErrSessionDependencyRequired
	}
	return &Client{transport: transport, service: service, profileService: profile, passwordService: password, sessionService: session, selfSessionService: self}, nil
}

func (c *Client) ListMySessions(ctx context.Context, request *sessionv1.ListMySessionsRequest) (*sessionv1.ListSessionsResponse, error) {
	if ctx == nil {
		return nil, dictionary.ErrGRPCAuthRequestContextRequired
	}
	if request == nil {
		return nil, dictionary.ErrSessionRequestRequired
	}
	child, cancel := c.transport.RequestContext(ctx)
	defer cancel()
	result, err := c.selfSessionService.ListMySessions(child, request)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, dictionary.ErrSessionResultInvalid
	}
	return result, nil
}

func (c *Client) RevokeMySession(ctx context.Context, request *sessionv1.RevokeMySessionRequest) (*sessionv1.RevokeSessionResponse, error) {
	if ctx == nil {
		return nil, dictionary.ErrGRPCAuthRequestContextRequired
	}
	if request == nil {
		return nil, dictionary.ErrSessionRequestRequired
	}
	child, cancel := c.transport.RequestContext(ctx)
	defer cancel()
	result, err := c.selfSessionService.RevokeMySession(child, request)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, dictionary.ErrSessionResultInvalid
	}
	return result, nil
}

func (c *Client) RevokeAllMySessions(ctx context.Context) (*sessionv1.RevokeAllMySessionsResponse, error) {
	if ctx == nil {
		return nil, dictionary.ErrGRPCAuthRequestContextRequired
	}
	child, cancel := c.transport.RequestContext(ctx)
	defer cancel()
	result, err := c.selfSessionService.RevokeAllMySessions(child, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, dictionary.ErrSessionResultInvalid
	}
	return result, nil
}
