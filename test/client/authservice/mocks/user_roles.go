package mocks

import (
	"context"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"google.golang.org/grpc"
)

type UserRoleManagementService struct {
	ListUserRolesFunc   func(context.Context, *userv1.ListUserRolesRequest) (*userv1.ListUserRolesResponse, error)
	GrantUserRolesFunc  func(context.Context, *userv1.GrantUserRolesRequest) (*userv1.GrantUserRolesResponse, error)
	RevokeUserRolesFunc func(context.Context, *userv1.RevokeUserRolesRequest) (*userv1.RevokeUserRolesResponse, error)
}

func (s *UserRoleManagementService) ListUserRoles(c context.Context, r *userv1.ListUserRolesRequest, _ ...grpc.CallOption) (*userv1.ListUserRolesResponse, error) {
	if s.ListUserRolesFunc == nil {
		panic("unexpected ListUserRoles")
	}
	return s.ListUserRolesFunc(c, r)
}
func (s *UserRoleManagementService) GrantUserRoles(c context.Context, r *userv1.GrantUserRolesRequest, _ ...grpc.CallOption) (*userv1.GrantUserRolesResponse, error) {
	if s.GrantUserRolesFunc == nil {
		panic("unexpected GrantUserRoles")
	}
	return s.GrantUserRolesFunc(c, r)
}
func (s *UserRoleManagementService) RevokeUserRoles(c context.Context, r *userv1.RevokeUserRolesRequest, _ ...grpc.CallOption) (*userv1.RevokeUserRolesResponse, error) {
	if s.RevokeUserRolesFunc == nil {
		panic("unexpected RevokeUserRoles")
	}
	return s.RevokeUserRolesFunc(c, r)
}
