package mocks

import (
	"context"

	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/pkg/pagination"
)

type Gateway struct {
	ListFunc func(context.Context, domain.ListQuery) (pagination.Page[domain.ListItem], error)
	Calls    int
}

func (gateway *Gateway) List(
	ctx context.Context,
	query domain.ListQuery,
) (pagination.Page[domain.ListItem], error) {
	gateway.Calls++
	return gateway.ListFunc(ctx, query)
}
