package auth

import (
	"context"
	"slices"
)

const RoleAdmin RoleCode = 2

type (
	RoleCode            int32
	principalContextKey struct{}
)

type Principal struct {
	ID        uint64
	Email     string
	Fullname  string
	RoleCodes []int32
	RoleNames []string
}

func (r RoleCode) Int32() int32 {
	return int32(r)
}

func (principal Principal) HasRole(roleCode int32) bool {
	return slices.Contains(principal.RoleCodes, roleCode)
}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	if ctx == nil {
		return Principal{}, false
	}
	principal, exists := ctx.Value(principalContextKey{}).(Principal)
	return principal, exists
}
