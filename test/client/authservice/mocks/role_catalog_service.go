package mocks

import (
	"context"

	rolev1 "github.com/siti-nabila/api-contracts/pb/role/v1"
	"google.golang.org/grpc"
)

type RoleCatalogService struct {
	List func(context.Context, *rolev1.ListRolesRequest) (*rolev1.ListRolesResponse, error)
	Get  func(context.Context, *rolev1.GetRoleRequest) (*rolev1.GetRoleResponse, error)
}

func (s *RoleCatalogService) ListRoles(c context.Context, r *rolev1.ListRolesRequest, _ ...grpc.CallOption) (*rolev1.ListRolesResponse, error) {
	if s.List == nil {
		panic("unexpected ListRoles")
	}
	return s.List(c, r)
}
func (s *RoleCatalogService) GetRole(c context.Context, r *rolev1.GetRoleRequest, _ ...grpc.CallOption) (*rolev1.GetRoleResponse, error) {
	if s.Get == nil {
		panic("unexpected GetRole")
	}
	return s.Get(c, r)
}
