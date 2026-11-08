package domain

import "context"

const ReadPermission = "iam.role.read"

type Actor struct {
	UserID             uint64
	MustChangePassword bool
	Permissions        map[string]bool
}

type ListInput map[string][]string

type ListQuery struct {
	Page, Limit                      int32
	Keyword, ApplicationCode, LastID string
	Statuses                         []string
}

// Page keeps the catalog's int64 total without changing legacy user pagination.
type Page[T any] struct {
	Items      []T    `json:"items"`
	Total      int64  `json:"total"`
	Page       int32  `json:"page"`
	Limit      int32  `json:"limit"`
	TotalPages int32  `json:"total_pages"`
	HasNext    bool   `json:"has_next"`
	HasPrev    bool   `json:"has_prev"`
	NextCursor string `json:"next_cursor"`
}

type Role struct {
	ID                uint64  `json:"id"`
	Code              string  `json:"code"`
	Name              string  `json:"name"`
	Description       *string `json:"description"`
	ApplicationID     uint64  `json:"application_id"`
	ApplicationCode   string  `json:"application_code"`
	Status            string  `json:"status"`
	IsSystem          bool    `json:"is_system"`
	Version           string  `json:"version"`
	RoleCode          int32   `json:"role_code"`
	CreatedAt         *string `json:"created_at"`
	UpdatedAt         *string `json:"updated_at"`
	AffectedUserCount uint64  `json:"affected_user_count"`
}

// Clone isolates nullable response values; response text is never write-normalized.
func (r Role) Clone() Role {
	r.Description = copyString(r.Description)
	r.CreatedAt = copyString(r.CreatedAt)
	r.UpdatedAt = copyString(r.UpdatedAt)
	return r
}
func copyString(p *string) *string {
	if p == nil {
		return nil
	}
	value := *p
	return &value
}

type Permission struct {
	ID              uint64 `json:"id"`
	ApplicationID   uint64 `json:"application_id"`
	ApplicationCode string `json:"application_code"`
	Code            string `json:"code"`
	Description     string `json:"description"`
}
type Detail struct {
	Role        Role         `json:"role"`
	Permissions []Permission `json:"permissions"`
}
type Option struct {
	Value    string `json:"value"`
	Label    string `json:"label"`
	Code     string `json:"code"`
	RoleCode string `json:"role_code"`
}
type ListGateway interface {
	List(context.Context, ListQuery) (Page[Role], error)
}
type GetGateway interface {
	Get(context.Context, uint64) (Detail, error)
}
