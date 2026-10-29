package usecase

import (
	"context"

	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

type Logout struct{ gateway domain.LogoutGateway }

func NewLogout(gateway domain.LogoutGateway) (*Logout, error) {
	if gateway == nil {
		return nil, dictionary.ErrAuthUseCaseGatewayRequired
	}
	return &Logout{gateway: gateway}, nil
}

func (uc *Logout) Execute(ctx context.Context, credential domain.LogoutCredential) (domain.LogoutResponse, error) {
	if ctx == nil {
		return domain.LogoutResponse{}, dictionary.ErrAuthUseCaseContextRequired
	}
	if !credential.Valid() {
		return domain.LogoutResponse{}, dictionary.ErrSessionRequestRequired
	}
	return uc.gateway.Logout(ctx, credential)
}
