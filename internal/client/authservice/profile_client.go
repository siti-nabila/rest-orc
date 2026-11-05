package authservice

import (
	"context"

	authv1 "github.com/siti-nabila/api-contracts/pb/auth/v1"
	profilev1 "github.com/siti-nabila/api-contracts/pb/profile/v1"
	"github.com/siti-nabila/rest-orc/internal/client/grpcclient"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"google.golang.org/grpc"
)

type ProfileService interface {
	UpdateProfile(context.Context, *profilev1.ProfileRequest, ...grpc.CallOption) (*profilev1.ProfileResponse, error)
}

func NewWithServices(transport *grpcclient.Client, service Service, profileService ProfileService) (*Client, error) {
	if transport == nil {
		return nil, dictionary.ErrGRPCAuthTransportRequired
	}
	if service == nil {
		return nil, dictionary.ErrGRPCAuthServiceRequired
	}
	if profileService == nil {
		return nil, dictionary.ErrGRPCProfileServiceRequired
	}
	return NewWithPasswordService(
		transport,
		service,
		profileService,
		authv1.NewAuthServiceClient(transport.Connection()),
	)
}

func (client *Client) UpdateProfile(ctx context.Context, request *profilev1.ProfileRequest) (*profilev1.ProfileResponse, error) {
	if ctx == nil {
		return nil, dictionary.ErrGRPCAuthRequestContextRequired
	}
	if request == nil {
		return nil, dictionary.ErrGRPCProfileRequestRequired
	}
	requestContext, cancel := client.transport.RequestContext(ctx)
	defer cancel()
	result, err := client.profileService.UpdateProfile(requestContext, request)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, dictionary.ErrGRPCProfileResponseRequired
	}
	return result, nil
}
