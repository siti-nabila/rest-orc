package domain

import (
	"slices"
	"strconv"
	"strings"

	cd "github.com/siti-nabila/api-contracts/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

// ParseApplicationListQuery reads decoded values without mutating the caller or decoding cursors.
func ParseApplicationListQuery(input ApplicationListInput) (ApplicationListQuery, error) {
	q := ApplicationListQuery{Page: 1, Limit: 20}
	errs := cd.FieldErrors{}
	for _, key := range []string{"fields", "sort", "search", "search_fields", "search_mode", "created_from", "created_to", "updated_from", "updated_to", "application_code", "code"} {
		if _, ok := input[key]; ok {
			errs.Add(key, dictionary.ErrUnsupportedApplicationControl)
		}
	}
	for _, key := range []string{"page", "limit", "keyword", "statuses", "last_id"} {
		values, ok := input[key]
		if !ok {
			continue
		}
		if len(values) != 1 {
			errs.Add(key, dictionary.ErrRepeatedApplicationQuery)
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
				errs.Add(key, dictionary.ErrInvalidApplicationPagination)
			} else if key == "page" {
				q.Page = int32(n)
			} else {
				q.Limit = int32(n)
			}
		case "keyword":
			if !validKeyword(value) {
				errs.Add(key, dictionary.ErrInvalidApplicationKeyword)
			} else {
				q.Keyword = strings.Join(strings.Fields(value), " ")
			}
		case "last_id":
			if len(value) > 2048 {
				errs.Add(key, dictionary.ErrInvalidApplicationCursor)
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
				errs.Add(key, dictionary.ErrInvalidApplicationStatuses)
			} else {
				slices.Sort(statuses)
				q.Statuses = slices.Compact(statuses)
			}
		}
	}
	if !errs.Empty() {
		return ApplicationListQuery{}, errs
	}
	return q, nil
}

// Validate checks a typed direct gateway request without normalizing it again.
func (q ApplicationListQuery) Validate() error {
	errs := cd.FieldErrors{}
	if q.Page < 1 {
		errs.Add("page", dictionary.ErrInvalidApplicationPagination)
	}
	if q.Limit < 1 || q.Limit > 100 {
		errs.Add("limit", dictionary.ErrInvalidApplicationPagination)
	}
	if !validKeyword(q.Keyword) {
		errs.Add("keyword", dictionary.ErrInvalidApplicationKeyword)
	}
	if len(q.LastID) > 2048 {
		errs.Add("last_id", dictionary.ErrInvalidApplicationCursor)
	}
	for _, s := range q.Statuses {
		if !validStatus(s) {
			errs.Add("statuses", dictionary.ErrInvalidApplicationStatuses)
			break
		}
	}
	if !errs.Empty() {
		return errs
	}
	return nil
}
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
