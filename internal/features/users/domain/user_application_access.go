package domain

import (
	"context"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

const (
	UserApplicationReadPermission   = "iam.user.read"
	UserApplicationGrantPermission  = "iam.user.application.grant"
	UserApplicationRevokePermission = "iam.user.application.revoke"
)

type UserApplicationAccessActor struct {
	UserID             uint64
	MustChangePassword bool
	Permissions        map[string]bool
}

// Access history and role history accept the same bounded public query controls.
type UserApplicationAccessListInput map[string][]string
type UserApplicationAccessListQuery UserRoleListQuery

func ParseUserApplicationAccessListQuery(input UserApplicationAccessListInput) (UserApplicationAccessListQuery, error) {
	q, err := parseUserRelationshipListQuery(UserRoleListInput(input), dictionary.ErrUnsupportedAccessControl)
	return UserApplicationAccessListQuery(q), err
}

func (q UserApplicationAccessListQuery) Validate() error {
	return UserRoleListQuery(q).Validate()
}

type ListUserApplicationAccessRequest struct {
	UserID string
	Input  UserApplicationAccessListInput
}

type UserApplicationAccess struct {
	ApplicationID     uint64  `json:"application_id"`
	ApplicationCode   string  `json:"application_code"`
	Name              string  `json:"name"`
	ApplicationStatus string  `json:"application_status"`
	IsSystem          bool    `json:"is_system"`
	Status            string  `json:"status"`
	GrantedBy         *uint64 `json:"granted_by"`
	GrantedAt         string  `json:"granted_at"`
	RevokedBy         *uint64 `json:"revoked_by"`
	RevokedAt         *string `json:"revoked_at"`
	RevokeReason      *string `json:"revoke_reason"`
}

type ListUserApplicationAccessResponse struct {
	Items      []UserApplicationAccess `json:"items"`
	Total      int64                   `json:"total"`
	Page       int32                   `json:"page"`
	Limit      int32                   `json:"limit"`
	TotalPages int32                   `json:"total_pages"`
	HasNext    bool                    `json:"has_next"`
	HasPrev    bool                    `json:"has_prev"`
	NextCursor string                  `json:"next_cursor"`
	UserID     uint64                  `json:"user_id"`
	Version    string                  `json:"version"`
}

type GrantApplicationAccessBody struct {
	RoleIDs []string `json:"role_ids"`
	Reason  *string  `json:"reason"`
}

type RevokeApplicationAccessBody struct {
	Reason *string `json:"reason"`
}

type ApplicationAccessCommandRequest struct {
	UserID          string
	ApplicationCode string
	Version         uint64
	IdempotencyKey  string
}

type GrantApplicationAccessRequest struct {
	ApplicationAccessCommandRequest
	GrantApplicationAccessBody
}

type RevokeApplicationAccessRequest struct {
	ApplicationAccessCommandRequest
	RevokeApplicationAccessBody
}

type ApplicationAccessCommand struct {
	UserID          uint64
	ApplicationCode string
	Reason          string
	Version         uint64
	IdempotencyKey  string
}

type GrantApplicationAccessCommand struct {
	ApplicationAccessCommand
	RoleIDs []uint64
}

type ApplicationAccessCommandResponse struct {
	UserID         uint64   `json:"user_id"`
	ApplicationID  uint64   `json:"application_id"`
	AccessStatus   *string  `json:"access_status"`
	ChangedRoleIDs []uint64 `json:"changed_role_ids"`
	Version        string   `json:"version"`
	Changed        bool     `json:"changed"`
}

type ListUserApplicationAccessGateway interface {
	ListUserApplicationAccess(context.Context, uint64, UserApplicationAccessListQuery) (ListUserApplicationAccessResponse, error)
}
type GrantApplicationAccessGateway interface {
	GrantApplicationAccess(context.Context, GrantApplicationAccessCommand) (ApplicationAccessCommandResponse, error)
}
type RevokeApplicationAccessGateway interface {
	RevokeApplicationAccess(context.Context, ApplicationAccessCommand) (ApplicationAccessCommandResponse, error)
}
