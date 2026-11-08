package mocks

import (
	"context"
	applicationv1 "github.com/siti-nabila/api-contracts/pb/application/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"google.golang.org/grpc"
)

type ApplicationAccessService struct {
	Read   func(context.Context, *userv1.ListUserApplicationAccessRequest) (*userv1.ListUserApplicationAccessResponse, error)
	Grant  func(context.Context, *applicationv1.GrantApplicationAccessRequest) (*applicationv1.GrantApplicationAccessResponse, error)
	Revoke func(context.Context, *applicationv1.RevokeApplicationAccessRequest) (*applicationv1.RevokeApplicationAccessResponse, error)
}

func (s *ApplicationAccessService) ListUserApplicationAccess(c context.Context, r *userv1.ListUserApplicationAccessRequest, _ ...grpc.CallOption) (*userv1.ListUserApplicationAccessResponse, error) {
	if s.Read == nil {
		panic("unexpected ListUserApplicationAccess")
	}
	return s.Read(c, r)
}
func (s *ApplicationAccessService) GrantApplicationAccess(c context.Context, r *applicationv1.GrantApplicationAccessRequest, _ ...grpc.CallOption) (*applicationv1.GrantApplicationAccessResponse, error) {
	if s.Grant == nil {
		panic("unexpected GrantApplicationAccess")
	}
	return s.Grant(c, r)
}
func (s *ApplicationAccessService) RevokeApplicationAccess(c context.Context, r *applicationv1.RevokeApplicationAccessRequest, _ ...grpc.CallOption) (*applicationv1.RevokeApplicationAccessResponse, error) {
	if s.Revoke == nil {
		panic("unexpected RevokeApplicationAccess")
	}
	return s.Revoke(c, r)
}
