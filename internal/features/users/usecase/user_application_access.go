package usecase

import (
	"context"

	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

func authorizeUserApplicationAccess(ctx context.Context, a domain.UserApplicationAccessActor, permissions ...string) error {
	if ctx == nil || a.UserID == 0 {
		return common.ErrUnauthorized
	}
	if a.MustChangePassword {
		return common.ErrForbidden
	}
	for _, permission := range permissions {
		if !a.Permissions[permission] {
			return common.ErrForbidden
		}
	}
	return nil
}

type ListUserApplicationAccess struct {
	gateway domain.ListUserApplicationAccessGateway
}

func NewListUserApplicationAccess(g domain.ListUserApplicationAccessGateway) (*ListUserApplicationAccess, error) {
	if g == nil {
		return nil, dictionary.ErrUsersUseCaseGatewayRequired
	}
	return &ListUserApplicationAccess{gateway: g}, nil
}
func (u *ListUserApplicationAccess) Execute(ctx context.Context, a domain.UserApplicationAccessActor, r domain.ListUserApplicationAccessRequest) (domain.ListUserApplicationAccessResponse, error) {
	if err := authorizeUserApplicationAccess(ctx, a, domain.UserApplicationReadPermission); err != nil {
		return domain.ListUserApplicationAccessResponse{}, err
	}
	id, err := domain.ParseUserDetailID(r.UserID)
	if err != nil {
		return domain.ListUserApplicationAccessResponse{}, err
	}
	q, err := domain.ParseUserApplicationAccessListQuery(r.Input)
	if err != nil {
		return domain.ListUserApplicationAccessResponse{}, err
	}
	out, err := u.gateway.ListUserApplicationAccess(ctx, id, q)
	if err != nil {
		return domain.ListUserApplicationAccessResponse{}, err
	}
	if out.UserID != id {
		return domain.ListUserApplicationAccessResponse{}, common.ErrInternalServerError
	}
	out.Items = append([]domain.UserApplicationAccess{}, out.Items...)
	for i := range out.Items {
		p := &out.Items[i]
		p.GrantedBy = copyUserRolePointer(p.GrantedBy)
		p.RevokedBy = copyUserRolePointer(p.RevokedBy)
		p.RevokedAt = copyUserRolePointer(p.RevokedAt)
		p.RevokeReason = copyUserRolePointer(p.RevokeReason)
	}
	return out, nil
}

type GrantApplicationAccess struct {
	gateway domain.GrantApplicationAccessGateway
}

func NewGrantApplicationAccess(g domain.GrantApplicationAccessGateway) (*GrantApplicationAccess, error) {
	if g == nil {
		return nil, dictionary.ErrUsersUseCaseGatewayRequired
	}
	return &GrantApplicationAccess{gateway: g}, nil
}
func (u *GrantApplicationAccess) Execute(ctx context.Context, a domain.UserApplicationAccessActor, r domain.GrantApplicationAccessRequest) (domain.ApplicationAccessCommandResponse, error) {
	if err := authorizeUserApplicationAccess(ctx, a, domain.UserApplicationGrantPermission, domain.UserRoleGrantPermission); err != nil {
		return domain.ApplicationAccessCommandResponse{}, err
	}
	command, err := r.NormalizeAndValidate()
	if err != nil {
		return domain.ApplicationAccessCommandResponse{}, err
	}
	out, err := u.gateway.GrantApplicationAccess(ctx, command)
	if err != nil {
		return domain.ApplicationAccessCommandResponse{}, err
	}
	if out.UserID != command.UserID {
		return domain.ApplicationAccessCommandResponse{}, common.ErrInternalServerError
	}
	out.AccessStatus = copyUserRolePointer(out.AccessStatus)
	out.ChangedRoleIDs = append([]uint64{}, out.ChangedRoleIDs...)
	return out, nil
}

type RevokeApplicationAccess struct {
	gateway domain.RevokeApplicationAccessGateway
}

func NewRevokeApplicationAccess(g domain.RevokeApplicationAccessGateway) (*RevokeApplicationAccess, error) {
	if g == nil {
		return nil, dictionary.ErrUsersUseCaseGatewayRequired
	}
	return &RevokeApplicationAccess{gateway: g}, nil
}
func (u *RevokeApplicationAccess) Execute(ctx context.Context, a domain.UserApplicationAccessActor, r domain.RevokeApplicationAccessRequest) (domain.ApplicationAccessCommandResponse, error) {
	if err := authorizeUserApplicationAccess(ctx, a, domain.UserApplicationRevokePermission, domain.UserRoleRevokePermission); err != nil {
		return domain.ApplicationAccessCommandResponse{}, err
	}
	command, err := r.NormalizeAndValidate()
	if err != nil {
		return domain.ApplicationAccessCommandResponse{}, err
	}
	out, err := u.gateway.RevokeApplicationAccess(ctx, command)
	if err != nil {
		return domain.ApplicationAccessCommandResponse{}, err
	}
	if out.UserID != command.UserID {
		return domain.ApplicationAccessCommandResponse{}, common.ErrInternalServerError
	}
	out.AccessStatus = copyUserRolePointer(out.AccessStatus)
	out.ChangedRoleIDs = append([]uint64{}, out.ChangedRoleIDs...)
	return out, nil
}
