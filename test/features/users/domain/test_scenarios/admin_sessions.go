package test_scenarios

import (
	"errors"
	"strings"
	"testing"

	contractdictionary "github.com/siti-nabila/api-contracts/pkg/dictionary"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func AdminSessionScenarios() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "target UUID key and version boundaries", Run: adminSessionBoundaries},
		{Name: "bounded pagination and unsupported controls", Run: adminPagination},
	}
}

func adminSessionBoundaries(t *testing.T) {
	for _, s := range []string{"1", "00023", "2147483647"} {
		if _, err := domain.ParseAdminSessionUserID(s); err != nil {
			t.Fatal(s, err)
		}
	}
	for _, s := range []string{"", "0", "2147483648", "-1", "+1", " 1", "1 ", "１", "0x1", strings.Repeat("9", 100)} {
		if _, err := domain.ParseAdminSessionUserID(s); err == nil {
			t.Fatal(s)
		}
	}
	for _, s := range []string{"1", "0001", "9007199254740993", "9223372036854775807"} {
		if _, err := domain.ParseAdminSessionVersion(s); err != nil {
			t.Fatal(s, err)
		}
	}
	for _, s := range []string{"", "0", "9223372036854775808", "18446744073709551616", "-1", "+1", " 1", "1 ", "１", "1.0"} {
		if _, err := domain.ParseAdminSessionVersion(s); err == nil {
			t.Fatal(s)
		}
	}
	for _, s := range []string{"00000000-0000-0000-0000-000000000000", "abcdefab-abcd-abcd-abcd-abcdefabcdef"} {
		if err := domain.ValidateAdminSessionID(s); err != nil {
			t.Fatal(s, err)
		}
	}
	for _, s := range []string{"", "ABCDEFAB-abcd-abcd-abcd-abcdefabcdef", " abcdefab-abcd-abcd-abcd-abcdefabcdef", "abcdefab-abcd-abcd-abcd-abcdefabcdef "} {
		if domain.ValidateAdminSessionID(s) == nil {
			t.Fatal(s)
		}
	}
	for _, s := range []string{"x", "a b", "a,b", strings.Repeat("x", 200), " x "} {
		if err := domain.ValidateSessionIdempotencyKey(s); err != nil {
			t.Fatal(err)
		}
	}
	for _, s := range []string{"", " ", strings.Repeat("x", 201), "é", "a\tb", "a\nb", "a\x00b", "a\x7fb"} {
		err := domain.ValidateSessionIdempotencyKey(s)
		var fields contractdictionary.FieldErrors
		if !errors.As(err, &fields) {
			t.Fatal("expected field error", err)
		}
		for _, lang := range []locale.Language{locale.English, locale.Indonesian, locale.Chinese} {
			if len(fields.Messages(lang)["idempotency_key"]) == 0 {
				t.Fatal("missing localized error")
			}
		}
	}
	for _, r := range []domain.RevokeAllUserSessionsRequest{{UserID: 0, IdempotencyKey: "k", Version: 1}, {UserID: 1, IdempotencyKey: "k", Version: 0}, {UserID: 1, IdempotencyKey: "k", Version: 1 << 63}, {UserID: 1, IdempotencyKey: "", Version: 1}} {
		if r.Validate() == nil {
			t.Fatal(r)
		}
	}
}

func adminPagination(t *testing.T) {
	q, err := domain.ParseAdminSessionsQuery(nil)
	if err != nil || q.Page != 1 || q.Limit != 20 {
		t.Fatal(q, err)
	}
	for _, field := range []string{"page", "limit", "last_id"} {
		if _, err := domain.ParseAdminSessionsQuery(map[string][]string{field: {"1", "1"}}); err == nil {
			t.Fatal(field)
		}
	}
	for _, field := range []string{"sort", "fields", "search", "keyword", "statuses", "application_code", "created_from", "created_to"} {
		if _, err := domain.ParseAdminSessionsQuery(map[string][]string{field: {""}}); err == nil {
			t.Fatal(field)
		}
	}
	for _, field := range []string{"page", "limit"} {
		for _, s := range []string{"", "0", "-1", "+1", "1.5", "１", " 1", "1 ", "2147483648"} {
			if _, err := domain.ParseAdminSessionsQuery(map[string][]string{field: {s}}); err == nil {
				t.Fatal(field, s)
			}
		}
	}
	if _, err := domain.ParseAdminSessionsQuery(map[string][]string{"limit": {"101"}}); err == nil {
		t.Fatal("limit")
	}
	q, err = domain.ParseAdminSessionsQuery(map[string][]string{"page": {"2147483647"}, "limit": {"00100"}, "last_id": {strings.Repeat("x", 2048)}, "user_id": {"99"}})
	if err != nil || q.Page != 2147483647 || q.Limit != 100 || len(q.LastID) != 2048 {
		t.Fatal(q, err)
	}
	if _, err = domain.ParseAdminSessionsQuery(map[string][]string{"last_id": {strings.Repeat("x", 2049)}}); err == nil {
		t.Fatal("cursor")
	}
}
