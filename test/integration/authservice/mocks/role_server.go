package mocks

import (
	"context"

	rolev1 "github.com/siti-nabila/api-contracts/pb/role/v1"
	shared "github.com/siti-nabila/rest-orc/test/shared/fixtures"
)

type RoleServer struct {
	rolev1.UnimplementedRoleServiceServer
}

func (*RoleServer) ListRoles(context.Context, *rolev1.ListRolesRequest) (*rolev1.ListRolesResponse, error) {
	return shared.RoleList(), nil
}
func (*RoleServer) GetRole(context.Context, *rolev1.GetRoleRequest) (*rolev1.GetRoleResponse, error) {
	return shared.RoleDetail(), nil
}
