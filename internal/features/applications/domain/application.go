package domain

import "context"

const ReadPermission = "iam.permission.read"

type ApplicationActor struct {
	UserID             uint64
	MustChangePassword bool
	Permissions        map[string]bool
}
type ApplicationListInput map[string][]string
type ApplicationListQuery struct {
	Page, Limit     int32
	Keyword, LastID string
	Statuses        []string
}

// ApplicationPage preserves the backend int64 total for both representations.
type ApplicationPage[T any] struct {
	Items      []T    `json:"items"`
	Total      int64  `json:"total"`
	Page       int32  `json:"page"`
	Limit      int32  `json:"limit"`
	TotalPages int32  `json:"total_pages"`
	HasNext    bool   `json:"has_next"`
	HasPrev    bool   `json:"has_prev"`
	NextCursor string `json:"next_cursor"`
}
type Application struct {
	ID       uint64 `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	IsSystem bool   `json:"is_system"`
}
type ApplicationOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}
type ApplicationListGateway interface {
	ListApplications(context.Context, ApplicationListQuery) (ApplicationPage[Application], error)
}
