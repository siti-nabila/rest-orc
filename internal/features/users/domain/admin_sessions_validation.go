package domain

import (
	"regexp"
	"strconv"

	"github.com/siti-nabila/api-contracts/pkg/dictionary"
	pkgdictionary "github.com/siti-nabila/rest-orc/pkg/dictionary"
)

var adminSessionIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func ValidateAdminSessionID(id string) error {
	if !adminSessionIDPattern.MatchString(id) {
		return dictionary.FieldErrors{"session_id": {pkgdictionary.ErrInvalidSessionID}}
	}
	return nil
}

// ParseAdminSessionsQuery accepts already URL-decoded values. Cursor contents are opaque.
func ParseAdminSessionsQuery(values map[string][]string) (AdminSessionsQuery, error) {
	query := AdminSessionsQuery{Page: 1, Limit: 20}
	errs := dictionary.FieldErrors{}
	for _, field := range []string{"sort", "fields", "search", "keyword", "statuses", "application_code", "created_from", "created_to"} {
		if _, present := values[field]; present {
			errs.Add(field, pkgdictionary.ErrUnsupportedSessionControl)
		}
	}
	for _, field := range []string{"page", "limit", "last_id"} {
		list, present := values[field]
		if !present {
			continue
		}
		if len(list) != 1 {
			errs.Add(field, pkgdictionary.ErrInvalidSessionPagination)
			continue
		}
		if field == "last_id" {
			if len(list[0]) > 2048 {
				errs.Add(field, pkgdictionary.ErrInvalidSessionPagination)
			} else {
				query.LastID = list[0]
			}
			continue
		}
		value := list[0]
		valid := value != ""
		for i := 0; i < len(value); i++ {
			if value[i] < '0' || value[i] > '9' {
				valid = false
			}
		}
		number, err := strconv.ParseInt(value, 10, 32)
		if !valid || err != nil || number < 1 || (field == "limit" && number > 100) {
			errs.Add(field, pkgdictionary.ErrInvalidSessionPagination)
			continue
		}
		if field == "page" {
			query.Page = int32(number)
		} else {
			query.Limit = int32(number)
		}
	}
	if !errs.Empty() {
		return AdminSessionsQuery{}, errs
	}
	return query, nil
}

func (query AdminSessionsQuery) Validate() error {
	errs := dictionary.FieldErrors{}
	if query.Page < 1 {
		errs.Add("page", pkgdictionary.ErrInvalidSessionPagination)
	}
	if query.Limit < 1 || query.Limit > 100 {
		errs.Add("limit", pkgdictionary.ErrInvalidSessionPagination)
	}
	if len(query.LastID) > 2048 {
		errs.Add("last_id", pkgdictionary.ErrInvalidSessionPagination)
	}
	if !errs.Empty() {
		return errs
	}
	return nil
}

func ParseAdminSessionUserID(value string) (uint64, error) {
	number, err := strconv.ParseUint(value, 10, 31)
	if !asciiDigits(value) || err != nil || number == 0 {
		return 0, dictionary.FieldErrors{"user_id": {pkgdictionary.ErrInvalidSessionUserID}}
	}
	return number, nil
}

func ValidateAdminSessionUserID(id uint64) error {
	if id == 0 || id > 2147483647 {
		return dictionary.FieldErrors{"user_id": {pkgdictionary.ErrInvalidSessionUserID}}
	}
	return nil
}

func ValidateSessionIdempotencyKey(key string) error {
	valid := len(key) >= 1 && len(key) <= 200
	nonblank := false
	for i := 0; i < len(key); i++ {
		if key[i] < 0x20 || key[i] > 0x7e {
			valid = false
		}
		if key[i] != ' ' {
			nonblank = true
		}
	}
	if !valid || !nonblank {
		return dictionary.FieldErrors{"idempotency_key": {pkgdictionary.ErrInvalidSessionIdempotencyKey}}
	}
	return nil
}

func ParseAdminSessionVersion(value string) (uint64, error) {
	number, err := strconv.ParseUint(value, 10, 63)
	if !asciiDigits(value) || err != nil || number == 0 {
		return 0, dictionary.FieldErrors{"version": {pkgdictionary.ErrInvalidSessionVersion}}
	}
	return number, nil
}

func (request RevokeUserSessionRequest) Validate() error {
	if err := ValidateAdminSessionUserID(request.UserID); err != nil {
		return err
	}
	if err := ValidateAdminSessionID(request.SessionID); err != nil {
		return err
	}
	return ValidateSessionIdempotencyKey(request.IdempotencyKey)
}

func (request RevokeAllUserSessionsRequest) Validate() error {
	if err := ValidateAdminSessionUserID(request.UserID); err != nil {
		return err
	}
	if err := ValidateSessionIdempotencyKey(request.IdempotencyKey); err != nil {
		return err
	}
	if request.Version == 0 || request.Version > 9223372036854775807 {
		return dictionary.FieldErrors{"version": {pkgdictionary.ErrInvalidSessionVersion}}
	}
	return nil
}
