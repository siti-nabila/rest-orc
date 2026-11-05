package usecase

import (
	"context"

	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

type (
	Register struct {
		gateway domain.RegisterGateway
	}
)

func NewRegister(gateway domain.RegisterGateway) (*Register, error) {
	if gateway == nil {
		return nil, dictionary.ErrAuthUseCaseGatewayRequired
	}
	return &Register{gateway: gateway}, nil
}

func (uc *Register) Execute(ctx context.Context, request domain.AuthRequest) (domain.RegisterResponse, error) {
	if ctx == nil {
		return domain.RegisterResponse{}, dictionary.ErrAuthUseCaseContextRequired
	}
	if err := request.ValidateRegister(); err != nil {
		return domain.RegisterResponse{}, err
	}
	return uc.gateway.Register(ctx, request)
}
