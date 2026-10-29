package usecase

import (
	"context"

	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

type ChangePassword struct {
	gateway domain.ChangePasswordGateway
}

func NewChangePassword(gateway domain.ChangePasswordGateway) (*ChangePassword, error) {
	if gateway == nil {
		return nil, dictionary.ErrAuthUseCaseGatewayRequired
	}
	return &ChangePassword{gateway: gateway}, nil
}

func (usecase *ChangePassword) Execute(
	ctx context.Context,
	userID uint64,
	request domain.ChangePasswordRequest,
) (domain.ChangePasswordResponse, error) {
	if ctx == nil {
		return domain.ChangePasswordResponse{}, dictionary.ErrAuthUseCaseContextRequired
	}
	if userID == 0 {
		return domain.ChangePasswordResponse{}, common.ErrUnauthorized
	}
	if err := request.Validate(); err != nil {
		return domain.ChangePasswordResponse{}, err
	}
	return usecase.gateway.ChangePassword(ctx, request)
}
