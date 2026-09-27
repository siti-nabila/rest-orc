package mocks

import (
	"context"

	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/pkg/pagination"
)

type Repository struct {
	ListFunc func(
		context.Context,
		domain.ListQuery,
	) (pagination.Page[domain.ListItem], error)
	RegisterFunc func(
		context.Context,
		domain.AuthRequest,
	) (domain.AuthResponse, error)
	LoginFunc func(
		context.Context,
		domain.AuthRequest,
	) (domain.AuthResponse, error)
	Calls         int
	RegisterCalls int
	LoginCalls    int
}

func (repository *Repository) Register(
	ctx context.Context,
	request domain.AuthRequest,
) (domain.AuthResponse, error) {
	repository.RegisterCalls++
	if repository.RegisterFunc == nil {
		panic("unexpected Register call")
	}
	return repository.RegisterFunc(ctx, request)
}

func (repository *Repository) Login(
	ctx context.Context,
	request domain.AuthRequest,
) (domain.AuthResponse, error) {
	repository.LoginCalls++
	if repository.LoginFunc == nil {
		panic("unexpected Login call")
	}
	return repository.LoginFunc(ctx, request)
}

func (repository *Repository) List(
	ctx context.Context,
	query domain.ListQuery,
) (pagination.Page[domain.ListItem], error) {
	repository.Calls++
	return repository.ListFunc(ctx, query)
}
