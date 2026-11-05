package usecase

import (
	"context"

	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

type ListUserSessions struct {
	gateway domain.ListUserSessionsGateway
}
type RevokeUserSession struct {
	gateway domain.RevokeUserSessionGateway
}
type RevokeAllUserSessions struct {
	gateway domain.RevokeAllUserSessionsGateway
}

func NewListUserSessions(gateway domain.ListUserSessionsGateway) (*ListUserSessions, error) {
	if gateway == nil {
		return nil, dictionary.ErrUsersUseCaseGatewayRequired
	}
	return &ListUserSessions{gateway: gateway}, nil
}

func NewRevokeUserSession(gateway domain.RevokeUserSessionGateway) (*RevokeUserSession, error) {
	if gateway == nil {
		return nil, dictionary.ErrUsersUseCaseGatewayRequired
	}
	return &RevokeUserSession{gateway: gateway}, nil
}

func NewRevokeAllUserSessions(gateway domain.RevokeAllUserSessionsGateway) (*RevokeAllUserSessions, error) {
	if gateway == nil {
		return nil, dictionary.ErrUsersUseCaseGatewayRequired
	}
	return &RevokeAllUserSessions{gateway: gateway}, nil
}

func adminSessionActorError(ctx context.Context, actor domain.AdminSessionActor, permission string) error {
	if ctx == nil || actor.UserID == 0 {
		return common.ErrUnauthorized
	}
	if actor.MustChangePassword || !actor.Permissions[permission] {
		return common.ErrForbidden
	}
	return nil
}

func (uc *ListUserSessions) Execute(ctx context.Context, actor domain.AdminSessionActor, target uint64, query domain.AdminSessionsQuery) (domain.AdminSessionsPage, error) {
	if err := adminSessionActorError(ctx, actor, domain.SessionReadPermission); err != nil {
		return domain.AdminSessionsPage{}, err
	}
	if err := domain.ValidateAdminSessionUserID(target); err != nil {
		return domain.AdminSessionsPage{}, err
	}
	if err := query.Validate(); err != nil {
		return domain.AdminSessionsPage{}, err
	}
	result, err := uc.gateway.ListUserSessions(ctx, target, query)
	if err != nil {
		return domain.AdminSessionsPage{}, err
	}
	items := make([]domain.AdminSession, len(result.Items))
	for i, item := range result.Items {
		if item.UserID != target {
			return domain.AdminSessionsPage{}, dictionary.ErrSessionResultInvalid
		}
		items[i] = item
		if item.LastSeenAt != nil {
			value := *item.LastSeenAt
			items[i].LastSeenAt = &value
		}
		if item.RevokedAt != nil {
			value := *item.RevokedAt
			items[i].RevokedAt = &value
		}
	}
	result.Items = items
	return result, nil
}

func (uc *RevokeUserSession) Execute(ctx context.Context, actor domain.AdminSessionActor, request domain.RevokeUserSessionRequest) (domain.RevokeUserSessionResponse, error) {
	if err := adminSessionActorError(ctx, actor, domain.SessionRevokePermission); err != nil {
		return domain.RevokeUserSessionResponse{}, err
	}
	if err := request.Validate(); err != nil {
		return domain.RevokeUserSessionResponse{}, err
	}
	return uc.gateway.RevokeUserSession(ctx, request)
}

func (uc *RevokeAllUserSessions) Execute(ctx context.Context, actor domain.AdminSessionActor, request domain.RevokeAllUserSessionsRequest) (domain.RevokeAllUserSessionsResponse, error) {
	if err := adminSessionActorError(ctx, actor, domain.SessionRevokePermission); err != nil {
		return domain.RevokeAllUserSessionsResponse{}, err
	}
	if err := request.Validate(); err != nil {
		return domain.RevokeAllUserSessionsResponse{}, err
	}
	return uc.gateway.RevokeAllUserSessions(ctx, request)
}
