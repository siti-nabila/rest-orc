package authservice

import (
	"strconv"
	"time"

	rolev1 "github.com/siti-nabila/api-contracts/pb/role/v1"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/roles/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func safeRole(r *rolev1.RoleSummary) (domain.Role, error) {
	if r == nil || !domain.ValidateID(r.Id) || !domain.ValidateID(r.ApplicationId) || r.RoleCode < 1 || r.Version == 0 || r.Version > 9223372036854775807 || (r.Status != "ACTIVE" && r.Status != "DISABLED") {
		return domain.Role{}, common.ErrInternalServerError
	}
	created, err := safeTimestamp(r.CreatedAt)
	if err != nil {
		return domain.Role{}, err
	}
	updated, err := safeTimestamp(r.UpdatedAt)
	if err != nil {
		return domain.Role{}, err
	}
	out := domain.Role{ID: r.Id, Code: r.Code, Name: r.Name, ApplicationID: r.ApplicationId, ApplicationCode: r.ApplicationCode, Status: r.Status, IsSystem: r.IsSystem, Version: strconv.FormatUint(r.Version, 10), RoleCode: r.RoleCode, CreatedAt: created, UpdatedAt: updated, AffectedUserCount: r.AffectedUserCount}
	if r.Description != nil {
		value := *r.Description
		out.Description = &value
	}
	return out, nil
}
func safeTimestamp(t *timestamppb.Timestamp) (*string, error) {
	if t == nil {
		return nil, nil
	}
	if t.CheckValid() != nil {
		return nil, common.ErrInternalServerError
	}
	text := t.AsTime().UTC().Format(time.RFC3339Nano)
	return &text, nil
}
