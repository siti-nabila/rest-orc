package usecase

import (
	"context"

	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

type ChangeUserStatus struct {
	gateway domain.ChangeUserStatusGateway
}

func NewChangeUserStatus(gateway domain.ChangeUserStatusGateway) (*ChangeUserStatus, error) {
	if gateway == nil {
		return nil, dictionary.ErrUsersUseCaseGatewayRequired
	}
	return &ChangeUserStatus{gateway: gateway}, nil
}

func (uc *ChangeUserStatus) Execute(ctx context.Context, actor domain.UserStatusActor, request domain.ChangeUserStatusRequest) (domain.ChangeUserStatusResponse, error) {
	if ctx == nil || actor.UserID == 0 {
		return domain.ChangeUserStatusResponse{}, common.ErrUnauthorized
	}
	if actor.MustChangePassword || !actor.Permissions[domain.UserStatusUpdatePermission] {
		return domain.ChangeUserStatusResponse{}, common.ErrForbidden
	}
	normalized, err := request.NormalizeAndValidate()
	if err != nil {
		return domain.ChangeUserStatusResponse{}, err
	}
	result, err := uc.gateway.ChangeUserStatus(ctx, normalized)
	if err != nil {
		return domain.ChangeUserStatusResponse{}, err
	}
	if result.UserID != request.UserID {
		return domain.ChangeUserStatusResponse{}, common.ErrInternalServerError
	}
	return result, nil
}
