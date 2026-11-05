package domain

import "context"

type AdminSessionActor struct {
	UserID             uint64
	MustChangePassword bool
	Permissions        map[string]bool
}

type AdminSessionsQuery struct {
	Page   int32
	Limit  int32
	LastID string
}

type AdminSession struct {
	ID          string  `json:"id"`
	UserID      uint64  `json:"user_id"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
	LastSeenAt  *string `json:"last_seen_at"`
	ExpiresAt   string  `json:"expires_at"`
	RevokedAt   *string `json:"revoked_at"`
	DeviceLabel string  `json:"device_label"`
	UserAgent   string  `json:"user_agent"`
	IPAddress   string  `json:"ip_address"`
	IsCurrent   bool    `json:"is_current"`
}

// Total follows the session service's int64 contract, independently of ListUsers.
type AdminSessionsPage struct {
	Items      []AdminSession `json:"items"`
	Total      int64          `json:"total"`
	Page       int32          `json:"page"`
	Limit      int32          `json:"limit"`
	TotalPages int32          `json:"total_pages"`
	HasNext    bool           `json:"has_next"`
	HasPrev    bool           `json:"has_prev"`
	NextCursor string         `json:"next_cursor"`
}

type RevokeUserSessionResponse struct {
	Changed bool `json:"changed"`
}
type RevokeAllUserSessionsResponse struct {
	Changed bool   `json:"changed"`
	Version string `json:"version"`
}

type ListUserSessionsGateway interface {
	ListUserSessions(context.Context, uint64, AdminSessionsQuery) (AdminSessionsPage, error)
}
type RevokeUserSessionGateway interface {
	RevokeUserSession(context.Context, RevokeUserSessionRequest) (RevokeUserSessionResponse, error)
}
type RevokeAllUserSessionsGateway interface {
	RevokeAllUserSessions(context.Context, RevokeAllUserSessionsRequest) (RevokeAllUserSessionsResponse, error)
}

const SessionReadPermission = "iam.session.read"
const SessionRevokePermission = "iam.session.revoke"

type RevokeUserSessionRequest struct {
	UserID         uint64
	SessionID      string
	IdempotencyKey string
}
type RevokeAllUserSessionsRequest struct {
	UserID         uint64
	IdempotencyKey string
	Version        uint64
}
