package fixtures

import (
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func UserDetailResponse(id uint64) *userv1.GetUserResponse {
	return &userv1.GetUserResponse{User: &userv1.User{
		Id: id, Email: "target@example.com", Name: " Legacy NAME ", Address: "Legacy\taddress",
		PhoneNo: "+62+628123", CountryCode: "+62", Status: "ACTIVE", Version: 9007199254740993,
		CreatedAt: timestamppb.New(MeCreatedAt()), Password: "private-password", Dob: "private-dob",
		Roles: []*userv1.UserRoleAssignment{
			{RoleId: 11, Code: "iam.reader", Name: "Reader", Status: "REVOKED", RoleStatus: "DISABLED"},
			{RoleId: 9, Code: "clm.member", Name: "Member", Status: "ACTIVE", RoleStatus: "ACTIVE"},
		},
		ApplicationAccess: []*userv1.UserApplicationAccess{{ApplicationId: 5, Code: "clm", Name: "CLM", Status: "REVOKED", ApplicationStatus: "DISABLED"}},
	}}
}

const UserDetailBody = `{"code":"SS","message":"User retrieved successfully.","data":{"id":23,"email":"target@example.com","name":" Legacy NAME ","address":"Legacy\taddress","phone":"+62+628123","country_code":"+62","status":"ACTIVE","version":"9007199254740993","created_at":"2026-10-20T08:09:10.123456789Z","must_change_password":false,"roles":[{"role_id":11,"code":"iam.reader","name":"Reader","status":"REVOKED","role_status":"DISABLED"},{"role_id":9,"code":"clm.member","name":"Member","status":"ACTIVE","role_status":"ACTIVE"}],"application_access":[{"application_id":5,"code":"clm","name":"CLM","status":"REVOKED","application_status":"DISABLED"}]}}`
