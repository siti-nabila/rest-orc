package test_scenarios

import (
	"encoding/json"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func MySessionsScenarios() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "bounded pagination and canonical UUID", Run: mySessionsValidation},
		{Name: "session output keeps int64 totals nulls false and safe fields", Run: mySessionsJSON},
	}
}
func mySessionsValidation(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want domain.MySessionsQuery
	}{
		{"", domain.MySessionsQuery{Page: 1, Limit: 20}},
		{"page=2147483647&limit=100&last_id=%252F%2B%20", domain.MySessionsQuery{Page: 2147483647, Limit: 100, LastID: "%2F+ "}},
		{"page=1&limit=1&last_id=", domain.MySessionsQuery{Page: 1, Limit: 1}},
		{"last_id=" + strings.Repeat("x", 2048), domain.MySessionsQuery{Page: 1, Limit: 20, LastID: strings.Repeat("x", 2048)}},
		{"user_id=99&namespace=forged&other=ignored", domain.MySessionsQuery{Page: 1, Limit: 20}},
	} {
		values, err := url.ParseQuery(tc.raw)
		if err != nil {
			t.Fatal(err)
		}
		got, err := domain.ParseMySessionsQuery(values)
		if err != nil || got != tc.want {
			t.Fatalf("%q: %+v %v", tc.raw, got, err)
		}
	}
	invalid := []string{"page=", "limit=", "page=0", "page=-1", "page=1.1", "page=%EF%BC%91", "page=%201", "page=2147483648", "page=999999999999999999999999999", "limit=101", "limit=0", "page=1&page=1", "limit=20&limit=20", "last_id=&last_id=", "last_id=" + strings.Repeat("x", 2049)}
	for _, key := range []string{"sort", "fields", "search", "keyword", "statuses", "application_code", "created_from", "created_to"} {
		invalid = append(invalid, key+"=")
	}
	for _, raw := range invalid {
		values, _ := url.ParseQuery(raw)
		if _, err := domain.ParseMySessionsQuery(values); err == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
	for _, id := range []string{"00000000-0000-0000-0000-000000000000", "abcdefab-abcd-abcd-abcd-abcdefabcdef"} {
		if err := domain.ValidateSessionID(id); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{"", "ABCDEFAB-abcd-abcd-abcd-abcdefabcdef", "abcdefab-abcd-abcd-abcd-abcdefabcdef ", "00000000000000000000000000000000"} {
		if domain.ValidateSessionID(id) == nil {
			t.Fatal("accepted invalid UUID")
		}
	}
}
func mySessionsJSON(t *testing.T) {
	raw, err := json.Marshal(domain.MySessionsPage{Items: []domain.MySession{{}}, Total: 2147483648})
	if err != nil {
		t.Fatal(err)
	}
	var page map[string]json.RawMessage
	if err = json.Unmarshal(raw, &page); err != nil {
		t.Fatal(err)
	}
	if string(page["total"]) != "2147483648" || len(page) != 8 {
		t.Fatal(string(raw))
	}
	var items []map[string]any
	if err = json.Unmarshal(page["items"], &items); err != nil {
		t.Fatal(err)
	}
	want := []string{"id", "user_id", "status", "created_at", "last_seen_at", "expires_at", "revoked_at", "device_label", "user_agent", "ip_address", "is_current"}
	if len(items) != 1 || len(items[0]) != len(want) {
		t.Fatal(string(raw))
	}
	for _, key := range want {
		if _, ok := items[0][key]; !ok {
			t.Fatal(key)
		}
	}
	if items[0]["last_seen_at"] != nil || items[0]["revoked_at"] != nil || items[0]["is_current"] != false {
		t.Fatal(string(raw))
	}
	for _, tc := range []struct {
		v    any
		want string
	}{{domain.RevokeMySessionResponse{}, `{"changed":false}`}, {domain.RevokeAllMySessionsResponse{}, `{"changed":false,"login_required":false}`}} {
		b, _ := json.Marshal(tc.v)
		if !reflect.DeepEqual(string(b), tc.want) {
			t.Fatal(string(b))
		}
	}
}
