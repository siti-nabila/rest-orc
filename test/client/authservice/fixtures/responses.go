package fixtures

import userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"

func MeResponse() *userv1.UserData {
	return &userv1.UserData{
		Id:        17,
		Email:     "admin@example.com",
		Fullname:  "Admin User",
		RoleNames: []string{"admin"},
	}
}

func ListUsersRequest() *userv1.ListUsersRequest {
	return &userv1.ListUsersRequest{}
}

func ListUsersResponse() *userv1.ListUsersResponse {
	return &userv1.ListUsersResponse{
		Items: []*userv1.UserListItem{
			{Id: 17, Email: "admin@example.com", Name: "Admin User"},
		},
		Total:      1,
		Page:       1,
		Limit:      10,
		TotalPages: 1,
	}
}
