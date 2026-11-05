package usecase

import (
	"context"

	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

type UpdateUserProfile struct {
	gateway domain.UpdateUserProfileGateway
}

func NewUpdateUserProfile(gateway domain.UpdateUserProfileGateway) (*UpdateUserProfile, error) {
	if gateway == nil {
		return nil, dictionary.ErrUsersUseCaseGatewayRequired
	}
	return &UpdateUserProfile{gateway: gateway}, nil
}

func (uc *UpdateUserProfile) Execute(ctx context.Context, actor domain.UserProfileActor, request domain.UpdateUserProfileRequest) (domain.UpdateUserProfileResponse, error) {
	if ctx == nil || actor.UserID == 0 {
		return domain.UpdateUserProfileResponse{}, common.ErrUnauthorized
	}
	if actor.MustChangePassword || !actor.Permissions[domain.UserUpdatePermission] {
		return domain.UpdateUserProfileResponse{}, common.ErrForbidden
	}
	normalized, err := request.NormalizeAndValidate()
	if err != nil {
		return domain.UpdateUserProfileResponse{}, err
	}
	if !normalized.Profile.HasFields() {
		return domain.UpdateUserProfileResponse{}, nil
	}
	result, err := uc.gateway.UpdateUserProfile(ctx, normalized)
	if err != nil {
		return domain.UpdateUserProfileResponse{}, err
	}
	if result.User == nil || result.User.ID != request.UserID {
		return domain.UpdateUserProfileResponse{}, common.ErrInternalServerError
	}
	user := *result.User
	user.Roles = make([]domain.UserDetailRole, len(result.User.Roles))
	copy(user.Roles, result.User.Roles)
	user.ApplicationAccess = make([]domain.UserDetailApplicationAccess, len(result.User.ApplicationAccess))
	copy(user.ApplicationAccess, result.User.ApplicationAccess)
	result.User = &user
	return result, nil
}
