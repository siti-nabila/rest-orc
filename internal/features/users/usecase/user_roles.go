package usecase

import (
	"context"

	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

func authorizeUserRole(ctx context.Context, a domain.UserRoleActor, permission string) error {
	if ctx == nil || a.UserID == 0 {
		return common.ErrUnauthorized
	}
	if a.MustChangePassword || !a.Permissions[permission] {
		return common.ErrForbidden
	}
	return nil
}

type ListUserRoles struct{ gateway domain.ListUserRolesGateway }

func NewListUserRoles(g domain.ListUserRolesGateway) (*ListUserRoles, error) {
	if g == nil {
		return nil, dictionary.ErrUsersUseCaseGatewayRequired
	}
	return &ListUserRoles{gateway: g}, nil
}
func (u *ListUserRoles) Execute(ctx context.Context, a domain.UserRoleActor, r domain.ListUserRolesRequest) (domain.ListUserRolesResponse, error) {
	if err := authorizeUserRole(ctx, a, domain.UserRoleReadPermission); err != nil {
		return domain.ListUserRolesResponse{}, err
	}
	id, err := domain.ParseUserDetailID(r.UserID)
	if err != nil {
		return domain.ListUserRolesResponse{}, err
	}
	q, err := domain.ParseUserRoleListQuery(r.Input)
	if err != nil {
		return domain.ListUserRolesResponse{}, err
	}
	out, err := u.gateway.ListUserRoles(ctx, id, q)
	if err != nil {
		return domain.ListUserRolesResponse{}, err
	}
	if out.UserID != id {
		return domain.ListUserRolesResponse{}, common.ErrInternalServerError
	}
	out.Items = append([]domain.UserRoleAssignment{}, out.Items...)
	for i := range out.Items {
		p := &out.Items[i]
		p.ApplicationAccessStatus = copyUserRolePointer(p.ApplicationAccessStatus)
		p.GrantedBy = copyUserRolePointer(p.GrantedBy)
		p.GrantedAt = copyUserRolePointer(p.GrantedAt)
		p.RevokedBy = copyUserRolePointer(p.RevokedBy)
		p.RevokedAt = copyUserRolePointer(p.RevokedAt)
		p.RevokeReason = copyUserRolePointer(p.RevokeReason)
	}
	return out, nil
}
func copyUserRolePointer[T any](p *T) *T {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

type GrantUserRoles struct{ gateway domain.GrantUserRolesGateway }

func NewGrantUserRoles(g domain.GrantUserRolesGateway) (*GrantUserRoles, error) {
	if g == nil {
		return nil, dictionary.ErrUsersUseCaseGatewayRequired
	}
	return &GrantUserRoles{gateway: g}, nil
}
func (u *GrantUserRoles) Execute(ctx context.Context, a domain.UserRoleActor, r domain.UserRoleCommandRequest) (domain.UserRoleCommandResponse, error) {
	if err := authorizeUserRole(ctx, a, domain.UserRoleGrantPermission); err != nil {
		return domain.UserRoleCommandResponse{}, err
	}
	command, err := r.NormalizeAndValidate()
	if err != nil {
		return domain.UserRoleCommandResponse{}, err
	}
	out, err := u.gateway.GrantUserRoles(ctx, command)
	if err != nil {
		return domain.UserRoleCommandResponse{}, err
	}
	if out.UserID != command.UserID {
		return domain.UserRoleCommandResponse{}, common.ErrInternalServerError
	}
	out.ChangedRoleIDs = append([]uint64{}, out.ChangedRoleIDs...)
	return out, nil
}

type RevokeUserRoles struct{ gateway domain.RevokeUserRolesGateway }

func NewRevokeUserRoles(g domain.RevokeUserRolesGateway) (*RevokeUserRoles, error) {
	if g == nil {
		return nil, dictionary.ErrUsersUseCaseGatewayRequired
	}
	return &RevokeUserRoles{gateway: g}, nil
}
func (u *RevokeUserRoles) Execute(ctx context.Context, a domain.UserRoleActor, r domain.UserRoleCommandRequest) (domain.UserRoleCommandResponse, error) {
	if err := authorizeUserRole(ctx, a, domain.UserRoleRevokePermission); err != nil {
		return domain.UserRoleCommandResponse{}, err
	}
	command, err := r.NormalizeAndValidate()
	if err != nil {
		return domain.UserRoleCommandResponse{}, err
	}
	out, err := u.gateway.RevokeUserRoles(ctx, command)
	if err != nil {
		return domain.UserRoleCommandResponse{}, err
	}
	if out.UserID != command.UserID {
		return domain.UserRoleCommandResponse{}, common.ErrInternalServerError
	}
	out.ChangedRoleIDs = append([]uint64{}, out.ChangedRoleIDs...)
	return out, nil
}
