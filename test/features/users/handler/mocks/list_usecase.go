package mocks

import (
	"context"

	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/pkg/pagination"
)

type ListUseCase struct {
	ExecuteFunc func(
		context.Context,
		domain.ListQuery,
	) (pagination.Page[domain.ListItem], error)
	Calls int
}

func (usecase *ListUseCase) Execute(
	ctx context.Context,
	query domain.ListQuery,
) (pagination.Page[domain.ListItem], error) {
	usecase.Calls++
	return usecase.ExecuteFunc(ctx, query)
}
