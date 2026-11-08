package domain

import (
	"slices"
	"strconv"
	"strings"

	cd "github.com/siti-nabila/api-contracts/pkg/dictionary"
	authdictionary "github.com/siti-nabila/api-contracts/pkg/dictionary/auth"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

func ParseUserRoleHeaders(keys, versions []string) (string, uint64, error) {
	if len(keys) != 1 {
		return "", 0, cd.FieldErrors{"idempotency_key": {dictionary.ErrInvalidSessionIdempotencyKey}}
	}
	if err := ValidateSessionIdempotencyKey(keys[0]); err != nil {
		return "", 0, err
	}
	if len(versions) != 1 {
		return "", 0, cd.FieldErrors{"version": {dictionary.ErrInvalidSessionVersion}}
	}
	version, err := ParseAdminSessionVersion(versions[0])
	if err != nil {
		return "", 0, err
	}
	return keys[0], version, nil
}

func (r UserRoleCommandRequest) NormalizeAndValidate() (UserRoleCommand, error) {
	id, err := ParseUserDetailID(r.UserID)
	if err != nil {
		return UserRoleCommand{}, err
	}
	out := UserRoleCommand{UserID: id, Version: r.Version, IdempotencyKey: r.IdempotencyKey}
	if err = validateUserCommandPreconditions(out.UserID, out.Version, out.IdempotencyKey); err != nil {
		return UserRoleCommand{}, err
	}
	errs := cd.FieldErrors{}
	roles := normalizeAssignmentRoles(r.RoleIDs, errs)
	out.Reason = normalizeAssignmentReason(r.Reason, errs)
	if !errs.Empty() {
		return UserRoleCommand{}, errs
	}
	out.RoleIDs = roles
	return out, nil
}

func normalizeAssignmentRoles(rawIDs []string, errs cd.FieldErrors) []uint64 {
	var roles []uint64
	if len(rawIDs) < 1 || len(rawIDs) > 20 {
		errs.Add("role_ids", dictionary.ErrInvalidAssignmentRoleIDs)
	} else {
		seen := map[uint64]bool{}
		for _, raw := range rawIDs {
			role, parseErr := strconv.ParseUint(raw, 10, 31)
			if !asciiDigits(raw) || parseErr != nil || role == 0 || seen[role] {
				errs.Add("role_ids", dictionary.ErrInvalidAssignmentRoleIDs)
				break
			}
			seen[role] = true
			roles = append(roles, role)
		}
	}
	slices.Sort(roles)
	return roles
}

func normalizeAssignmentReason(reason *string, errs cd.FieldErrors) string {
	var normalized string
	if reason == nil {
		errs.Add("reason", authdictionary.ErrRequired)
	} else {
		if !validAddress(*reason) {
			errs.Add("reason", dictionary.ErrInvalidStatusReasonCharacters)
		}
		normalized = strings.Trim(*reason, " \r\n")
		validateProfileLength(errs, "reason", normalized, 3, 500)
	}
	return normalized
}

func validateUserCommandPreconditions(userID, version uint64, key string) error {
	if err := ValidateUserDetailID(userID); err != nil {
		return err
	}
	if err := ValidateSessionIdempotencyKey(key); err != nil {
		return err
	}
	if version == 0 || version > 9223372036854775807 {
		return cd.FieldErrors{"version": {dictionary.ErrInvalidSessionVersion}}
	}
	return nil
}

// Validate also protects direct gateway consumers; it never repairs caller input.
func (r UserRoleCommand) Validate() error {
	if err := validateUserCommandPreconditions(r.UserID, r.Version, r.IdempotencyKey); err != nil {
		return err
	}
	errs := cd.FieldErrors{}
	validateAssignmentRoles(r.RoleIDs, errs)
	if !validAddress(r.Reason) {
		errs.Add("reason", dictionary.ErrInvalidStatusReasonCharacters)
	}
	validateProfileLength(errs, "reason", r.Reason, 3, 500)
	if !errs.Empty() {
		return errs
	}
	return nil
}

func validateAssignmentRoles(roles []uint64, errs cd.FieldErrors) {
	if len(roles) < 1 || len(roles) > 20 {
		errs.Add("role_ids", dictionary.ErrInvalidAssignmentRoleIDs)
	}
	for i, id := range roles {
		if id == 0 || id > 2147483647 || i > 0 && id <= roles[i-1] {
			errs.Add("role_ids", dictionary.ErrInvalidAssignmentRoleIDs)
			break
		}
	}
}
