package domain

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	cd "github.com/siti-nabila/api-contracts/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

var applicationPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// ParseListQuery reads decoded values without mutating the caller or decoding cursors.
func ParseListQuery(input ListInput) (ListQuery, error) {
	q := ListQuery{Page: 1, Limit: 20}
	errs := cd.FieldErrors{}
	for _, key := range []string{"fields", "sort", "search", "search_fields", "search_mode", "created_from", "created_to", "updated_from", "updated_to", "code"} {
		if _, ok := input[key]; ok {
			errs.Add(key, dictionary.ErrUnsupportedRoleControl)
		}
	}
	for _, key := range []string{"page", "limit", "keyword", "application_code", "statuses", "last_id"} {
		values, ok := input[key]
		if !ok {
			continue
		}
		if len(values) != 1 {
			errs.Add(key, dictionary.ErrRepeatedRoleQuery)
			continue
		}
		value := values[0]
		switch key {
		case "page", "limit":
			max := int64(2147483647)
			if key == "limit" {
				max = 100
			}
			n, err := strconv.ParseInt(value, 10, 32)
			if !digits(value) || err != nil || n < 1 || n > max {
				errs.Add(key, dictionary.ErrInvalidRolePagination)
			} else if key == "page" {
				q.Page = int32(n)
			} else {
				q.Limit = int32(n)
			}
		case "application_code":
			if !validApplication(value) {
				errs.Add(key, dictionary.ErrInvalidRoleApplication)
			} else {
				q.ApplicationCode = value
			}
		case "keyword":
			if !validKeyword(value) {
				errs.Add(key, dictionary.ErrInvalidRoleKeyword)
			} else {
				q.Keyword = strings.Join(strings.Fields(value), " ")
			}
		case "last_id":
			if len(value) > 2048 {
				errs.Add(key, dictionary.ErrInvalidRoleCursor)
			} else {
				q.LastID = value
			}
		case "statuses":
			statuses := strings.Split(value, ",")
			valid := true
			for i, status := range statuses {
				status = strings.Trim(status, " ")
				statuses[i] = status
				if !validStatus(status) {
					valid = false
				}
			}
			if !valid {
				errs.Add(key, dictionary.ErrInvalidRoleStatuses)
			} else {
				slices.Sort(statuses)
				q.Statuses = slices.Compact(statuses)
			}
		}
	}
	if !errs.Empty() {
		return ListQuery{}, errs
	}
	return q, nil
}

// Validate checks a typed direct gateway request without normalizing it again.
func (q ListQuery) Validate() error {
	errs := cd.FieldErrors{}
	if q.Page < 1 {
		errs.Add("page", dictionary.ErrInvalidRolePagination)
	}
	if q.Limit < 1 || q.Limit > 100 {
		errs.Add("limit", dictionary.ErrInvalidRolePagination)
	}
	if q.ApplicationCode != "" && !validApplication(q.ApplicationCode) {
		errs.Add("application_code", dictionary.ErrInvalidRoleApplication)
	}
	if !validKeyword(q.Keyword) {
		errs.Add("keyword", dictionary.ErrInvalidRoleKeyword)
	}
	if len(q.LastID) > 2048 {
		errs.Add("last_id", dictionary.ErrInvalidRoleCursor)
	}
	for _, s := range q.Statuses {
		if !validStatus(s) {
			errs.Add("statuses", dictionary.ErrInvalidRoleStatuses)
			break
		}
	}
	if !errs.Empty() {
		return errs
	}
	return nil
}
func ParseRoleID(value string) (uint64, error) {
	id, err := strconv.ParseUint(value, 10, 31)
	if !digits(value) || err != nil || id == 0 {
		return 0, cd.FieldErrors{"role_id": {dictionary.ErrInvalidRoleID}}
	}
	return id, nil
}
func ValidateID(id uint64) bool { return id > 0 && id <= 2147483647 }
func digits(s string) bool {
	if s == "" {
		return false
	}
	for i := range len(s) {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
func validApplication(s string) bool {
	return len(s) >= 1 && len(s) <= 50 && applicationPattern.MatchString(s)
}
func validStatus(s string) bool { return s == "ACTIVE" || s == "DISABLED" }
func validKeyword(s string) bool {
	if len(s) > 200 {
		return false
	}
	for i := range len(s) {
		b := s[i]
		if !(b >= 0x20 && b <= 0x7e || b >= 0x09 && b <= 0x0d) {
			return false
		}
	}
	return true
}
