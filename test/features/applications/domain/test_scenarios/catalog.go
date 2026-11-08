package test_scenarios

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	cd "github.com/siti-nabila/api-contracts/pkg/dictionary"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	"github.com/siti-nabila/rest-orc/internal/features/applications/domain"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func All() []testutils.Scenario {
	return []testutils.Scenario{{Name: "application exact safe JSON views and int64 totals", Run: schemas}, {Name: "application accepted query boundaries canonicalization and ownership", Run: validQueries}, {Name: "application invalid fields duplicates unsupported controls and locales", Run: invalidQueries}}
}
func schemas(t *testing.T) {
	for _, tc := range []struct {
		v      any
		n      int
		checks map[string]string
	}{
		{domain.Application{ID: 5, Code: "clm", Name: " Legacy\té ", Status: "DISABLED"}, 5, map[string]string{"id": "5", "code": `"clm"`, "is_system": "false"}},
		{domain.ApplicationOption{Value: "clm", Label: "clm - Contract Lifecycle Management"}, 2, map[string]string{"value": `"clm"`}},
		{domain.ApplicationPage[domain.Application]{Items: []domain.Application{}, Total: 9223372036854775807}, 8, map[string]string{"total": "9223372036854775807", "items": "[]", "has_next": "false", "has_prev": "false"}},
	} {
		b, e := json.Marshal(tc.v)
		var m map[string]json.RawMessage
		if e != nil || json.Unmarshal(b, &m) != nil || len(m) != tc.n {
			t.Fatal(string(b), e)
		}
		for k, w := range tc.checks {
			if string(m[k]) != w {
				t.Fatal(k, string(b))
			}
		}
	}
}
func validQueries(t *testing.T) {
	q, e := domain.ParseApplicationListQuery(nil)
	if e != nil || q.Page != 1 || q.Limit != 20 || len(q.Statuses) != 0 {
		t.Fatal(q, e)
	}
	for _, row := range []struct{ k, v string }{{"page", "1"}, {"page", "2147483647"}, {"page", "0001"}, {"limit", "00100"}, {"limit", "1"}, {"keyword", strings.Repeat("x", 200)}, {"keyword", " \t\n\v\f\r"}, {"last_id", strings.Repeat("é", 1024)}, {"last_id", " opaque+/=% "}, {"last_id", ""}, {"statuses", " DISABLED,ACTIVE,ACTIVE "}, {"keyword", "  app\t\n name "}} {
		t.Run(row.k+"/"+row.v, func(t *testing.T) {
			input := domain.ApplicationListInput{row.k: {row.v}, "namespace": {"forged"}, "user_id": {"99"}}
			before, _ := json.Marshal(input)
			got, e := domain.ParseApplicationListQuery(input)
			if e != nil || got.Validate() != nil {
				t.Fatal(got, e)
			}
			if row.k == "keyword" && strings.TrimSpace(row.v) == "" && got.Keyword != "" {
				t.Fatal(got)
			}
			if row.k == "keyword" && strings.Contains(row.v, "app") && got.Keyword != "app name" {
				t.Fatal(got)
			}
			if row.k == "last_id" && got.LastID != row.v {
				t.Fatal("cursor changed")
			}
			if row.k == "statuses" && !reflect.DeepEqual(got.Statuses, []string{"ACTIVE", "DISABLED"}) {
				t.Fatal(got)
			}
			if len(got.Statuses) > 0 {
				got.Statuses[0] = "mutated"
			}
			after, _ := json.Marshal(input)
			if string(before) != string(after) {
				t.Fatal("input mutated")
			}
		})
	}
}
func invalidQueries(t *testing.T) {
	bad := func(input domain.ApplicationListInput, keys ...string) {
		t.Helper()
		_, e := domain.ParseApplicationListQuery(input)
		var f cd.FieldErrors
		if !errors.As(e, &f) || len(f) != len(keys) {
			t.Fatal(input, f, e)
		}
		for _, lang := range []locale.Language{locale.English, locale.Indonesian, locale.Chinese} {
			m := f.Messages(lang)
			for _, k := range keys {
				if len(m[k]) == 0 || m[k][0] == "" || strings.Contains(m[k][0], "private-sentinel") || lang != locale.English && m[k][0] == f.Messages(locale.English)[k][0] {
					t.Fatal(lang, k, m)
				}
			}
		}
	}
	for _, key := range []string{"page", "limit", "last_id", "keyword", "statuses"} {
		for _, values := range [][]string{nil, {}, {"1", "1"}} {
			bad(domain.ApplicationListInput{key: values}, key)
		}
	}
	for _, key := range []string{"page", "limit"} {
		for _, raw := range []string{"", "0", "-1", "+1", " 1", "1 ", "1.2", "１", "2147483648", "99999999999999999999"} {
			bad(domain.ApplicationListInput{key: {raw}}, key)
		}
	}
	bad(domain.ApplicationListInput{"limit": {"101"}}, "limit")
	for _, raw := range []string{"", "ACTIVE,", ",ACTIVE", "active", "REVOKED", "ACTIVE,,DISABLED", "\tACTIVE"} {
		bad(domain.ApplicationListInput{"statuses": {raw}}, "statuses")
	}
	for _, raw := range []string{"é", "x\x00", "x\x7f", "x\x08", strings.Repeat("x", 201)} {
		bad(domain.ApplicationListInput{"keyword": {raw}}, "keyword")
	}
	bad(domain.ApplicationListInput{"last_id": {strings.Repeat("x", 2049)}}, "last_id")
	for _, key := range []string{"fields", "sort", "search", "search_fields", "search_mode", "application_code", "code", "created_from", "created_to", "updated_from", "updated_to"} {
		bad(domain.ApplicationListInput{key: {""}}, key)
	}
	bad(domain.ApplicationListInput{"page": {"0"}, "limit": {"101"}, "statuses": {"REVOKED"}, "keyword": {"private-sentinel\x00"}}, "page", "limit", "statuses", "keyword")
	for _, mutate := range []func(*domain.ApplicationListQuery){func(q *domain.ApplicationListQuery) { q.Page = 0 }, func(q *domain.ApplicationListQuery) { q.Limit = 0 }, func(q *domain.ApplicationListQuery) { q.Limit = 101 }, func(q *domain.ApplicationListQuery) { q.Keyword = "é" }, func(q *domain.ApplicationListQuery) { q.LastID = strings.Repeat("x", 2049) }, func(q *domain.ApplicationListQuery) { q.Statuses = []string{"REVOKED"} }} {
		q, _ := domain.ParseApplicationListQuery(nil)
		mutate(&q)
		if q.Validate() == nil {
			t.Fatal(q)
		}
	}
}
