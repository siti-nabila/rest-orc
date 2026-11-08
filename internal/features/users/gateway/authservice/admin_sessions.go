package authservice

import (
	"context"
	"strconv"
	"time"

	paginatorv1 "github.com/siti-nabila/api-contracts/pb/paginator/v1"
	sessionv1 "github.com/siti-nabila/api-contracts/pb/session/v1"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AdminSessionClient interface {
	ListUserSessions(context.Context, *sessionv1.ListUserSessionsRequest) (*sessionv1.ListSessionsResponse, error)
	RevokeUserSession(context.Context, *sessionv1.RevokeUserSessionRequest) (*sessionv1.RevokeSessionResponse, error)
	RevokeAllUserSessions(context.Context, *sessionv1.RevokeAllUserSessionsRequest) (*sessionv1.RevokeAllUserSessionsResponse, error)
}
type AdminSessionsGateway struct{ client AdminSessionClient }

func NewAdminSessions(client AdminSessionClient) (*AdminSessionsGateway, error) {
	if client == nil {
		return nil, dictionary.ErrUsersGatewayClientRequired
	}
	return &AdminSessionsGateway{client: client}, nil
}

// Preconditions belong to this operation, never to inherited outgoing metadata.
func adminSessionContext(ctx context.Context, key string) context.Context {
	md, _ := metadata.FromOutgoingContext(ctx)
	md = md.Copy()
	md.Delete("idempotency-key")
	md.Delete("version")
	if key != "" {
		md.Set("idempotency-key", key)
	}
	return metadata.NewOutgoingContext(ctx, md)
}

func (g *AdminSessionsGateway) ListUserSessions(ctx context.Context, target uint64, q domain.AdminSessionsQuery) (domain.AdminSessionsPage, error) {
	if ctx == nil {
		return domain.AdminSessionsPage{}, dictionary.ErrUsersGatewayContextRequired
	}
	if err := domain.ValidateAdminSessionUserID(target); err != nil {
		return domain.AdminSessionsPage{}, err
	}
	if err := q.Validate(); err != nil {
		return domain.AdminSessionsPage{}, err
	}
	result, err := g.client.ListUserSessions(adminSessionContext(ctx, ""), &sessionv1.ListUserSessionsRequest{
		UserId: target, Query: &paginatorv1.PageQuery{Page: q.Page, Limit: q.Limit, LastId: q.LastID, Sort: []*paginatorv1.Sort{{Field: "created_at", Desc: true}, {Field: "id", Desc: true}}},
	})
	if err != nil {
		return domain.AdminSessionsPage{}, err
	}
	return adminSessionsPage(result)
}

func (g *AdminSessionsGateway) RevokeUserSession(ctx context.Context, request domain.RevokeUserSessionRequest) (domain.RevokeUserSessionResponse, error) {
	if ctx == nil {
		return domain.RevokeUserSessionResponse{}, dictionary.ErrUsersGatewayContextRequired
	}
	if err := request.Validate(); err != nil {
		return domain.RevokeUserSessionResponse{}, err
	}
	result, err := g.client.RevokeUserSession(adminSessionContext(ctx, request.IdempotencyKey), &sessionv1.RevokeUserSessionRequest{UserId: request.UserID, SessionId: request.SessionID})
	if err != nil {
		return domain.RevokeUserSessionResponse{}, err
	}
	if result == nil {
		return domain.RevokeUserSessionResponse{}, dictionary.ErrSessionResultInvalid
	}
	return domain.RevokeUserSessionResponse{Changed: result.Changed}, nil
}

func (g *AdminSessionsGateway) RevokeAllUserSessions(ctx context.Context, request domain.RevokeAllUserSessionsRequest) (domain.RevokeAllUserSessionsResponse, error) {
	if ctx == nil {
		return domain.RevokeAllUserSessionsResponse{}, dictionary.ErrUsersGatewayContextRequired
	}
	if err := request.Validate(); err != nil {
		return domain.RevokeAllUserSessionsResponse{}, err
	}
	version := request.Version
	result, err := g.client.RevokeAllUserSessions(adminSessionContext(ctx, request.IdempotencyKey), &sessionv1.RevokeAllUserSessionsRequest{UserId: request.UserID, ExpectedUserVersion: &version})
	if err != nil {
		return domain.RevokeAllUserSessionsResponse{}, err
	}
	if result == nil || result.Version == 0 || result.Version > 9223372036854775807 {
		return domain.RevokeAllUserSessionsResponse{}, dictionary.ErrSessionResultInvalid
	}
	return domain.RevokeAllUserSessionsResponse{Changed: result.Changed, Version: strconv.FormatUint(result.Version, 10)}, nil
}

func adminSessionsPage(result *sessionv1.ListSessionsResponse) (domain.AdminSessionsPage, error) {
	if result == nil || result.Meta == nil {
		return domain.AdminSessionsPage{}, dictionary.ErrSessionResultInvalid
	}
	m := result.Meta
	if m.Page < 1 || m.Limit < 1 || m.Limit > 100 || m.Total < 0 || m.TotalPages < 0 {
		return domain.AdminSessionsPage{}, dictionary.ErrSessionResultInvalid
	}
	page := domain.AdminSessionsPage{
		Items: make([]domain.AdminSession, len(result.Items)),
		Total: m.Total, Page: m.Page, Limit: m.Limit, TotalPages: m.TotalPages,
		HasNext: m.HasNext, HasPrev: m.HasPrev, NextCursor: result.NextCursor,
	}
	for i, row := range result.Items {
		item, err := adminSession(row)
		if err != nil {
			return domain.AdminSessionsPage{}, err
		}
		page.Items[i] = item
	}
	return page, nil
}

func adminSession(row *sessionv1.Session) (domain.AdminSession, error) {
	if row == nil || domain.ValidateAdminSessionID(row.Id) != nil || (row.UserId == 0 || row.UserId > 2147483647) || (row.Status != "ACTIVE" && row.Status != "REVOKED") || row.CreatedAt == nil || row.ExpiresAt == nil {
		return domain.AdminSession{}, dictionary.ErrSessionResultInvalid
	}
	for _, stamp := range []*timestamppb.Timestamp{row.CreatedAt, row.ExpiresAt, row.LastSeenAt, row.RevokedAt} {
		if stamp != nil && stamp.CheckValid() != nil {
			return domain.AdminSession{}, dictionary.ErrSessionResultInvalid
		}
	}
	return domain.AdminSession{
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
