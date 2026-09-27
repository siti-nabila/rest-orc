package mocks

import (
	"context"

	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
)

type AuthUseCase struct {
	ExecuteFunc func(
		context.Context,
		domain.AuthRequest,
	) (domain.AuthResponse, error)
	Calls int
}

func (usecase *AuthUseCase) Execute(
	ctx context.Context,
	request domain.AuthRequest,
) (domain.AuthResponse, error) {
	usecase.Calls++
	return usecase.ExecuteFunc(ctx, request)
}
