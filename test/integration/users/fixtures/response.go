package fixtures

import userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"

const SuccessBody = `{"code":"SS","message":"Users retrieved successfully.","data":{"items":[{"id":17,"email":"admin@example.com","name":"Admin User","address":"Jakarta","phone":"+621111111"}],"total":1,"page":2,"limit":20,"total_pages":1,"has_next":false,"has_prev":true,"next_cursor":"cursor-40"}}`

func MeResponse() *userv1.UserData {
	return &userv1.UserData{
		Id:        17,
		Email:     "admin@example.com",
		Fullname:  "Admin User",
		RoleCodes: []int32{2},
		RoleNames: []string{"admin"},
	}
}

func ListUsersResponse() *userv1.ListUsersResponse {
	return &userv1.ListUsersResponse{
		Items: []*userv1.UserListItem{
			{
				Id:      17,
				Email:   "admin@example.com",
				Name:    "Admin User",
				Address: "Jakarta",
				Phone:   "+621111111",
			},
		},
		Total:      1,
		Page:       2,
		Limit:      20,
		TotalPages: 1,
		HasPrev:    true,
		NextCursor: "cursor-40",
	}
}
