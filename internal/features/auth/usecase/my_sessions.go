package usecase

import (
	"context"

	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

type ListMySessions struct{ gateway domain.ListMySessionsGateway }
type RevokeMySession struct{ gateway domain.RevokeMySessionGateway }
type RevokeAllMySessions struct {
	gateway domain.RevokeAllMySessionsGateway
}

func NewListMySessions(gateway domain.ListMySessionsGateway) (*ListMySessions, error) {
	if gateway == nil {
		return nil, dictionary.ErrAuthUseCaseGatewayRequired
	}
	return &ListMySessions{gateway: gateway}, nil
}

func NewRevokeMySession(gateway domain.RevokeMySessionGateway) (*RevokeMySession, error) {
	if gateway == nil {
		return nil, dictionary.ErrAuthUseCaseGatewayRequired
	}
	return &RevokeMySession{gateway: gateway}, nil
}

func NewRevokeAllMySessions(gateway domain.RevokeAllMySessionsGateway) (*RevokeAllMySessions, error) {
	if gateway == nil {
		return nil, dictionary.ErrAuthUseCaseGatewayRequired
	}
	return &RevokeAllMySessions{gateway: gateway}, nil
}

func sessionActorError(ctx context.Context, actor domain.SessionActor) error {
	if ctx == nil || actor.UserID == 0 {
		return common.ErrUnauthorized
	}
	if actor.MustChangePassword {
		return common.ErrForbidden
	}
	return nil
}

func (uc *ListMySessions) Execute(ctx context.Context, actor domain.SessionActor, query domain.MySessionsQuery) (domain.MySessionsPage, error) {
	if err := sessionActorError(ctx, actor); err != nil {
		return domain.MySessionsPage{}, err
	}
	if err := query.Validate(); err != nil {
		return domain.MySessionsPage{}, err
	}
	result, err := uc.gateway.ListMySessions(ctx, query)
	if err != nil {
		return domain.MySessionsPage{}, err
	}
	items := make([]domain.MySession, len(result.Items))
	for i, item := range result.Items {
		if item.UserID != actor.UserID {
			return domain.MySessionsPage{}, dictionary.ErrSessionResultInvalid
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

func (uc *RevokeMySession) Execute(ctx context.Context, actor domain.SessionActor, id string) (domain.RevokeMySessionResponse, error) {
	if err := sessionActorError(ctx, actor); err != nil {
		return domain.RevokeMySessionResponse{}, err
	}
	if err := domain.ValidateSessionID(id); err != nil {
		return domain.RevokeMySessionResponse{}, err
	}
	return uc.gateway.RevokeMySession(ctx, id)
}

func (uc *RevokeAllMySessions) Execute(ctx context.Context, actor domain.SessionActor) (domain.RevokeAllMySessionsResponse, error) {
	if err := sessionActorError(ctx, actor); err != nil {
		return domain.RevokeAllMySessionsResponse{}, err
	}
	return uc.gateway.RevokeAllMySessions(ctx)
}
