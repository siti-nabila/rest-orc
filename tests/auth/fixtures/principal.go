package fixtures

import (
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	requestauth "github.com/siti-nabila/rest-orc/internal/auth"
)

func UserData() *userv1.UserData {
	return &userv1.UserData{
		Id:        17,
		Email:     "admin@example.com",
		Fullname:  "Admin User",
		RoleIds:   []uint64{7},
		RoleNames: []string{"admin"},
	}
}

func Principal() requestauth.Principal {
	return requestauth.Principal{
		ID:        17,
		Email:     "admin@example.com",
		Fullname:  "Admin User",
		RoleIDs:   []uint64{7},
		RoleNames: []string{"admin"},
	}
}
