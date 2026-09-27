package usecase

import (
	"context"

	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/pkg/pagination"
)

type List struct {
	repository domain.Repository
}

func NewList(repository domain.Repository) (*List, error) {
	if repository == nil {
		return nil, dictionary.ErrUsersUseCaseRepositoryRequired
	}
	return &List{repository: repository}, nil
}

func (usecase *List) Execute(
	ctx context.Context,
	query domain.ListQuery,
) (pagination.Page[domain.ListItem], error) {
	if ctx == nil {
		return pagination.Page[domain.ListItem]{},
			dictionary.ErrUsersUseCaseContextRequired
	}
	return usecase.repository.List(ctx, query)
}
