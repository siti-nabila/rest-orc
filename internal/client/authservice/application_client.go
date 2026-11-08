package authservice

import (
	"context"

	applicationv1 "github.com/siti-nabila/api-contracts/pb/application/v1"
	"github.com/siti-nabila/rest-orc/internal/client/grpcclient"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"google.golang.org/grpc"
)

// ApplicationCatalogService contains only the catalog read on the existing connection.
type ApplicationCatalogService interface {
	ListApplications(context.Context, *applicationv1.ListApplicationsRequest, ...grpc.CallOption) (*applicationv1.ListApplicationsResponse, error)
}

// NewWithApplicationCatalogService preserves old constructor guards and SDK defaults.
func NewWithApplicationCatalogService(transport *grpcclient.Client, service Service, profile ProfileService, password PasswordService, session SessionService, self SelfSessionService, admin AdminSessionService, detail UserDetailService, status UserStatusService, userProfile UserProfileService, creation UserCreationService, roles RoleCatalogService, assignments UserRoleManagementService, applications ApplicationCatalogService) (*Client, error) {
	c, err := NewWithUserRoleManagementService(transport, service, profile, password, session, self, admin, detail, status, userProfile, creation, roles, assignments)
	if err != nil {
		return nil, err
	}
	if applications == nil {
		return nil, dictionary.ErrApplicationCatalogServiceRequired
	}
	c.applicationCatalogService = applications
	return c, nil
}
func (c *Client) ListApplications(ctx context.Context, r *applicationv1.ListApplicationsRequest) (*applicationv1.ListApplicationsResponse, error) {
	if ctx == nil {
		return nil, dictionary.ErrGRPCAuthRequestContextRequired
	}
	if r == nil {
		return nil, dictionary.ErrListApplicationsRequestRequired
	}
	child, cancel := c.transport.RequestContext(ctx)
	defer cancel()
	out, err := c.applicationCatalogService.ListApplications(child, r)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, dictionary.ErrListApplicationsResponseRequired
	}
	return out, nil
}

// ApplicationAccessService owns the two compound ApplicationService commands.
type ApplicationAccessService interface {
	GrantApplicationAccess(context.Context, *applicationv1.GrantApplicationAccessRequest, ...grpc.CallOption) (*applicationv1.GrantApplicationAccessResponse, error)
	RevokeApplicationAccess(context.Context, *applicationv1.RevokeApplicationAccessRequest, ...grpc.CallOption) (*applicationv1.RevokeApplicationAccessResponse, error)
}

// NewWithApplicationAccessServices preserves earlier constructor guards and SDK defaults.
func NewWithApplicationAccessServices(transport *grpcclient.Client, service Service, profile ProfileService, password PasswordService, session SessionService, self SelfSessionService, admin AdminSessionService, detail UserDetailService, status UserStatusService, userProfile UserProfileService, creation UserCreationService, roles RoleCatalogService, assignments UserRoleManagementService, applications ApplicationCatalogService, reads UserApplicationAccessService, commands ApplicationAccessService) (*Client, error) {
	c, err := NewWithApplicationCatalogService(transport, service, profile, password, session, self, admin, detail, status, userProfile, creation, roles, assignments, applications)
	if err != nil {
		return nil, err
	}
	if reads == nil {
		return nil, dictionary.ErrUserApplicationAccessServiceRequired
	}
	if commands == nil {
		return nil, dictionary.ErrApplicationAccessServiceRequired
	}
	c.userApplicationAccessService, c.applicationAccessService = reads, commands
	return c, nil
}

func (c *Client) GrantApplicationAccess(ctx context.Context, r *applicationv1.GrantApplicationAccessRequest) (*applicationv1.GrantApplicationAccessResponse, error) {
	if ctx == nil {
		return nil, dictionary.ErrGRPCAuthRequestContextRequired
	}
	if r == nil {
		return nil, dictionary.ErrGrantApplicationAccessRequestRequired
	}
	child, cancel := c.transport.RequestContext(ctx)
	defer cancel()
	out, err := c.applicationAccessService.GrantApplicationAccess(child, r)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, dictionary.ErrGrantApplicationAccessResponseRequired
	}
	return out, nil
}

func (c *Client) RevokeApplicationAccess(ctx context.Context, r *applicationv1.RevokeApplicationAccessRequest) (*applicationv1.RevokeApplicationAccessResponse, error) {
	if ctx == nil {
		return nil, dictionary.ErrGRPCAuthRequestContextRequired
	}
	if r == nil {
		return nil, dictionary.ErrRevokeApplicationAccessRequestRequired
	}
	child, cancel := c.transport.RequestContext(ctx)
	defer cancel()
	out, err := c.applicationAccessService.RevokeApplicationAccess(child, r)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, dictionary.ErrRevokeApplicationAccessResponseRequired
	}
	return out, nil
}
