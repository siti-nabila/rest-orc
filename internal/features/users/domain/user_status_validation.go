package domain

import (
	"strings"

	"github.com/siti-nabila/api-contracts/pkg/dictionary"
	authdictionary "github.com/siti-nabila/api-contracts/pkg/dictionary/auth"
	pkgdictionary "github.com/siti-nabila/rest-orc/pkg/dictionary"
)

// ParseUserStatusHeaders receives all parsed values, including duplicate headers.
func ParseUserStatusHeaders(keys, versions []string) (string, uint64, error) {
	if len(keys) != 1 {
		return "", 0, dictionary.FieldErrors{"idempotency_key": {pkgdictionary.ErrInvalidSessionIdempotencyKey}}
	}
	if err := ValidateSessionIdempotencyKey(keys[0]); err != nil {
		return "", 0, err
	}
	if len(versions) != 1 {
		return "", 0, dictionary.FieldErrors{"version": {pkgdictionary.ErrInvalidSessionVersion}}
	}
	version, err := ParseAdminSessionVersion(versions[0])
	if err != nil {
		return "", 0, err
	}
	return keys[0], version, nil
}

func ValidUserStatus(status string) bool {
	switch status {
	case "PENDING", "ACTIVE", "SUSPENDED", "DEACTIVATED":
		return true
	}
	return false
}

func (request ChangeUserStatusRequest) NormalizeAndValidate() (ChangeUserStatusRequest, error) {
	if err := ValidateUserDetailID(request.UserID); err != nil {
		return ChangeUserStatusRequest{}, err
	}
	if err := ValidateSessionIdempotencyKey(request.IdempotencyKey); err != nil {
		return ChangeUserStatusRequest{}, err
	}
	if request.Version == 0 || request.Version > 9223372036854775807 {
		return ChangeUserStatusRequest{}, dictionary.FieldErrors{"version": {pkgdictionary.ErrInvalidSessionVersion}}
	}
	result := request
	errs := dictionary.FieldErrors{}
	if request.Status == nil {
		errs.Add("status", authdictionary.ErrRequired)
	} else {
		value := *request.Status
		result.Status = &value
		if !ValidUserStatus(value) {
			errs.Add("status", pkgdictionary.ErrInvalidUserStatus)
		}
	}
	if request.Reason == nil {
		errs.Add("reason", authdictionary.ErrRequired)
	} else {
		raw := *request.Reason
		if !validAddress(raw) {
			errs.Add("reason", pkgdictionary.ErrInvalidStatusReasonCharacters)
		}
		value := strings.Trim(raw, " \r\n")
		result.Reason = &value
		validateProfileLength(errs, "reason", value, 3, 500)
	}
	if !errs.Empty() {
		return ChangeUserStatusRequest{}, errs
	}
	return result, nil
}
