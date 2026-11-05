package mocks

import (
	"context"

	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
)

type PasswordUseCase struct {
	ExecuteFunc func(context.Context, uint64, domain.ChangePasswordRequest) (domain.ChangePasswordResponse, error)
	Calls       int
}

func (usecase *PasswordUseCase) Execute(ctx context.Context, userID uint64, request domain.ChangePasswordRequest) (domain.ChangePasswordResponse, error) {
	usecase.Calls++
	return usecase.ExecuteFunc(ctx, userID, request)
}
