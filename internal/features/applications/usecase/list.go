package usecase

import (
	"context"

	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/applications/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

type ApplicationCatalogList struct{ gateway domain.ApplicationListGateway }

func NewList(g domain.ApplicationListGateway) (*ApplicationCatalogList, error) {
	if g == nil {
		return nil, dictionary.ErrApplicationsUseCaseGatewayRequired
	}
	return &ApplicationCatalogList{gateway: g}, nil
}
func (u *ApplicationCatalogList) Execute(ctx context.Context, a domain.ApplicationActor, input domain.ApplicationListInput) (domain.ApplicationPage[domain.Application], error) {
	if ctx == nil || a.UserID == 0 {
		return domain.ApplicationPage[domain.Application]{}, common.ErrUnauthorized
	}
	if a.MustChangePassword || !a.Permissions[domain.ReadPermission] {
		return domain.ApplicationPage[domain.Application]{}, common.ErrForbidden
	}
	q, err := domain.ParseApplicationListQuery(input)
	if err != nil {
		return domain.ApplicationPage[domain.Application]{}, err
	}
	out, err := u.gateway.ListApplications(ctx, q)
	if err != nil {
		return domain.ApplicationPage[domain.Application]{}, err
	}
	items := make([]domain.Application, len(out.Items))
	copy(items, out.Items)
	out.Items = items
	return out, nil
}
