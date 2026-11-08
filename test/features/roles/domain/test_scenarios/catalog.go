package test_scenarios

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	cd "github.com/siti-nabila/api-contracts/pkg/dictionary"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	"github.com/siti-nabila/rest-orc/internal/features/roles/domain"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func All() []testutils.Scenario {
	return []testutils.Scenario{{Name: "role DTO precision nulls and independent identity fields", Run: schemas}, {Name: "role query accepted bounds canonical filters and isolation", Run: validQueries}, {Name: "role invalid query fields locales and targets", Run: invalidQueries}}
}
func schemas(t *testing.T) {
	desc, created, updated := " legacy\té ", "created", "updated"
	r := domain.Role{ID: 10, RoleCode: 3, Code: "clm.legal_reviewer", Version: "9007199254740993", Description: &desc, CreatedAt: &created, UpdatedAt: &updated}
	clone := r.Clone()
	*clone.Description = "changed"
	*clone.CreatedAt = "changed"
	*clone.UpdatedAt = "changed"
	if desc != " legacy\té " || created != "created" || updated != "updated" {
		t.Fatal("alias")
	}
	r.Description = nil
	r.CreatedAt = nil
	r.UpdatedAt = nil
	cases := []struct {
		value  any
		count  int
		checks map[string]string
	}{
		{r, 13, map[string]string{"id": "10", "role_code": "3", "version": `"9007199254740993"`, "description": "null", "created_at": "null", "updated_at": "null", "is_system": "false", "affected_user_count": "0"}},
		{domain.Permission{ID: 7, ApplicationID: 5, Code: "clm.read"}, 5, map[string]string{"id": "7", "application_id": "5"}},
		{domain.Option{Value: "10", RoleCode: "3", Code: "clm.legal_reviewer"}, 4, map[string]string{"value": `"10"`, "role_code": `"3"`}},
		{domain.Page[domain.Role]{Items: []domain.Role{}, Total: 9223372036854775807}, 8, map[string]string{"total": "9223372036854775807", "items": "[]", "has_next": "false"}},
	}
	for _, tc := range cases {
		raw, err := json.Marshal(tc.value)
		var fields map[string]json.RawMessage
		if err != nil || json.Unmarshal(raw, &fields) != nil || len(fields) != tc.count {
			t.Fatal(string(raw), err)
		}
		for key, want := range tc.checks {
			if string(fields[key]) != want {
				t.Fatal(key, string(raw))
			}
		}
	}
}
func validQueries(t *testing.T) {
	q, err := domain.ParseListQuery(nil)
	if err != nil || q.Page != 1 || q.Limit != 20 || q.Keyword != "" || len(q.Statuses) != 0 {
		t.Fatal(q, err)
	}
	for _, row := range []struct{ key, value string }{
		{"page", "2147483647"}, {"page", "0001"}, {"limit", "00100"}, {"application_code", "a"}, {"application_code", strings.Repeat("a", 50)}, {"application_code", "clm_2"}, {"keyword", strings.Repeat("x", 200)}, {"keyword", " \t\n \v\f\r"}, {"last_id", strings.Repeat("é", 1024)}, {"last_id", " opaque+/=% "}, {"last_id", ""}, {"statuses", " DISABLED,ACTIVE,ACTIVE "}, {"keyword", "  legal\t\n reviewer "},
	} {
		t.Run(row.key+"/"+row.value, func(t *testing.T) {
			input := domain.ListInput{row.key: {row.value}, "user_id": {"forged"}, "namespace": {"forged"}}
			before, _ := json.Marshal(input)
			got, err := domain.ParseListQuery(input)
			if err != nil || got.Validate() != nil {
				t.Fatal(got, err)
			}
			if row.key == "keyword" && strings.TrimSpace(row.value) == "" && got.Keyword != "" {
				t.Fatal(got)
			}
			if row.key == "keyword" && strings.Contains(row.value, "legal") && got.Keyword != "legal reviewer" {
				t.Fatal(got)
			}
			if row.key == "last_id" && got.LastID != row.value {
				t.Fatal("cursor changed")
			}
			if row.key == "statuses" && !reflect.DeepEqual(got.Statuses, []string{"ACTIVE", "DISABLED"}) {
				t.Fatal(got)
			}
			if len(got.Statuses) > 0 {
				got.Statuses[0] = "changed"
			}
			after, _ := json.Marshal(input)
			if string(before) != string(after) {
				t.Fatal("input mutated")
			}
		})
	}
	for _, id := range []string{"1", "00010", "2147483647"} {
		got, err := domain.ParseRoleID(id)
		if err != nil || !domain.ValidateID(got) {
			t.Fatal(got, err)
		}
	}
}
func invalidQueries(t *testing.T) {
	rows := map[string][]string{
		"page":             {"", "0", "-1", "+1", " 1", "1 ", "１", "1.0", "2147483648", "999999999999999999999"},
		"limit":            {"", "0", "-1", "+1", " 1", "101", "2147483648"},
		"application_code": {"", "Clm", " clm", "clm ", "1clm", "clm.app", "clmé", strings.Repeat("a", 51)},
		"keyword":          {strings.Repeat("x", 201), "\x00", "\x01", "\x08", "\x0e", "\x1f", "\x7f", "é", "\u00a0x"},
		"statuses":         {"", " ", "ACTIVE,", ",ACTIVE", "ACTIVE,,DISABLED", "active", "ACTIVE\t", " ACTIVE\u00a0", "REVOKED"},
		"last_id":          {strings.Repeat("x", 2049)},
	}
	for field, values := range rows {
		for _, value := range values {
			t.Run(field+"/"+value, func(t *testing.T) {
				if _, err := domain.ParseListQuery(domain.ListInput{field: {value}}); err == nil {
					t.Fatal("accepted")
				}
			})
		}
	}
	for _, field := range []string{"page", "limit", "application_code", "statuses", "keyword", "last_id"} {
		for _, values := range [][]string{nil, {"x", "x"}} {
			if _, err := domain.ParseListQuery(domain.ListInput{field: values}); err == nil {
				t.Fatal("duplicate", field)
			}
		}
	}
	for _, field := range []string{"fields", "sort", "search", "search_fields", "search_mode", "created_from", "created_to", "updated_from", "updated_to", "code"} {
		if _, err := domain.ParseListQuery(domain.ListInput{field: {""}}); err == nil {
			t.Fatal("unsupported", field)
		}
	}
	for _, id := range []string{"", "0", "-1", "+1", " 1", "1 ", "1.0", "１", "2147483648", "options", "clm.admin"} {
		if _, err := domain.ParseRoleID(id); err == nil {
			t.Fatal("bad id", id)
		}
	}
	for _, id := range []uint64{0, 2147483648} {
		if domain.ValidateID(id) {
			t.Fatal("bad id")
		}
	}
	_, err := domain.ParseListQuery(domain.ListInput{"page": {"bad-secret"}, "limit": {"bad-secret"}, "application_code": {"bad-secret"}, "statuses": {"bad-secret"}, "keyword": {"bad-secret\x00"}})
	var fields cd.FieldErrors
	if !errors.As(err, &fields) || len(fields) != 5 {
		t.Fatal(err)
	}
	for _, lang := range []locale.Language{locale.English, locale.Indonesian, locale.Chinese} {
		messages := fields.Messages(lang)
		b, _ := json.Marshal(messages)
		if len(messages) != 5 || strings.Contains(string(b), "bad-secret") {
			t.Fatal(string(b))
		}
	}
	for _, q := range []domain.ListQuery{{Page: 0, Limit: 20}, {Page: 1, Limit: 0}, {Page: 1, Limit: 101}, {Page: 1, Limit: 20, ApplicationCode: "Bad"}, {Page: 1, Limit: 20, Statuses: []string{"bad"}}, {Page: 1, Limit: 20, Keyword: "\x00"}, {Page: 1, Limit: 20, LastID: strings.Repeat("x", 2049)}} {
		if q.Validate() == nil {
			t.Fatal("typed invalid", q)
		}
	}
}
