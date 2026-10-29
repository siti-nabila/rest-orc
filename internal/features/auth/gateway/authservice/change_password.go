package authservice

import (
	"context"

	authv1 "github.com/siti-nabila/api-contracts/pb/auth/v1"
	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

type PasswordClient interface {
	ChangePassword(context.Context, *authv1.ChangePasswordRequest) (*authv1.ChangePasswordResponse, error)
}

type PasswordGateway struct {
	client PasswordClient
}

func NewPassword(client PasswordClient) (*PasswordGateway, error) {
	if client == nil {
		return nil, dictionary.ErrAuthGatewayClientRequired
	}
	return &PasswordGateway{client: client}, nil
}

func (gateway *PasswordGateway) ChangePassword(
	ctx context.Context,
	request domain.ChangePasswordRequest,
) (domain.ChangePasswordResponse, error) {
	if ctx == nil {
		return domain.ChangePasswordResponse{}, dictionary.ErrAuthGatewayContextRequired
	}
	result, err := gateway.client.ChangePassword(ctx, &authv1.ChangePasswordRequest{
		CurrentPassword: request.CurrentPassword,
		NewPassword:     request.NewPassword,
	})
	if err != nil {
		return domain.ChangePasswordResponse{}, err
	}
	if result == nil {
		return domain.ChangePasswordResponse{}, dictionary.ErrGRPCChangePasswordResponseRequired
	}
	return domain.ChangePasswordResponse{
		Changed:       result.Changed,
		LoginRequired: result.LoginRequired,
	}, nil
}
