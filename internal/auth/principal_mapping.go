package auth

import (
	"errors"
	"maps"
	"slices"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

func principalFromMe(user *userv1.UserData) (Principal, error) {
	if user == nil {
		return Principal{}, dictionary.ErrGRPCAuthMeResponseRequired
	}
	principal := Principal{
		ID: user.GetId(), Email: user.GetEmail(), Fullname: user.GetFullname(),
		Address: user.GetAddress(), Phone: user.GetPhone(), CountryCode: user.GetCountryCode(),
		Status: user.GetStatus(), Version: user.GetVersion(),
		MustChangePassword: user.GetMustChangePassword(),
	}
	if timestamp := user.GetCreatedAt(); timestamp != nil {
		if err := timestamp.CheckValid(); err != nil {
			return Principal{}, errors.Join(common.ErrInternalServerError, err)
		}
		createdAt := timestamp.AsTime()
		principal.CreatedAt = &createdAt
	}
	if principal.MustChangePassword {
		return principal, nil
	}

	principal.RoleCodes = slices.Clone(user.GetRoleCodes())
	principal.RoleNames = slices.Clone(user.GetRoleNames())
	principal.Permissions = maps.Clone(user.GetPermissions())
	for _, role := range user.GetRoles() {
		if role == nil {
			continue
		}
		principal.Roles = append(principal.Roles, RoleAssignment{
			RoleID: role.GetRoleId(), Code: role.GetCode(), Name: role.GetName(),
			Status: role.GetStatus(), RoleStatus: role.GetRoleStatus(),
		})
	}
	for _, access := range user.GetApplicationAccess() {
		if access == nil {
			continue
		}
		principal.ApplicationAccess = append(principal.ApplicationAccess, ApplicationAccess{
			ApplicationID: access.GetApplicationId(), Code: access.GetCode(), Name: access.GetName(),
			Status: access.GetStatus(), ApplicationStatus: access.GetApplicationStatus(),
		})
	}
	return principal, nil
}
