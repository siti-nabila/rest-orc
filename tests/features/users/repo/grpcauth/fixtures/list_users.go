package fixtures

import (
	"time"

	paginatorv1 "github.com/siti-nabila/api-contracts/pb/paginator/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/pkg/pagination"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	CreatedFrom = time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	CreatedTo   = time.Date(2026, time.July, 31, 23, 59, 59, 0, time.UTC)
)

func Query() domain.ListQuery {
	return domain.ListQuery{
		Page: pagination.Query{
			Page:   2,
			Limit:  20,
			Fields: []string{"id", "email", "name"},
			Sort: []pagination.Sort{
				{Field: "name", Desc: false},
				{Field: "id", Desc: true},
			},
			Search: &pagination.Search{
				Keyword: "admin",
			},
			LastID: "cursor-20",
		},
		Filter: domain.ListFilter{
			CreatedFrom: &CreatedFrom,
			CreatedTo:   &CreatedTo,
			RoleCodes:   []uint64{1, 2},
		},
	}
}

func ExpectedRequest() *userv1.ListUsersRequest {
	return &userv1.ListUsersRequest{
		Query: &paginatorv1.PageQuery{
			Page:   2,
			Limit:  20,
			Fields: []string{"id", "email", "name"},
			Sort: []*paginatorv1.Sort{
				{Field: "name", Desc: false},
				{Field: "id", Desc: true},
			},
			Search: &paginatorv1.Search{
				Keyword: "admin",
			},
			LastId: "cursor-20",
		},
		Filter: &userv1.UserFilter{
			CreatedFrom: timestamppb.New(CreatedFrom),
			CreatedTo:   timestamppb.New(CreatedTo),
			RoleCodes:   []uint64{1, 2},
		},
	}
}

func Response() *userv1.ListUsersResponse {
	return &userv1.ListUsersResponse{
		Items: []*userv1.UserListItem{
			{
				Id:      17,
				Email:   "admin@example.com",
				Name:    "Admin User",
				Address: "Jakarta",
				Phone:   "+621111111",
			},
			{
				Id:      18,
				Email:   "operator@example.com",
				Name:    "Operator User",
				Address: "Bandung",
				Phone:   "+622222222",
			},
		},
		Total:      42,
		Page:       2,
		Limit:      20,
		TotalPages: 3,
		HasNext:    true,
		HasPrev:    true,
		NextCursor: "cursor-40",
	}
}

func ExpectedPage() pagination.Page[domain.ListItem] {
	return pagination.Page[domain.ListItem]{
		Items: []domain.ListItem{
			{
				ID:      17,
				Email:   "admin@example.com",
				Name:    "Admin User",
				Address: "Jakarta",
				Phone:   "+621111111",
			},
			{
				ID:      18,
				Email:   "operator@example.com",
				Name:    "Operator User",
				Address: "Bandung",
				Phone:   "+622222222",
			},
		},
		Total:      42,
		Page:       2,
		Limit:      20,
		TotalPages: 3,
		HasNext:    true,
		HasPrev:    true,
		NextCursor: "cursor-40",
	}
}
