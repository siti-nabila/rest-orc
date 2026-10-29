package fixtures

import (
	"time"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	requestauth "github.com/siti-nabila/rest-orc/internal/auth"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func MeUserData() *userv1.UserData {
	return &userv1.UserData{
		Id: 17, Email: "member@example.com", Fullname: "Member User",
		Address: "Jakarta", Phone: "+62+628123", CountryCode: "+62",
		CreatedAt: timestamppb.New(MeCreatedAt()), Status: "ACTIVE", Version: 9007199254740993,
		RoleCodes: []int32{3}, RoleNames: []string{"member"},
		Roles: []*userv1.UserRoleAssignment{
			{RoleId: 9, Code: "clm.member", Name: "Member", Status: "ACTIVE", RoleStatus: "ACTIVE"},
			{RoleId: 11, Code: "clm.reader", Name: "Reader", Status: "ACTIVE", RoleStatus: "ACTIVE"},
		},
		ApplicationAccess: []*userv1.UserApplicationAccess{
			{ApplicationId: 5, Code: "clm", Name: "CLM", Status: "ACTIVE", ApplicationStatus: "ACTIVE"},
		},
		Permissions: map[string]bool{"clm.document.read": true, "clm.document.delete": false},
	}
}

func MeCreatedAt() time.Time {
	return time.Date(2026, time.October, 20, 8, 9, 10, 123456789, time.UTC)
}

func MePrincipal() requestauth.Principal {
	createdAt := MeCreatedAt()
	return requestauth.Principal{
		ID: 17, Email: "member@example.com", Fullname: "Member User",
		Address: "Jakarta", Phone: "+62+628123", CountryCode: "+62",
		CreatedAt: &createdAt, Status: "ACTIVE", Version: 9007199254740993,
		RoleCodes: []int32{3}, RoleNames: []string{"member"},
		Roles: []requestauth.RoleAssignment{
			{RoleID: 9, Code: "clm.member", Name: "Member", Status: "ACTIVE", RoleStatus: "ACTIVE"},
			{RoleID: 11, Code: "clm.reader", Name: "Reader", Status: "ACTIVE", RoleStatus: "ACTIVE"},
		},
		ApplicationAccess: []requestauth.ApplicationAccess{
			{ApplicationID: 5, Code: "clm", Name: "CLM", Status: "ACTIVE", ApplicationStatus: "ACTIVE"},
		},
		Permissions: map[string]bool{"clm.document.read": true, "clm.document.delete": false},
	}
}
