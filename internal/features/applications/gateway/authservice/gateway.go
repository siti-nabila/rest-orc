package authservice

import (
	"context"
	"slices"

	applicationv1 "github.com/siti-nabila/api-contracts/pb/application/v1"
	paginatorv1 "github.com/siti-nabila/api-contracts/pb/paginator/v1"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/applications/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"google.golang.org/grpc/metadata"
)

type ApplicationCatalogClient interface {
	ListApplications(context.Context, *applicationv1.ListApplicationsRequest) (*applicationv1.ListApplicationsResponse, error)
}
type ApplicationCatalogGateway struct{ client ApplicationCatalogClient }

func New(c ApplicationCatalogClient) (*ApplicationCatalogGateway, error) {
	if c == nil {
		return nil, dictionary.ErrApplicationsGatewayClientRequired
	}
	return &ApplicationCatalogGateway{client: c}, nil
}
func (g *ApplicationCatalogGateway) ListApplications(ctx context.Context, q domain.ApplicationListQuery) (domain.ApplicationPage[domain.Application], error) {
	if ctx == nil {
		return domain.ApplicationPage[domain.Application]{}, dictionary.ErrApplicationsGatewayContextRequired
	}
	if err := q.Validate(); err != nil {
		return domain.ApplicationPage[domain.Application]{}, err
	}
	page := &paginatorv1.PageQuery{Page: q.Page, Limit: q.Limit, LastId: q.LastID, Sort: []*paginatorv1.Sort{{Field: "id", Desc: true}}}
	if q.Keyword != "" {
		page.Search = &paginatorv1.Search{Fields: []string{"keyword"}, Keyword: q.Keyword, Mode: paginatorv1.SearchMode_SEARCH_MODE_FULL_TEXT_TRIGRAM}
	}
	request := &applicationv1.ListApplicationsRequest{Query: page, Filter: &applicationv1.ApplicationFilter{Statuses: slices.Clone(q.Statuses)}}
	md, _ := metadata.FromOutgoingContext(ctx)
	md = md.Copy()
	md.Delete("version")
	md.Delete("idempotency-key")
	r, err := g.client.ListApplications(metadata.NewOutgoingContext(ctx, md), request)
	if err != nil {
		return domain.ApplicationPage[domain.Application]{}, err
	}
	if r == nil || r.Meta == nil {
		return domain.ApplicationPage[domain.Application]{}, common.ErrInternalServerError
	}
	m := r.Meta
	if m.Page < 1 || m.Limit < 1 || m.Limit > 100 || m.Total < 0 || m.TotalPages < 0 {
		return domain.ApplicationPage[domain.Application]{}, common.ErrInternalServerError
	}
	items := make([]domain.Application, len(r.Items))
	for i, item := range r.Items {
		if item == nil || item.Id == 0 || item.Id > 2147483647 || (item.Status != "ACTIVE" && item.Status != "DISABLED") {
			return domain.ApplicationPage[domain.Application]{}, common.ErrInternalServerError
		}
		items[i] = domain.Application{ID: item.Id, Code: item.Code, Name: item.Name, Status: item.Status, IsSystem: item.IsSystem}
	}
	return domain.ApplicationPage[domain.Application]{
		Items: items, Total: m.Total, Page: m.Page, Limit: m.Limit, TotalPages: m.TotalPages,
		HasNext: m.HasNext, HasPrev: m.HasPrev, NextCursor: r.NextCursor,
	}, nil
}
