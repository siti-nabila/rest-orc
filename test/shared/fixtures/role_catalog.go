package fixtures

import (
	paginatorv1 "github.com/siti-nabila/api-contracts/pb/paginator/v1"
	permissionv1 "github.com/siti-nabila/api-contracts/pb/permission/v1"
	rolev1 "github.com/siti-nabila/api-contracts/pb/role/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func RoleSummary() *rolev1.RoleSummary {
	description := "Reviewer legal kontrak"
	return &rolev1.RoleSummary{Id: 10, Code: "clm.legal_reviewer", Name: " Legacy NAME\té ", Description: &description, ApplicationId: 5, ApplicationCode: "clm", Status: "DISABLED", IsSystem: false, Version: 9007199254740993, RoleCode: 3, CreatedAt: timestamppb.New(MeCreatedAt()), AffectedUserCount: 0}
}
func RoleList() *rolev1.ListRolesResponse {
	return &rolev1.ListRolesResponse{Items: []*rolev1.RoleSummary{RoleSummary()}, Meta: &paginatorv1.PageMeta{Page: 1, Limit: 20, Total: 1, TotalPages: 1}}
}
func RoleDetail() *rolev1.GetRoleResponse {
	return &rolev1.GetRoleResponse{Role: RoleSummary(), Permissions: []*permissionv1.PermissionItem{{Id: 7, ApplicationId: 5, ApplicationCode: "clm", Code: "clm.contract.read", Description: "Read contracts"}}}
}

const RoleJSON = `{"id":10,"code":"clm.legal_reviewer","name":" Legacy NAME\té ","description":"Reviewer legal kontrak","application_id":5,"application_code":"clm","status":"DISABLED","is_system":false,"version":"9007199254740993","role_code":3,"created_at":"2026-10-20T08:09:10.123456789Z","updated_at":null,"affected_user_count":0}`
const RolePermissionsJSON = `[{"id":7,"application_id":5,"application_code":"clm","code":"clm.contract.read","description":"Read contracts"}]`
