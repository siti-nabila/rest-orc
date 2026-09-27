package grpcauth

import (
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/pkg/pagination"
)

type listResponse struct {
	value *userv1.ListUsersResponse
}

func (response listResponse) page() (pagination.Page[domain.ListItem], error) {
	items := make([]domain.ListItem, 0, len(response.value.GetItems()))
	for _, item := range response.value.GetItems() {
		if item == nil {
			return pagination.Page[domain.ListItem]{},
				dictionary.ErrUsersRepositoryListItemRequired
		}
		items = append(items, domain.ListItem{
			ID:      item.GetId(),
			Email:   item.GetEmail(),
			Name:    item.GetName(),
			Address: item.GetAddress(),
			Phone:   item.GetPhone(),
		})
	}

	return pagination.Page[domain.ListItem]{
		Items:      items,
		Total:      response.value.GetTotal(),
		Page:       response.value.GetPage(),
		Limit:      response.value.GetLimit(),
		TotalPages: response.value.GetTotalPages(),
		HasNext:    response.value.GetHasNext(),
		HasPrev:    response.value.GetHasPrev(),
		NextCursor: response.value.GetNextCursor(),
	}, nil
}
