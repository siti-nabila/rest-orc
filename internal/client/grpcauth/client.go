package grpcauth

import (
	"context"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/rest-orc/internal/client/grpcclient"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Service interface {
	Me(
		ctx context.Context,
		request *emptypb.Empty,
		options ...grpc.CallOption,
	) (*userv1.UserData, error)
	ListUsers(
		ctx context.Context,
		request *userv1.ListUsersRequest,
		options ...grpc.CallOption,
	) (*userv1.ListUsersResponse, error)

	Register(ctx context.Context, request *userv1.AuthRequest, opts ...grpc.CallOption) (*userv1.UserTokenResponse, error)
	Login(ctx context.Context, request *userv1.AuthRequest, opts ...grpc.CallOption) (*userv1.UserTokenResponse, error)
}

type Client struct {
	transport *grpcclient.Client
	service   Service
}

func New(transport *grpcclient.Client) (*Client, error) {
	if transport == nil {
		return nil, dictionary.ErrGRPCAuthTransportRequired
	}

	return NewWithService(
		transport,
		userv1.NewUserServiceClient(transport.Connection()),
	)
}

func NewWithService(
	transport *grpcclient.Client,
	service Service,
) (*Client, error) {
	if transport == nil {
		return nil, dictionary.ErrGRPCAuthTransportRequired
	}
	if service == nil {
		return nil, dictionary.ErrGRPCAuthServiceRequired
	}

	return &Client{
		transport: transport,
		service:   service,
	}, nil
}

func (client *Client) Me(ctx context.Context) (*userv1.UserData, error) {
	if ctx == nil {
		return nil, dictionary.ErrGRPCAuthRequestContextRequired
	}

	requestContext, cancel := client.transport.RequestContext(ctx)
	defer cancel()

	response, err := client.service.Me(requestContext, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, dictionary.ErrGRPCAuthMeResponseRequired
	}
	return response, nil
}

func (client *Client) ListUsers(
	ctx context.Context,
	request *userv1.ListUsersRequest,
) (*userv1.ListUsersResponse, error) {
	if ctx == nil {
		return nil, dictionary.ErrGRPCAuthRequestContextRequired
	}
	if request == nil {
		return nil, dictionary.ErrGRPCAuthListUsersRequestRequired
	}

	requestContext, cancel := client.transport.RequestContext(ctx)
	defer cancel()

	response, err := client.service.ListUsers(requestContext, request)
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, dictionary.ErrGRPCAuthListUsersResponseRequired
	}
	return response, nil
}

func (c *Client) Register(ctx context.Context, request *userv1.AuthRequest) (*userv1.UserTokenResponse, error) {
	if ctx == nil {
		return nil, dictionary.ErrGRPCAuthRequestContextRequired
	}
	if request == nil {
		return nil, dictionary.ErrGRPCAuthRegisterRequestRequired
	}

	requestContext, cancel := c.transport.RequestContext(ctx)
	defer cancel()

	response, err := c.service.Register(requestContext, request)
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, dictionary.ErrGRPCAuthRegisterResponseRequired
	}
	return response, nil
}

func (c *Client) Login(ctx context.Context, request *userv1.AuthRequest) (*userv1.UserTokenResponse, error) {
	if ctx == nil {
		return nil, dictionary.ErrGRPCAuthRequestContextRequired
	}
	if request == nil {
		return nil, dictionary.ErrGRPCAuthLoginRequestRequired
	}

	requestContext, cancel := c.transport.RequestContext(ctx)
	defer cancel()

	response, err := c.service.Login(requestContext, request)
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, dictionary.ErrGRPCAuthLoginResponseRequired
	}
	return response, nil
}
