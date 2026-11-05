package auth

import (
	"context"
	"maps"
	"slices"
	"time"
)

const RoleAdmin RoleCode = 2

type (
	RoleCode            int32
	principalContextKey struct{}
)

type Principal struct {
	ID                 uint64
	Email              string
	Fullname           string
	Address            string
	Phone              string
	CountryCode        string
	CreatedAt          *time.Time
	Status             string
	Version            uint64
	MustChangePassword bool
	RoleCodes          []int32
	RoleNames          []string
	Roles              []RoleAssignment
	ApplicationAccess  []ApplicationAccess
	Permissions        map[string]bool
}

type RoleAssignment struct {
	RoleID     uint64
	Code       string
	Name       string
	Status     string
	RoleStatus string
}

type ApplicationAccess struct {
	ApplicationID     uint64
	Code              string
	Name              string
	Status            string
	ApplicationStatus string
}

func (r RoleCode) Int32() int32 {
	return int32(r)
}

func (principal Principal) HasRole(roleCode int32) bool {
	return slices.Contains(principal.RoleCodes, roleCode)
}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal.clone())
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	if ctx == nil {
		return Principal{}, false
	}
	principal, exists := ctx.Value(principalContextKey{}).(Principal)
	return principal.clone(), exists
}

func (principal Principal) clone() Principal {
	principal.RoleCodes = slices.Clone(principal.RoleCodes)
	principal.RoleNames = slices.Clone(principal.RoleNames)
	principal.Roles = slices.Clone(principal.Roles)
	principal.ApplicationAccess = slices.Clone(principal.ApplicationAccess)
	principal.Permissions = maps.Clone(principal.Permissions)
	if principal.CreatedAt != nil {
		createdAt := *principal.CreatedAt
		principal.CreatedAt = &createdAt
	}
	return principal
}
