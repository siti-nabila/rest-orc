package authservice

import (
	"context"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

type Client interface {
	Register(ctx context.Context, request *userv1.AuthRequest) (*userv1.UserTokenResponse, error)
	Login(ctx context.Context, request *userv1.AuthRequest) (*userv1.UserTokenResponse, error)
}

type Gateway struct{ client Client }

func New(client Client) (*Gateway, error) {
	if client == nil {
		return nil, dictionary.ErrAuthGatewayClientRequired
	}
	return &Gateway{client: client}, nil
}

func (gateway *Gateway) Register(ctx context.Context, request domain.AuthRequest) (domain.RegisterResponse, error) {
	if ctx == nil {
		return domain.RegisterResponse{}, dictionary.ErrAuthGatewayContextRequired
	}

	protobufRequest := &userv1.AuthRequest{
		Email:    request.Email,
		Password: request.Password,
	}
	response, err := gateway.client.Register(credentialContext(ctx, ""), protobufRequest)
	if err != nil {
		return domain.RegisterResponse{}, err
	}
	if response == nil {
		return domain.RegisterResponse{}, dictionary.ErrGRPCAuthRegisterResponseRequired
	}
	if response.RegistrationId == 0 || response.RegistrationStatus != "PENDING" || response.Token != "" || response.RefreshToken != "" || response.SessionId != "" || response.AccessExpiresAt != nil || response.SessionExpiresAt != nil || response.MustChangePassword {
		return domain.RegisterResponse{}, dictionary.ErrSessionResultInvalid
	}
	return domain.RegisterResponse{RegistrationID: response.RegistrationId, RegistrationStatus: response.RegistrationStatus}, nil
}

func (gateway *Gateway) Login(ctx context.Context, request domain.AuthRequest) (domain.SessionResult, error) {
	if ctx == nil {
		return domain.SessionResult{}, dictionary.ErrAuthGatewayContextRequired
	}
	protobufRequest := &userv1.AuthRequest{
		Email:    request.Email,
		Password: request.Password,
	}
	response, err := gateway.client.Login(credentialContext(ctx, ""), protobufRequest)
	if err != nil {
		return domain.SessionResult{}, err
	}
	if response == nil {
		return domain.SessionResult{}, dictionary.ErrGRPCAuthLoginResponseRequired
	}
	return sessionResult(response)
}
