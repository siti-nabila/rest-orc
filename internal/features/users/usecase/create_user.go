package usecase

import (
	"context"

	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

type CreateUser struct {
	gateway domain.CreateUserGateway
}

func NewCreateUser(gateway domain.CreateUserGateway) (*CreateUser, error) {
	if gateway == nil {
		return nil, dictionary.ErrUsersUseCaseGatewayRequired
	}
	return &CreateUser{gateway: gateway}, nil
}

func (uc *CreateUser) Execute(ctx context.Context, actor domain.UserCreationActor, request domain.CreateUserRequest) (domain.CreateUserResponse, error) {
	if ctx == nil || actor.UserID == 0 {
		return domain.CreateUserResponse{}, common.ErrUnauthorized
	}
	if actor.MustChangePassword || !actor.Permissions[domain.UserCreatePermission] {
		return domain.CreateUserResponse{}, common.ErrForbidden
	}
	normalized, err := request.NormalizeAndValidate()
	if err != nil {
		return domain.CreateUserResponse{}, err
	}
	result, err := uc.gateway.CreateUser(ctx, normalized)
	if err != nil {
		return domain.CreateUserResponse{}, err
	}
	if result.User == nil {
		return domain.CreateUserResponse{}, common.ErrInternalServerError
	}
	user := *result.User
	user.Roles = make([]domain.UserDetailRole, len(result.User.Roles))
	copy(user.Roles, result.User.Roles)
	user.ApplicationAccess = make([]domain.UserDetailApplicationAccess, len(result.User.ApplicationAccess))
	copy(user.ApplicationAccess, result.User.ApplicationAccess)
	result.User = &user
	return result, nil
}
