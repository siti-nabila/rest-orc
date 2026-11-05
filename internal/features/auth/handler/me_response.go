package handler

import (
	"strconv"
	"time"

	requestauth "github.com/siti-nabila/rest-orc/internal/auth"
	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
)

func meResponse(principal requestauth.Principal) domain.MeResponse {
	data := domain.MeResponse{
		ID: principal.ID, Email: principal.Email, Fullname: principal.Fullname,
		Address: principal.Address, Phone: principal.Phone, CountryCode: principal.CountryCode,
		Status: principal.Status, Version: strconv.FormatUint(principal.Version, 10),
		MustChangePassword: principal.MustChangePassword,
		RoleCodes:          append([]int32{}, principal.RoleCodes...),
		RoleNames:          append([]string{}, principal.RoleNames...),
		Roles:              make([]domain.MeRole, 0, len(principal.Roles)),
		ApplicationAccess:  make([]domain.MeApplicationAccess, 0, len(principal.ApplicationAccess)),
		Permissions:        principal.Permissions,
	}
	if principal.CreatedAt != nil {
		createdAt := principal.CreatedAt.UTC().Format(time.RFC3339Nano)
		data.CreatedAt = &createdAt
	}
	if data.Permissions == nil {
		data.Permissions = map[string]bool{}
	}
	for _, role := range principal.Roles {
		data.Roles = append(data.Roles, domain.MeRole{
			RoleID: role.RoleID, Code: role.Code, Name: role.Name,
			Status: role.Status, RoleStatus: role.RoleStatus,
		})
	}
	for _, access := range principal.ApplicationAccess {
		data.ApplicationAccess = append(data.ApplicationAccess, domain.MeApplicationAccess{
			ApplicationID: access.ApplicationID, Code: access.Code, Name: access.Name,
			Status: access.Status, ApplicationStatus: access.ApplicationStatus,
		})
	}
	return data
}
