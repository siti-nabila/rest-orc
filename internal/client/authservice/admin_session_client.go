package authservice

import (
	"context"

	sessionv1 "github.com/siti-nabila/api-contracts/pb/session/v1"
	"github.com/siti-nabila/rest-orc/internal/client/grpcclient"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"google.golang.org/grpc"
)

// AdminSessionService owns user-target session RPCs; SelfSessionService owns
// the caller's sessions, and SessionService owns AuthService refresh/logout.
type AdminSessionService interface {
	ListUserSessions(context.Context, *sessionv1.ListUserSessionsRequest, ...grpc.CallOption) (*sessionv1.ListSessionsResponse, error)
	RevokeUserSession(context.Context, *sessionv1.RevokeUserSessionRequest, ...grpc.CallOption) (*sessionv1.RevokeSessionResponse, error)
	RevokeAllUserSessions(context.Context, *sessionv1.RevokeAllUserSessionsRequest, ...grpc.CallOption) (*sessionv1.RevokeAllUserSessionsResponse, error)
}

func NewWithAdminSessionService(transport *grpcclient.Client, service Service, profile ProfileService, password PasswordService, session SessionService, self SelfSessionService, admin AdminSessionService) (*Client, error) {
	client, err := NewWithSelfSessionService(transport, service, profile, password, session, self)
	if err != nil {
		return nil, err
	}
	if admin == nil {
		return nil, dictionary.ErrSessionDependencyRequired
	}
	client.adminSessionService = admin
	return client, nil
}

func (c *Client) ListUserSessions(ctx context.Context, request *sessionv1.ListUserSessionsRequest) (*sessionv1.ListSessionsResponse, error) {
	if ctx == nil {
		return nil, dictionary.ErrGRPCAuthRequestContextRequired
	}
	if request == nil {
		return nil, dictionary.ErrSessionRequestRequired
	}
	child, cancel := c.transport.RequestContext(ctx)
	defer cancel()
	result, err := c.adminSessionService.ListUserSessions(child, request)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, dictionary.ErrSessionResultInvalid
	}
	return result, nil
}

func (c *Client) RevokeUserSession(ctx context.Context, request *sessionv1.RevokeUserSessionRequest) (*sessionv1.RevokeSessionResponse, error) {
	if ctx == nil {
		return nil, dictionary.ErrGRPCAuthRequestContextRequired
	}
	if request == nil {
		return nil, dictionary.ErrSessionRequestRequired
	}
	child, cancel := c.transport.RequestContext(ctx)
	defer cancel()
	result, err := c.adminSessionService.RevokeUserSession(child, request)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, dictionary.ErrSessionResultInvalid
	}
	return result, nil
}

func (c *Client) RevokeAllUserSessions(ctx context.Context, request *sessionv1.RevokeAllUserSessionsRequest) (*sessionv1.RevokeAllUserSessionsResponse, error) {
	if ctx == nil {
		return nil, dictionary.ErrGRPCAuthRequestContextRequired
	}
	if request == nil {
		return nil, dictionary.ErrSessionRequestRequired
	}
	child, cancel := c.transport.RequestContext(ctx)
	defer cancel()
	result, err := c.adminSessionService.RevokeAllUserSessions(child, request)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, dictionary.ErrSessionResultInvalid
	}
	return result, nil
}
