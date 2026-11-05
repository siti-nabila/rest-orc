package usecase

import (
	"context"

	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

type (
	Login struct {
		gateway domain.LoginGateway
	}
)

func NewLogin(gateway domain.LoginGateway) (*Login, error) {
	if gateway == nil {
		return nil, dictionary.ErrAuthUseCaseGatewayRequired
	}
	return &Login{gateway: gateway}, nil
}

func (uc *Login) Execute(ctx context.Context, request domain.AuthRequest) (domain.SessionResult, error) {
	if ctx == nil {
		return domain.SessionResult{}, dictionary.ErrAuthUseCaseContextRequired
	}
	if err := request.ValidateLogin(); err != nil {
		return domain.SessionResult{}, err
	}
	return uc.gateway.Login(ctx, request)
}
