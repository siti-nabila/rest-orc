package authservice

import (
	"context"

	rolev1 "github.com/siti-nabila/api-contracts/pb/role/v1"
	"github.com/siti-nabila/rest-orc/internal/client/grpcclient"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"google.golang.org/grpc"
)

// RoleCatalogService isolates only the generated role reads on the existing transport.
type RoleCatalogService interface {
	ListRoles(context.Context, *rolev1.ListRolesRequest, ...grpc.CallOption) (*rolev1.ListRolesResponse, error)
	GetRole(context.Context, *rolev1.GetRoleRequest, ...grpc.CallOption) (*rolev1.GetRoleResponse, error)
}

func NewWithRoleCatalogService(transport *grpcclient.Client, service Service, profile ProfileService, password PasswordService, session SessionService, self SelfSessionService, admin AdminSessionService, detail UserDetailService, status UserStatusService, userProfile UserProfileService, userCreation UserCreationService, roles RoleCatalogService) (*Client, error) {
	c, err := NewWithUserCreationService(transport, service, profile, password, session, self, admin, detail, status, userProfile, userCreation)
	if err != nil {
		return nil, err
	}
	if roles == nil {
		return nil, dictionary.ErrRoleCatalogServiceRequired
	}
	c.roleCatalogService = roles
	return c, nil
}
func (c *Client) ListRoles(ctx context.Context, r *rolev1.ListRolesRequest) (*rolev1.ListRolesResponse, error) {
	if ctx == nil {
		return nil, dictionary.ErrGRPCAuthRequestContextRequired
	}
	if r == nil {
		return nil, dictionary.ErrListRolesRequestRequired
	}
	child, cancel := c.transport.RequestContext(ctx)
	defer cancel()
	out, err := c.roleCatalogService.ListRoles(child, r)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, dictionary.ErrListRolesResponseRequired
	}
	return out, nil
}
func (c *Client) GetRole(ctx context.Context, r *rolev1.GetRoleRequest) (*rolev1.GetRoleResponse, error) {
	if ctx == nil {
		return nil, dictionary.ErrGRPCAuthRequestContextRequired
	}
	if r == nil {
		return nil, dictionary.ErrGetRoleRequestRequired
	}
	child, cancel := c.transport.RequestContext(ctx)
	defer cancel()
	out, err := c.roleCatalogService.GetRole(child, r)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, dictionary.ErrGetRoleResponseRequired
	}
	return out, nil
}
