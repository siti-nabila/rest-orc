package mocks

import (
	"context"

	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
)

type RefreshUseCase struct {
	ExecuteFunc func(context.Context, string) (domain.SessionResult, error)
}

func (uc *RefreshUseCase) Execute(ctx context.Context, r string) (domain.SessionResult, error) {
	return uc.ExecuteFunc(ctx, r)
}

type LogoutUseCase struct {
	ExecuteFunc func(context.Context, domain.LogoutCredential) (domain.LogoutResponse, error)
}

func (uc *LogoutUseCase) Execute(ctx context.Context, r domain.LogoutCredential) (domain.LogoutResponse, error) {
	return uc.ExecuteFunc(ctx, r)
}
