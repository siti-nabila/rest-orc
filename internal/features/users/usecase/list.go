package usecase

import (
	"context"

	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/pkg/pagination"
)

type List struct {
	gateway domain.Gateway
}

func NewList(gateway domain.Gateway) (*List, error) {
	if gateway == nil {
		return nil, dictionary.ErrUsersUseCaseGatewayRequired
	}
	return &List{gateway: gateway}, nil
}

func (usecase *List) Execute(
	ctx context.Context,
	query domain.ListQuery,
) (pagination.Page[domain.ListItem], error) {
	if ctx == nil {
		return pagination.Page[domain.ListItem]{},
			dictionary.ErrUsersUseCaseContextRequired
	}
	return usecase.gateway.List(ctx, query)
}
