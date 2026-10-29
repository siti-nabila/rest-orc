package usecase

import (
	"context"

	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

type Refresh struct{ gateway domain.RefreshGateway }

func NewRefresh(gateway domain.RefreshGateway) (*Refresh, error) {
	if gateway == nil {
		return nil, dictionary.ErrAuthUseCaseGatewayRequired
	}
	return &Refresh{gateway: gateway}, nil
}

func (uc *Refresh) Execute(ctx context.Context, refresh string) (domain.SessionResult, error) {
	if ctx == nil {
		return domain.SessionResult{}, dictionary.ErrAuthUseCaseContextRequired
	}
	if refresh == "" {
		return domain.SessionResult{}, dictionary.ErrSessionRequestRequired
	}
	return uc.gateway.Refresh(ctx, refresh)
}
