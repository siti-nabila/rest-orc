package usecase

import (
	"context"

	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/roles/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

type List struct{ gateway domain.ListGateway }
type Get struct{ gateway domain.GetGateway }

func NewList(g domain.ListGateway) (*List, error) {
	if g == nil {
		return nil, dictionary.ErrRolesUseCaseGatewayRequired
	}
	return &List{gateway: g}, nil
}
func NewGet(g domain.GetGateway) (*Get, error) {
	if g == nil {
		return nil, dictionary.ErrRolesUseCaseGatewayRequired
	}
	return &Get{gateway: g}, nil
}
func authorize(ctx context.Context, a domain.Actor) error {
	if ctx == nil || a.UserID == 0 {
		return common.ErrUnauthorized
	}
	if a.MustChangePassword || !a.Permissions[domain.ReadPermission] {
		return common.ErrForbidden
	}
	return nil
}
func (uc *List) Execute(ctx context.Context, a domain.Actor, input domain.ListInput) (domain.Page[domain.Role], error) {
	if err := authorize(ctx, a); err != nil {
		return domain.Page[domain.Role]{}, err
	}
	q, err := domain.ParseListQuery(input)
	if err != nil {
		return domain.Page[domain.Role]{}, err
	}
	result, err := uc.gateway.List(ctx, q)
	if err != nil {
		return domain.Page[domain.Role]{}, err
	}
	items := make([]domain.Role, len(result.Items))
	for i, r := range result.Items {
		items[i] = r.Clone()
	}
	result.Items = items
	return result, nil
}
func (uc *Get) Execute(ctx context.Context, a domain.Actor, target string) (domain.Detail, error) {
	if err := authorize(ctx, a); err != nil {
		return domain.Detail{}, err
	}
	id, err := domain.ParseRoleID(target)
	if err != nil {
		return domain.Detail{}, err
	}
	result, err := uc.gateway.Get(ctx, id)
	if err != nil {
		return domain.Detail{}, err
	}
	if result.Role.ID != id {
		return domain.Detail{}, common.ErrInternalServerError
	}
	result.Role = result.Role.Clone()
	permissions := make([]domain.Permission, len(result.Permissions))
	copy(permissions, result.Permissions)
	result.Permissions = permissions
	return result, nil
}
