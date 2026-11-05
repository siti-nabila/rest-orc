package domain

import "context"

type SessionActor struct {
	UserID             uint64
	MustChangePassword bool
}

type MySessionsQuery struct {
	Page   int32
	Limit  int32
	LastID string
}

type MySession struct {
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
type MySessionsPage struct {
	Items      []MySession `json:"items"`
	Total      int64       `json:"total"`
	Page       int32       `json:"page"`
	Limit      int32       `json:"limit"`
	TotalPages int32       `json:"total_pages"`
	HasNext    bool        `json:"has_next"`
	HasPrev    bool        `json:"has_prev"`
	NextCursor string      `json:"next_cursor"`
}

type RevokeMySessionResponse struct {
	Changed bool `json:"changed"`
}
type RevokeAllMySessionsResponse struct {
	Changed       bool `json:"changed"`
	LoginRequired bool `json:"login_required"`
}

type ListMySessionsGateway interface {
	ListMySessions(context.Context, MySessionsQuery) (MySessionsPage, error)
}
type RevokeMySessionGateway interface {
	RevokeMySession(context.Context, string) (RevokeMySessionResponse, error)
}
type RevokeAllMySessionsGateway interface {
	RevokeAllMySessions(context.Context) (RevokeAllMySessionsResponse, error)
}
