package usecase

import (
	"context"

	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

type (
	Login struct {
		repository domain.Repository
	}
)

func NewLogin(repo domain.Repository) (*Login, error) {
	if repo == nil {
		return nil, dictionary.ErrUsersUseCaseRepositoryRequired
	}
	return &Login{repository: repo}, nil
}

func (uc *Login) Execute(ctx context.Context, request domain.AuthRequest) (domain.AuthResponse, error) {
	if ctx == nil {
		return domain.AuthResponse{}, dictionary.ErrUsersUseCaseContextRequired
	}
	return uc.repository.Login(ctx, request)
}
