package fixtures

import (
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/pkg/pagination"
)

const (
	SuccessBody         = `{"code":"SS","message":"Users retrieved successfully.","data":{"items":[{"id":17,"email":"admin@example.com","name":"Admin User","address":"Jakarta","phone":"+621111111"}],"total":42,"page":2,"limit":20,"total_pages":3,"has_next":true,"has_prev":true,"next_cursor":"cursor-40"}}`
	RegisterSuccessBody = `{"code":"SS","message":"Register Successfully","data":{"Token":"register-token"}}`
	LoginSuccessBody    = `{"code":"SS","message":"Login Successfully","data":{"Token":"login-token"}}`
)

func Page() pagination.Page[domain.ListItem] {
	return pagination.Page[domain.ListItem]{
		Items: []domain.ListItem{
			{
				ID:      17,
				Email:   "admin@example.com",
				Name:    "Admin User",
				Address: "Jakarta",
				Phone:   "+621111111",
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
