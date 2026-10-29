package authservice

import (
	"context"

	authv1 "github.com/siti-nabila/api-contracts/pb/auth/v1"
	sessionv1 "github.com/siti-nabila/api-contracts/pb/session/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/rest-orc/internal/client/grpcclient"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"google.golang.org/grpc"
)

type SessionService interface {
	RefreshSession(context.Context, *authv1.RefreshSessionRequest, ...grpc.CallOption) (*userv1.UserTokenResponse, error)
	Logout(context.Context, *authv1.LogoutRequest, ...grpc.CallOption) (*authv1.LogoutResponse, error)
}

func NewWithSessionService(transport *grpcclient.Client, service Service, profile ProfileService, password PasswordService, session SessionService) (*Client, error) {
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
	if session == nil {
		return nil, dictionary.ErrSessionDependencyRequired
	}
	return NewWithSelfSessionService(transport, service, profile, password, session, sessionv1.NewSessionServiceClient(transport.Connection()))
}

func (c *Client) RefreshSession(ctx context.Context, request *authv1.RefreshSessionRequest) (*userv1.UserTokenResponse, error) {
	if ctx == nil {
		return nil, dictionary.ErrGRPCAuthRequestContextRequired
	}
	if request == nil {
		return nil, dictionary.ErrSessionRequestRequired
	}
	child, cancel := c.transport.RequestContext(ctx)
	defer cancel()
	result, err := c.sessionService.RefreshSession(child, request)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, dictionary.ErrSessionResultInvalid
	}
	return result, nil
}

func (c *Client) Logout(ctx context.Context, request *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	if ctx == nil {
		return nil, dictionary.ErrGRPCAuthRequestContextRequired
	}
	if request == nil {
		return nil, dictionary.ErrSessionRequestRequired
	}
	child, cancel := c.transport.RequestContext(ctx)
	defer cancel()
	result, err := c.sessionService.Logout(child, request)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, dictionary.ErrSessionResultInvalid
	}
	return result, nil
}
