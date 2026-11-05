package authservice

import (
	"context"

	"time"

	paginatorv1 "github.com/siti-nabila/api-contracts/pb/paginator/v1"
	sessionv1 "github.com/siti-nabila/api-contracts/pb/session/v1"
	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SelfSessionClient interface {
	ListMySessions(context.Context, *sessionv1.ListMySessionsRequest) (*sessionv1.ListSessionsResponse, error)
	RevokeMySession(context.Context, *sessionv1.RevokeMySessionRequest) (*sessionv1.RevokeSessionResponse, error)
	RevokeAllMySessions(context.Context) (*sessionv1.RevokeAllMySessionsResponse, error)
}
type MySessionsGateway struct{ client SelfSessionClient }

func NewMySessions(client SelfSessionClient) (*MySessionsGateway, error) {
	if client == nil {
		return nil, dictionary.ErrAuthGatewayClientRequired
	}
	return &MySessionsGateway{client: client}, nil
}

func (g *MySessionsGateway) ListMySessions(ctx context.Context, q domain.MySessionsQuery) (domain.MySessionsPage, error) {
	if ctx == nil {
		return domain.MySessionsPage{}, dictionary.ErrAuthGatewayContextRequired
	}
	if err := q.Validate(); err != nil {
		return domain.MySessionsPage{}, err
	}
	result, err := g.client.ListMySessions(ctx, &sessionv1.ListMySessionsRequest{
		Query: &paginatorv1.PageQuery{
			Page: q.Page, Limit: q.Limit, LastId: q.LastID,
			Sort: []*paginatorv1.Sort{
				{Field: "created_at", Desc: true},
				{Field: "id", Desc: true},
			},
		},
	})
	if err != nil {
		return domain.MySessionsPage{}, err
	}
	return mySessionsPage(result)
}

func (g *MySessionsGateway) RevokeMySession(ctx context.Context, id string) (domain.RevokeMySessionResponse, error) {
	if ctx == nil {
		return domain.RevokeMySessionResponse{}, dictionary.ErrAuthGatewayContextRequired
	}
	if err := domain.ValidateSessionID(id); err != nil {
		return domain.RevokeMySessionResponse{}, err
	}
	result, err := g.client.RevokeMySession(ctx, &sessionv1.RevokeMySessionRequest{SessionId: id})
	if err != nil {
		return domain.RevokeMySessionResponse{}, err
	}
	if result == nil {
		return domain.RevokeMySessionResponse{}, dictionary.ErrSessionResultInvalid
	}
	return domain.RevokeMySessionResponse{Changed: result.Changed}, nil
}

func (g *MySessionsGateway) RevokeAllMySessions(ctx context.Context) (domain.RevokeAllMySessionsResponse, error) {
	if ctx == nil {
		return domain.RevokeAllMySessionsResponse{}, dictionary.ErrAuthGatewayContextRequired
	}
	result, err := g.client.RevokeAllMySessions(ctx)
	if err != nil {
		return domain.RevokeAllMySessionsResponse{}, err
	}
	if result == nil {
		return domain.RevokeAllMySessionsResponse{}, dictionary.ErrSessionResultInvalid
	}
	return domain.RevokeAllMySessionsResponse{Changed: result.Changed, LoginRequired: result.LoginRequired}, nil
}

func mySessionsPage(result *sessionv1.ListSessionsResponse) (domain.MySessionsPage, error) {
	if result == nil || result.Meta == nil {
		return domain.MySessionsPage{}, dictionary.ErrSessionResultInvalid
	}
	m := result.Meta
	if m.Page < 1 || m.Limit < 1 || m.Limit > 100 || m.Total < 0 || m.TotalPages < 0 {
		return domain.MySessionsPage{}, dictionary.ErrSessionResultInvalid
	}
	page := domain.MySessionsPage{
		Items: make([]domain.MySession, len(result.Items)),
		Total: m.Total, Page: m.Page, Limit: m.Limit, TotalPages: m.TotalPages,
		HasNext: m.HasNext, HasPrev: m.HasPrev, NextCursor: result.NextCursor,
	}
	for i, row := range result.Items {
		item, err := mySession(row)
		if err != nil {
			return domain.MySessionsPage{}, err
		}
		page.Items[i] = item
	}
	return page, nil
}

func mySession(row *sessionv1.Session) (domain.MySession, error) {
	if row == nil || domain.ValidateSessionID(row.Id) != nil || row.UserId == 0 || (row.Status != "ACTIVE" && row.Status != "REVOKED") || row.CreatedAt == nil || row.ExpiresAt == nil {
		return domain.MySession{}, dictionary.ErrSessionResultInvalid
	}
	for _, stamp := range []*timestamppb.Timestamp{row.CreatedAt, row.ExpiresAt, row.LastSeenAt, row.RevokedAt} {
		if stamp != nil && stamp.CheckValid() != nil {
			return domain.MySession{}, dictionary.ErrSessionResultInvalid
		}
	}
	return domain.MySession{
		ID: row.Id, UserID: row.UserId, Status: row.Status,
		CreatedAt:   row.CreatedAt.AsTime().UTC().Format(time.RFC3339Nano),
		ExpiresAt:   row.ExpiresAt.AsTime().UTC().Format(time.RFC3339Nano),
		LastSeenAt:  optionalSessionTime(row.LastSeenAt),
		RevokedAt:   optionalSessionTime(row.RevokedAt),
		DeviceLabel: row.DeviceLabel, UserAgent: row.UserAgent,
		IPAddress: row.IpAddress, IsCurrent: row.IsCurrent,
	}, nil
}

func optionalSessionTime(stamp *timestamppb.Timestamp) *string {
	if stamp == nil {
		return nil
	}
	value := stamp.AsTime().UTC().Format(time.RFC3339Nano)
	return &value
}
