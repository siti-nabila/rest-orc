package authservice

import (
	"context"

	authv1 "github.com/siti-nabila/api-contracts/pb/auth/v1"
	"github.com/siti-nabila/rest-orc/internal/client/grpcclient"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"google.golang.org/grpc"
)

type PasswordService interface {
	ChangePassword(context.Context, *authv1.ChangePasswordRequest, ...grpc.CallOption) (*authv1.ChangePasswordResponse, error)
}

func NewWithPasswordService(
	transport *grpcclient.Client,
	service Service,
	profileService ProfileService,
	passwordService PasswordService,
) (*Client, error) {
	if transport == nil {
		return nil, dictionary.ErrGRPCAuthTransportRequired
	}
	if service == nil {
		return nil, dictionary.ErrGRPCAuthServiceRequired
	}
	if profileService == nil {
		return nil, dictionary.ErrGRPCProfileServiceRequired
	}
	if passwordService == nil {
		return nil, dictionary.ErrGRPCPasswordServiceRequired
	}
	return NewWithSessionService(transport, service, profileService, passwordService, authv1.NewAuthServiceClient(transport.Connection()))
}

func (client *Client) ChangePassword(
	ctx context.Context,
	request *authv1.ChangePasswordRequest,
) (*authv1.ChangePasswordResponse, error) {
	if ctx == nil {
		return nil, dictionary.ErrGRPCAuthRequestContextRequired
	}
	if request == nil {
		return nil, dictionary.ErrGRPCChangePasswordRequestRequired
	}
	requestContext, cancel := client.transport.RequestContext(ctx)
	defer cancel()

	result, err := client.passwordService.ChangePassword(requestContext, request)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, dictionary.ErrGRPCChangePasswordResponseRequired
	}
	return result, nil
}
