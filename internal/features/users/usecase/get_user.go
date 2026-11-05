package usecase

import (
	"context"

	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

type GetUser struct{ gateway domain.GetUserGateway }

func NewGetUser(gateway domain.GetUserGateway) (*GetUser, error) {
	if gateway == nil {
		return nil, dictionary.ErrUsersUseCaseGatewayRequired
	}
	return &GetUser{gateway: gateway}, nil
}

func (uc *GetUser) Execute(ctx context.Context, actor domain.UserDetailActor, target uint64) (domain.UserDetail, error) {
	if ctx == nil || actor.UserID == 0 {
		return domain.UserDetail{}, common.ErrUnauthorized
	}
	if actor.MustChangePassword || !actor.Permissions[domain.UserReadPermission] {
		return domain.UserDetail{}, common.ErrForbidden
	}
	if err := domain.ValidateUserDetailID(target); err != nil {
		return domain.UserDetail{}, err
	}
	result, err := uc.gateway.GetUser(ctx, target)
	if err != nil {
		return domain.UserDetail{}, err
	}
	if result.ID != target {
		return domain.UserDetail{}, common.ErrInternalServerError
	}
	roles := make([]domain.UserDetailRole, len(result.Roles))
	copy(roles, result.Roles)
	applications := make([]domain.UserDetailApplicationAccess, len(result.ApplicationAccess))
	copy(applications, result.ApplicationAccess)
	result.Roles, result.ApplicationAccess = roles, applications
	return result, nil
}
