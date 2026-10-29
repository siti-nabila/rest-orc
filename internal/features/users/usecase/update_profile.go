package usecase

import (
	"context"
	"strconv"

	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

type UpdateProfile struct{ updater domain.ProfileUpdater }

func NewUpdateProfile(updater domain.ProfileUpdater) (*UpdateProfile, error) {
	if updater == nil {
		return nil, dictionary.ErrUsersUseCaseGatewayRequired
	}
	return &UpdateProfile{updater: updater}, nil
}

func (usecase *UpdateProfile) Execute(ctx context.Context, actor domain.ProfileActor, request domain.UpdateProfileRequest) (domain.UpdateProfileResponse, error) {
	if ctx == nil {
		return domain.UpdateProfileResponse{}, dictionary.ErrUsersUseCaseContextRequired
	}
	if actor.UserID == 0 {
		return domain.UpdateProfileResponse{}, common.ErrUnauthorized
	}
	if actor.MustChangePassword {
		return domain.UpdateProfileResponse{}, common.ErrForbidden
	}
	normalized, err := request.NormalizeAndValidate()
	if err != nil {
		return domain.UpdateProfileResponse{}, err
	}
	if !normalized.HasFields() {
		return domain.UpdateProfileResponse{}, nil
	}
	result, err := usecase.updater.UpdateProfile(ctx, normalized)
	if err != nil {
		return domain.UpdateProfileResponse{}, err
	}
	if result.Profile == nil || result.Profile.ID == 0 || result.Profile.UserID != actor.UserID || result.Version == nil {
		return domain.UpdateProfileResponse{}, common.ErrInternalServerError
	}
	version, err := strconv.ParseUint(*result.Version, 10, 64)
	if err != nil || version == 0 {
		return domain.UpdateProfileResponse{}, common.ErrInternalServerError
	}
	return result, nil
}
