package grpcauth

import (
	paginatorv1 "github.com/siti-nabila/api-contracts/pb/paginator/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/pkg/pagination"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type listRequest struct {
	query domain.ListQuery
}

func (request listRequest) protobuf() *userv1.ListUsersRequest {
	return &userv1.ListUsersRequest{
		Query:  request.pageQuery(),
		Filter: request.filter(),
	}
}

func (request listRequest) pageQuery() *paginatorv1.PageQuery {
	query := request.query.Page
	sorts := make([]*paginatorv1.Sort, 0, len(query.Sort))
	for _, sort := range query.Sort {
		sorts = append(sorts, &paginatorv1.Sort{
			Field: sort.Field,
			Desc:  sort.Desc,
		})
	}

	return &paginatorv1.PageQuery{
		Page:   query.Page,
		Limit:  query.Limit,
		Fields: append([]string(nil), query.Fields...),
		Sort:   sorts,
		Search: request.search(),
		LastId: query.LastID,
	}
}

func (request listRequest) search() *paginatorv1.Search {
	search := request.query.Page.Search
	if search == nil {
		return nil
	}

	mode := paginatorv1.SearchMode_SEARCH_MODE_UNSPECIFIED
	switch search.Mode {
	case pagination.SearchModeContains:
		mode = paginatorv1.SearchMode_SEARCH_MODE_CONTAINS
	case pagination.SearchModePrefix:
		mode = paginatorv1.SearchMode_SEARCH_MODE_PREFIX
	case pagination.SearchModeFullText:
		mode = paginatorv1.SearchMode_SEARCH_MODE_FULL_TEXT
	case pagination.SearchModeTrigram:
		mode = paginatorv1.SearchMode_SEARCH_MODE_TRIGRAM
	case pagination.SearchModeFullTextTrigram:
		mode = paginatorv1.SearchMode_SEARCH_MODE_FULL_TEXT_TRIGRAM
	}

	return &paginatorv1.Search{
		Fields:  append([]string(nil), search.Fields...),
		Keyword: search.Keyword,
		Mode:    mode,
	}
}

func (request listRequest) filter() *userv1.UserFilter {
	filter := request.query.Filter
	var createdFrom, createdTo *timestamppb.Timestamp
	if filter.CreatedFrom != nil {
		createdFrom = timestamppb.New(*filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		createdTo = timestamppb.New(*filter.CreatedTo)
	}

	return &userv1.UserFilter{
		CreatedFrom: createdFrom,
		CreatedTo:   createdTo,
		RoleCodes:   append([]uint64(nil), filter.RoleCodes...),
		Statuses:    append([]string(nil), filter.Statuses...),
	}
}
