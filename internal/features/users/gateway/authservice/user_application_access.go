package authservice

import (
	"context"
	"slices"
	"strconv"

	applicationv1 "github.com/siti-nabila/api-contracts/pb/application/v1"
	paginatorv1 "github.com/siti-nabila/api-contracts/pb/paginator/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

type UserApplicationAccessClient interface {
	ListUserApplicationAccess(context.Context, *userv1.ListUserApplicationAccessRequest) (*userv1.ListUserApplicationAccessResponse, error)
	GrantApplicationAccess(context.Context, *applicationv1.GrantApplicationAccessRequest) (*applicationv1.GrantApplicationAccessResponse, error)
	RevokeApplicationAccess(context.Context, *applicationv1.RevokeApplicationAccessRequest) (*applicationv1.RevokeApplicationAccessResponse, error)
}

type UserApplicationAccessGateway struct{ client UserApplicationAccessClient }

func NewUserApplicationAccess(client UserApplicationAccessClient) (*UserApplicationAccessGateway, error) {
	if client == nil {
		return nil, dictionary.ErrUsersGatewayClientRequired
	}
	return &UserApplicationAccessGateway{client: client}, nil
}

func (g *UserApplicationAccessGateway) ListUserApplicationAccess(ctx context.Context, target uint64, q domain.UserApplicationAccessListQuery) (domain.ListUserApplicationAccessResponse, error) {
	if ctx == nil {
		return domain.ListUserApplicationAccessResponse{}, dictionary.ErrUsersGatewayContextRequired
	}
	if err := domain.ValidateUserDetailID(target); err != nil {
		return domain.ListUserApplicationAccessResponse{}, err
	}
	if err := q.Validate(); err != nil {
		return domain.ListUserApplicationAccessResponse{}, err
	}
	page := &paginatorv1.PageQuery{Page: q.Page, Limit: q.Limit, LastId: q.LastID, Sort: []*paginatorv1.Sort{{Field: "application_id"}}}
	if q.Keyword != "" {
		page.Search = &paginatorv1.Search{Fields: []string{"application_code", "name"}, Keyword: q.Keyword, Mode: paginatorv1.SearchMode_SEARCH_MODE_CONTAINS}
	}
	request := &userv1.ListUserApplicationAccessRequest{
		UserId: target,
		Query:  page,
		Filter: &userv1.UserApplicationAccessFilter{
			ApplicationCode: q.ApplicationCode,
			Statuses:        slices.Clone(q.Statuses),
		},
	}
	r, err := g.client.ListUserApplicationAccess(adminSessionContext(ctx, ""), request)
	if err != nil {
		return domain.ListUserApplicationAccessResponse{}, err
	}
	if r == nil || r.Meta == nil || !validUserRoleResultTarget(r.UserId, target, r.Version) {
		return domain.ListUserApplicationAccessResponse{}, common.ErrInternalServerError
	}
	m := r.Meta
	if m.Page < 1 || m.Limit < 1 || m.Limit > 100 || m.Total < 0 || m.TotalPages < 0 {
		return domain.ListUserApplicationAccessResponse{}, common.ErrInternalServerError
	}
	items := make([]domain.UserApplicationAccess, len(r.Items))
	for i, item := range r.Items {
		mapped, e := safeUserApplicationAccess(item)
		if e != nil {
			return domain.ListUserApplicationAccessResponse{}, e
		}
		items[i] = mapped
	}
	return domain.ListUserApplicationAccessResponse{
		Items: items, Total: m.Total, Page: m.Page, Limit: m.Limit,
		TotalPages: m.TotalPages, HasNext: m.HasNext, HasPrev: m.HasPrev,
		NextCursor: r.NextCursor, UserID: r.UserId, Version: strconv.FormatUint(r.Version, 10),
	}, nil
}

func safeUserApplicationAccess(r *userv1.UserApplicationAccessItem) (domain.UserApplicationAccess, error) {
	if r == nil || domain.ValidateUserDetailID(r.ApplicationId) != nil || !assignmentState(r.Status) || !roleDefinitionState(r.ApplicationStatus) || r.GrantedAt == nil {
		return domain.UserApplicationAccess{}, common.ErrInternalServerError
	}
	if r.GrantedBy != nil && domain.ValidateUserDetailID(*r.GrantedBy) != nil || r.RevokedBy != nil && domain.ValidateUserDetailID(*r.RevokedBy) != nil {
		return domain.UserApplicationAccess{}, common.ErrInternalServerError
	}
	granted, err := userRoleTimestamp(r.GrantedAt)
	if err != nil {
		return domain.UserApplicationAccess{}, err
	}
	revoked, err := userRoleTimestamp(r.RevokedAt)
	if err != nil {
		return domain.UserApplicationAccess{}, err
	}
	return domain.UserApplicationAccess{
		ApplicationID:     r.ApplicationId,
		ApplicationCode:   r.ApplicationCode,
		Name:              r.Name,
		ApplicationStatus: r.ApplicationStatus,
		IsSystem:          r.IsSystem,
		Status:            r.Status,
		GrantedBy:         cloneUserRoleValue(r.GrantedBy),
		GrantedAt:         *granted,
		RevokedBy:         cloneUserRoleValue(r.RevokedBy),
		RevokedAt:         revoked,
		RevokeReason:      cloneUserRoleValue(r.RevokeReason),
	}, nil
}

func (g *UserApplicationAccessGateway) GrantApplicationAccess(ctx context.Context, r domain.GrantApplicationAccessCommand) (domain.ApplicationAccessCommandResponse, error) {
	if ctx == nil {
		return domain.ApplicationAccessCommandResponse{}, dictionary.ErrUsersGatewayContextRequired
	}
	if err := r.Validate(); err != nil {
		return domain.ApplicationAccessCommandResponse{}, err
	}
	version := r.Version
	request := &applicationv1.GrantApplicationAccessRequest{
		UserId: r.UserID, ApplicationCode: r.ApplicationCode,
		RoleIds: slices.Clone(r.RoleIDs), Reason: r.Reason, ExpectedUserVersion: &version,
	}
	out, err := g.client.GrantApplicationAccess(adminSessionContext(ctx, r.IdempotencyKey), request)
	if err != nil {
		return domain.ApplicationAccessCommandResponse{}, err
	}
	if out == nil || out.AccessStatus == nil || *out.AccessStatus != "ACTIVE" {
		return domain.ApplicationAccessCommandResponse{}, common.ErrInternalServerError
	}
	for _, id := range out.ChangedRoleIds {
		if !slices.Contains(r.RoleIDs, id) {
			return domain.ApplicationAccessCommandResponse{}, common.ErrInternalServerError
		}
	}
	return safeApplicationAccessCommand(r.UserID, out.UserId, out.ApplicationId, out.Version, out.AccessStatus, out.ChangedRoleIds, out.Changed)
}

func (g *UserApplicationAccessGateway) RevokeApplicationAccess(ctx context.Context, r domain.ApplicationAccessCommand) (domain.ApplicationAccessCommandResponse, error) {
	if ctx == nil {
		return domain.ApplicationAccessCommandResponse{}, dictionary.ErrUsersGatewayContextRequired
	}
	if err := r.Validate(); err != nil {
		return domain.ApplicationAccessCommandResponse{}, err
	}
	version := r.Version
	request := &applicationv1.RevokeApplicationAccessRequest{
		UserId: r.UserID, ApplicationCode: r.ApplicationCode,
		Reason: r.Reason, ExpectedUserVersion: &version,
	}
	out, err := g.client.RevokeApplicationAccess(adminSessionContext(ctx, r.IdempotencyKey), request)
	if err != nil {
		return domain.ApplicationAccessCommandResponse{}, err
	}
	if out == nil || out.AccessStatus != nil && *out.AccessStatus != "REVOKED" {
		return domain.ApplicationAccessCommandResponse{}, common.ErrInternalServerError
	}
	return safeApplicationAccessCommand(r.UserID, out.UserId, out.ApplicationId, out.Version, out.AccessStatus, out.ChangedRoleIds, out.Changed)
}

// An access-only change can have no changed roles. Revoke can affect any number of roles.
func safeApplicationAccessCommand(target, id, applicationID, version uint64, status *string, ids []uint64, changed bool) (domain.ApplicationAccessCommandResponse, error) {
	if !validUserRoleResultTarget(id, target, version) || domain.ValidateUserDetailID(applicationID) != nil || !changed && len(ids) > 0 {
		return domain.ApplicationAccessCommandResponse{}, common.ErrInternalServerError
	}
	for i, role := range ids {
		if domain.ValidateUserDetailID(role) != nil || i > 0 && role <= ids[i-1] {
			return domain.ApplicationAccessCommandResponse{}, common.ErrInternalServerError
		}
	}
	return domain.ApplicationAccessCommandResponse{
		UserID: id, ApplicationID: applicationID, AccessStatus: cloneUserRoleValue(status),
		ChangedRoleIDs: append([]uint64{}, ids...), Version: strconv.FormatUint(version, 10), Changed: changed,
	}, nil
}
