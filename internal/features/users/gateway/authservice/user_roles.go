package authservice

import (
	"context"
	"slices"
	"strconv"
	"time"

	paginatorv1 "github.com/siti-nabila/api-contracts/pb/paginator/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type UserRoleManagementClient interface {
	ListUserRoles(context.Context, *userv1.ListUserRolesRequest) (*userv1.ListUserRolesResponse, error)
	GrantUserRoles(context.Context, *userv1.GrantUserRolesRequest) (*userv1.GrantUserRolesResponse, error)
	RevokeUserRoles(context.Context, *userv1.RevokeUserRolesRequest) (*userv1.RevokeUserRolesResponse, error)
}
type UserRoleManagementGateway struct{ client UserRoleManagementClient }

func NewUserRoleManagement(c UserRoleManagementClient) (*UserRoleManagementGateway, error) {
	if c == nil {
		return nil, dictionary.ErrUsersGatewayClientRequired
	}
	return &UserRoleManagementGateway{client: c}, nil
}
func (g *UserRoleManagementGateway) ListUserRoles(ctx context.Context, target uint64, q domain.UserRoleListQuery) (domain.ListUserRolesResponse, error) {
	if ctx == nil {
		return domain.ListUserRolesResponse{}, dictionary.ErrUsersGatewayContextRequired
	}
	if err := domain.ValidateUserDetailID(target); err != nil {
		return domain.ListUserRolesResponse{}, err
	}
	if err := q.Validate(); err != nil {
		return domain.ListUserRolesResponse{}, err
	}
	page := &paginatorv1.PageQuery{Page: q.Page, Limit: q.Limit, LastId: q.LastID, Sort: []*paginatorv1.Sort{{Field: "id", Desc: false}}}
	if q.Keyword != "" {
		page.Search = &paginatorv1.Search{Fields: []string{"keyword"}, Keyword: q.Keyword, Mode: paginatorv1.SearchMode_SEARCH_MODE_FULL_TEXT_TRIGRAM}
	}
	request := &userv1.ListUserRolesRequest{UserId: target, Query: page, Filter: &userv1.UserRoleAssignmentFilter{ApplicationCode: q.ApplicationCode, Statuses: slices.Clone(q.Statuses)}}
	r, err := g.client.ListUserRoles(adminSessionContext(ctx, ""), request)
	if err != nil {
		return domain.ListUserRolesResponse{}, err
	}
	if r == nil || r.Meta == nil || !validUserRoleResultTarget(r.UserId, target, r.Version) {
		return domain.ListUserRolesResponse{}, common.ErrInternalServerError
	}
	m := r.Meta
	if m.Page < 1 || m.Limit < 1 || m.Limit > 100 || m.Total < 0 || m.TotalPages < 0 {
		return domain.ListUserRolesResponse{}, common.ErrInternalServerError
	}
	items := make([]domain.UserRoleAssignment, len(r.Items))
	for i, item := range r.Items {
		mapped, e := safeUserRoleAssignment(item)
		if e != nil {
			return domain.ListUserRolesResponse{}, e
		}
		items[i] = mapped
	}
	return domain.ListUserRolesResponse{
		Items:      items,
		Total:      m.Total,
		Page:       m.Page,
		Limit:      m.Limit,
		TotalPages: m.TotalPages,
		HasNext:    m.HasNext,
		HasPrev:    m.HasPrev,
		NextCursor: r.NextCursor,
		UserID:     r.UserId,
		Version:    strconv.FormatUint(r.Version, 10),
	}, nil
}
func (g *UserRoleManagementGateway) GrantUserRoles(ctx context.Context, r domain.UserRoleCommand) (domain.UserRoleCommandResponse, error) {
	if ctx == nil {
		return domain.UserRoleCommandResponse{}, dictionary.ErrUsersGatewayContextRequired
	}
	if err := r.Validate(); err != nil {
		return domain.UserRoleCommandResponse{}, err
	}
	version := r.Version
	out, err := g.client.GrantUserRoles(adminSessionContext(ctx, r.IdempotencyKey), &userv1.GrantUserRolesRequest{UserId: r.UserID, RoleIds: slices.Clone(r.RoleIDs), Reason: r.Reason, ExpectedUserVersion: &version})
	if err != nil {
		return domain.UserRoleCommandResponse{}, err
	}
	if out == nil {
		return domain.UserRoleCommandResponse{}, common.ErrInternalServerError
	}
	return safeUserRoleCommand(r, out.UserId, out.Version, out.ChangedRoleIds, out.Changed)
}
func (g *UserRoleManagementGateway) RevokeUserRoles(ctx context.Context, r domain.UserRoleCommand) (domain.UserRoleCommandResponse, error) {
	if ctx == nil {
		return domain.UserRoleCommandResponse{}, dictionary.ErrUsersGatewayContextRequired
	}
	if err := r.Validate(); err != nil {
		return domain.UserRoleCommandResponse{}, err
	}
	version := r.Version
	out, err := g.client.RevokeUserRoles(adminSessionContext(ctx, r.IdempotencyKey), &userv1.RevokeUserRolesRequest{UserId: r.UserID, RoleIds: slices.Clone(r.RoleIDs), Reason: r.Reason, ExpectedUserVersion: &version})
	if err != nil {
		return domain.UserRoleCommandResponse{}, err
	}
	if out == nil {
		return domain.UserRoleCommandResponse{}, common.ErrInternalServerError
	}
	return safeUserRoleCommand(r, out.UserId, out.Version, out.ChangedRoleIds, out.Changed)
}
func validUserRoleResultTarget(id, target, version uint64) bool {
	return id == target && domain.ValidateUserDetailID(id) == nil && version > 0 && version <= 9223372036854775807
}
func safeUserRoleCommand(r domain.UserRoleCommand, id, version uint64, ids []uint64, changed bool) (domain.UserRoleCommandResponse, error) {
	if !validUserRoleResultTarget(id, r.UserID, version) || changed != (len(ids) > 0) {
		return domain.UserRoleCommandResponse{}, common.ErrInternalServerError
	}
	for i, role := range ids {
		if role == 0 || role > 2147483647 || i > 0 && role <= ids[i-1] || !slices.Contains(r.RoleIDs, role) {
			return domain.UserRoleCommandResponse{}, common.ErrInternalServerError
		}
	}
	owned := append([]uint64{}, ids...)
	return domain.UserRoleCommandResponse{UserID: id, ChangedRoleIDs: owned, Version: strconv.FormatUint(version, 10), Changed: changed}, nil
}
func safeUserRoleAssignment(r *userv1.UserRoleAssignmentItem) (domain.UserRoleAssignment, error) {
	if r == nil || domain.ValidateUserDetailID(r.Id) != nil || domain.ValidateUserDetailID(r.RoleId) != nil || domain.ValidateUserDetailID(r.ApplicationId) != nil || r.RoleCode < 1 || !assignmentState(r.Status) || !roleDefinitionState(r.RoleStatus) || !roleDefinitionState(r.ApplicationStatus) {
		return domain.UserRoleAssignment{}, common.ErrInternalServerError
	}
	if r.ApplicationAccessStatus != nil && !assignmentState(*r.ApplicationAccessStatus) || r.GrantedBy != nil && domain.ValidateUserDetailID(*r.GrantedBy) != nil || r.RevokedBy != nil && domain.ValidateUserDetailID(*r.RevokedBy) != nil {
		return domain.UserRoleAssignment{}, common.ErrInternalServerError
	}
	granted, err := userRoleTimestamp(r.GrantedAt)
	if err != nil {
		return domain.UserRoleAssignment{}, err
	}
	revoked, err := userRoleTimestamp(r.RevokedAt)
	if err != nil {
		return domain.UserRoleAssignment{}, err
	}
	return domain.UserRoleAssignment{
		ID:                      r.Id,
		RoleID:                  r.RoleId,
		Code:                    r.Code,
		RoleCode:                r.RoleCode,
		Name:                    r.Name,
		Status:                  r.Status,
		RoleStatus:              r.RoleStatus,
		ApplicationID:           r.ApplicationId,
		ApplicationCode:         r.ApplicationCode,
		ApplicationStatus:       r.ApplicationStatus,
		IsSystem:                r.IsSystem,
		ApplicationAccessStatus: cloneUserRoleValue(r.ApplicationAccessStatus),
		GrantedBy:               cloneUserRoleValue(r.GrantedBy),
		GrantedAt:               granted,
		RevokedBy:               cloneUserRoleValue(r.RevokedBy),
		RevokedAt:               revoked,
		RevokeReason:            cloneUserRoleValue(r.RevokeReason),
	}, nil
}
func assignmentState(s string) bool     { return s == "ACTIVE" || s == "REVOKED" }
func roleDefinitionState(s string) bool { return s == "ACTIVE" || s == "DISABLED" }
func cloneUserRoleValue[T any](p *T) *T {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}
func userRoleTimestamp(t *timestamppb.Timestamp) (*string, error) {
	if t == nil {
		return nil, nil
	}
	if t.CheckValid() != nil {
		return nil, common.ErrInternalServerError
	}
	s := t.AsTime().UTC().Format(time.RFC3339Nano)
	return &s, nil
}
