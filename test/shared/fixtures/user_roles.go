package fixtures

import (
	paginatorv1 "github.com/siti-nabila/api-contracts/pb/paginator/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func UserRoleAssignment() *userv1.UserRoleAssignmentItem {
	return &userv1.UserRoleAssignmentItem{Id: 101, RoleId: 10, Code: "clm.legal_reviewer", RoleCode: 3, Name: " Legacy NAME\té ", Status: "REVOKED", RoleStatus: "DISABLED", ApplicationId: 5, ApplicationCode: "clm", ApplicationStatus: "DISABLED", GrantedAt: timestamppb.New(MeCreatedAt())}
}
func UserRoleList(target uint64) *userv1.ListUserRolesResponse {
	return &userv1.ListUserRolesResponse{UserId: target, Version: 9007199254740993, Items: []*userv1.UserRoleAssignmentItem{UserRoleAssignment()}, Meta: &paginatorv1.PageMeta{Page: 1, Limit: 20, Total: 1, TotalPages: 1}}
}

const UserRoleAssignmentJSON = `{"id":101,"role_id":10,"code":"clm.legal_reviewer","role_code":3,"name":" Legacy NAME\té ","status":"REVOKED","role_status":"DISABLED","application_id":5,"application_code":"clm","application_status":"DISABLED","is_system":false,"application_access_status":null,"granted_by":null,"granted_at":"2026-10-20T08:09:10.123456789Z","revoked_by":null,"revoked_at":null,"revoke_reason":null}`
