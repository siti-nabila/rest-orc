package domain

import "context"

const (
	UserRoleReadPermission   = "iam.user.read"
	UserRoleGrantPermission  = "iam.user.role.grant"
	UserRoleRevokePermission = "iam.user.role.revoke"
)

type UserRoleActor struct {
	UserID             uint64
	MustChangePassword bool
	Permissions        map[string]bool
}

type UserRoleListInput map[string][]string

type UserRoleListQuery struct {
	Page, Limit                      int32
	ApplicationCode, Keyword, LastID string
	Statuses                         []string
}

type ListUserRolesRequest struct {
	UserID string
	Input  UserRoleListInput
}

type UserRoleAssignment struct {
	ID                      uint64  `json:"id"`
	RoleID                  uint64  `json:"role_id"`
	Code                    string  `json:"code"`
	RoleCode                int32   `json:"role_code"`
	Name                    string  `json:"name"`
	Status                  string  `json:"status"`
	RoleStatus              string  `json:"role_status"`
	ApplicationID           uint64  `json:"application_id"`
	ApplicationCode         string  `json:"application_code"`
	ApplicationStatus       string  `json:"application_status"`
	IsSystem                bool    `json:"is_system"`
	ApplicationAccessStatus *string `json:"application_access_status"`
	GrantedBy               *uint64 `json:"granted_by"`
	GrantedAt               *string `json:"granted_at"`
	RevokedBy               *uint64 `json:"revoked_by"`
	RevokedAt               *string `json:"revoked_at"`
	RevokeReason            *string `json:"revoke_reason"`
}

type ListUserRolesResponse struct {
	Items      []UserRoleAssignment `json:"items"`
	Total      int64                `json:"total"`
	Page       int32                `json:"page"`
	Limit      int32                `json:"limit"`
	TotalPages int32                `json:"total_pages"`
	HasNext    bool                 `json:"has_next"`
	HasPrev    bool                 `json:"has_prev"`
	NextCursor string               `json:"next_cursor"`
	UserID     uint64               `json:"user_id"`
	Version    string               `json:"version"`
}

type UserRoleCommandBody struct {
	RoleIDs []string `json:"role_ids"`
	Reason  *string  `json:"reason"`
}

type UserRoleCommandRequest struct {
	UserID string
	UserRoleCommandBody
	Version        uint64
	IdempotencyKey string
}

// UserRoleCommand is a validated, owned batch for one downstream operation.
type UserRoleCommand struct {
	UserID         uint64
	RoleIDs        []uint64
	Reason         string
	Version        uint64
	IdempotencyKey string
}

type UserRoleCommandResponse struct {
	UserID         uint64   `json:"user_id"`
	ChangedRoleIDs []uint64 `json:"changed_role_ids"`
	Version        string   `json:"version"`
	Changed        bool     `json:"changed"`
}

type ListUserRolesGateway interface {
	ListUserRoles(context.Context, uint64, UserRoleListQuery) (ListUserRolesResponse, error)
}
type GrantUserRolesGateway interface {
	GrantUserRoles(context.Context, UserRoleCommand) (UserRoleCommandResponse, error)
}
type RevokeUserRolesGateway interface {
	RevokeUserRoles(context.Context, UserRoleCommand) (UserRoleCommandResponse, error)
}
