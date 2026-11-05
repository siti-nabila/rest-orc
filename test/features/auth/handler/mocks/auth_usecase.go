package mocks

import (
	"context"

	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
)

type AuthUseCase struct {
	ExecuteFunc func(
		context.Context,
		domain.AuthRequest,
	) (domain.RegisterResponse, error)
	Calls int
}

func (usecase *AuthUseCase) Execute(
	ctx context.Context,
	request domain.AuthRequest,
) (domain.RegisterResponse, error) {
	usecase.Calls++
	return usecase.ExecuteFunc(ctx, request)
}

type LoginUseCase struct {
	ExecuteFunc func(
		context.Context,
		domain.AuthRequest,
	) (domain.SessionResult, error)
	Calls int
}

func (usecase *LoginUseCase) Execute(
	ctx context.Context,
	request domain.AuthRequest,
) (domain.SessionResult, error) {
	usecase.Calls++
	return usecase.ExecuteFunc(ctx, request)
}
