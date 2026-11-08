package authservice

import (
	"context"

	paginatorv1 "github.com/siti-nabila/api-contracts/pb/paginator/v1"
	rolev1 "github.com/siti-nabila/api-contracts/pb/role/v1"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/roles/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"google.golang.org/grpc/metadata"
)

type RoleCatalogClient interface {
	ListRoles(context.Context, *rolev1.ListRolesRequest) (*rolev1.ListRolesResponse, error)
	GetRole(context.Context, *rolev1.GetRoleRequest) (*rolev1.GetRoleResponse, error)
}
type RoleCatalogGateway struct{ client RoleCatalogClient }

func New(c RoleCatalogClient) (*RoleCatalogGateway, error) {
	if c == nil {
		return nil, dictionary.ErrRolesGatewayClientRequired
	}
	return &RoleCatalogGateway{client: c}, nil
}
func (g *RoleCatalogGateway) List(ctx context.Context, q domain.ListQuery) (domain.Page[domain.Role], error) {
	if ctx == nil {
		return domain.Page[domain.Role]{}, dictionary.ErrRolesGatewayContextRequired
	}
	if err := q.Validate(); err != nil {
		return domain.Page[domain.Role]{}, err
	}
	page := &paginatorv1.PageQuery{Page: q.Page, Limit: q.Limit, LastId: q.LastID, Sort: []*paginatorv1.Sort{{Field: "created_at", Desc: true}, {Field: "updated_at", Desc: true}, {Field: "id", Desc: true}}}
	if q.Keyword != "" {
		page.Search = &paginatorv1.Search{Fields: []string{"keyword"}, Keyword: q.Keyword, Mode: paginatorv1.SearchMode_SEARCH_MODE_FULL_TEXT_TRIGRAM}
	}
	request := &rolev1.ListRolesRequest{Query: page, Filter: &rolev1.RoleFilter{ApplicationCode: q.ApplicationCode, Statuses: append([]string(nil), q.Statuses...)}}
	response, err := g.client.ListRoles(readContext(ctx), request)
	if err != nil {
		return domain.Page[domain.Role]{}, err
	}
	if response == nil || response.Meta == nil {
		return domain.Page[domain.Role]{}, common.ErrInternalServerError
	}
	m := response.Meta
	if m.Page < 1 || m.Limit < 1 || m.Limit > 100 || m.Total < 0 || m.TotalPages < 0 {
		return domain.Page[domain.Role]{}, common.ErrInternalServerError
	}
	items := make([]domain.Role, len(response.Items))
	for i, r := range response.Items {
		item, err := safeRole(r)
		if err != nil {
			return domain.Page[domain.Role]{}, err
		}
		items[i] = item
	}
	return domain.Page[domain.Role]{Items: items, Total: m.Total, Page: m.Page, Limit: m.Limit, TotalPages: m.TotalPages, HasNext: m.HasNext, HasPrev: m.HasPrev, NextCursor: response.NextCursor}, nil
}
func (g *RoleCatalogGateway) Get(ctx context.Context, id uint64) (domain.Detail, error) {
	if ctx == nil {
		return domain.Detail{}, dictionary.ErrRolesGatewayContextRequired
	}
	if !domain.ValidateID(id) {
		return domain.Detail{}, common.ErrBadRequest
	}
	response, err := g.client.GetRole(readContext(ctx), &rolev1.GetRoleRequest{RoleId: id})
	if err != nil {
		return domain.Detail{}, err
	}
	if response == nil {
		return domain.Detail{}, common.ErrInternalServerError
	}
	role, err := safeRole(response.Role)
	if err != nil {
		return domain.Detail{}, err
	}
	if role.ID != id {
		return domain.Detail{}, common.ErrInternalServerError
	}
	permissions := make([]domain.Permission, len(response.Permissions))
	for i, p := range response.Permissions {
		if p == nil || !domain.ValidateID(p.Id) || !domain.ValidateID(p.ApplicationId) || p.ApplicationId != role.ApplicationID || p.ApplicationCode != role.ApplicationCode {
			return domain.Detail{}, common.ErrInternalServerError
		}
		permissions[i] = domain.Permission{ID: p.Id, ApplicationID: p.ApplicationId, ApplicationCode: p.ApplicationCode, Code: p.Code, Description: p.Description}
	}
	return domain.Detail{Role: role, Permissions: permissions}, nil
}

// Reads have no mutation precondition; inherited values cannot select technical state.
func readContext(ctx context.Context) context.Context {
	md, _ := metadata.FromOutgoingContext(ctx)
	md = md.Copy()
	md.Delete("version")
	md.Delete("idempotency-key")
	return metadata.NewOutgoingContext(ctx, md)
}
