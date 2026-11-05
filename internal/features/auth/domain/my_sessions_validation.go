package domain

import (
	"regexp"
	"strconv"

	"github.com/siti-nabila/api-contracts/pkg/dictionary"
	pkgdictionary "github.com/siti-nabila/rest-orc/pkg/dictionary"
)

var sessionIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func ValidateSessionID(id string) error {
	if !sessionIDPattern.MatchString(id) {
		return dictionary.FieldErrors{"session_id": {pkgdictionary.ErrInvalidSessionID}}
	}
	return nil
}

// ParseMySessionsQuery accepts already URL-decoded values. Cursor contents are opaque.
func ParseMySessionsQuery(values map[string][]string) (MySessionsQuery, error) {
	query := MySessionsQuery{Page: 1, Limit: 20}
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
		return MySessionsQuery{}, errs
	}
	return query, nil
}

func (query MySessionsQuery) Validate() error {
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
