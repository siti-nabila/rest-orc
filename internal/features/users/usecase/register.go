package usecase

import (
	"context"

	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

type (
	Register struct {
		repository domain.Repository
	}
)

func NewRegister(repo domain.Repository) (*Register, error) {
	if repo == nil {
		return nil, dictionary.ErrUsersUseCaseRepositoryRequired
	}
	return &Register{repository: repo}, nil
}

func (uc *Register) Execute(ctx context.Context, request domain.AuthRequest) (domain.AuthResponse, error) {
	if ctx == nil {
		return domain.AuthResponse{}, dictionary.ErrUsersUseCaseContextRequired
	}
	return uc.repository.Register(ctx, request)
}
