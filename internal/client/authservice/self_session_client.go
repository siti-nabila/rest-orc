package authservice

import (
	"context"

	applicationv1 "github.com/siti-nabila/api-contracts/pb/application/v1"
	rolev1 "github.com/siti-nabila/api-contracts/pb/role/v1"
	sessionv1 "github.com/siti-nabila/api-contracts/pb/session/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
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
	return &Client{
		transport: transport, service: service, profileService: profile,
		passwordService: password, sessionService: session, selfSessionService: self,
		adminSessionService:          sessionv1.NewSessionServiceClient(transport.Connection()),
		userDetailService:            userv1.NewUserServiceClient(transport.Connection()),
		userStatusService:            userv1.NewUserServiceClient(transport.Connection()),
		userProfileService:           userv1.NewUserServiceClient(transport.Connection()),
		userCreationService:          userv1.NewUserServiceClient(transport.Connection()),
		roleCatalogService:           rolev1.NewRoleServiceClient(transport.Connection()),
		userRoleManagementService:    userv1.NewUserServiceClient(transport.Connection()),
		applicationCatalogService:    applicationv1.NewApplicationServiceClient(transport.Connection()),
		userApplicationAccessService: userv1.NewUserServiceClient(transport.Connection()),
		applicationAccessService:     applicationv1.NewApplicationServiceClient(transport.Connection()),
	}, nil
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
