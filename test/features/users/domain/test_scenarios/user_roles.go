package test_scenarios

import (
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"

	cd "github.com/siti-nabila/api-contracts/pkg/dictionary"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func UserRoleScenarios() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "assignment JSON shapes precision and nulls", Run: userRoleShapes},
		{Name: "assignment pagination filters search and cursor", Run: userRoleQueries},
		{Name: "assignment IDs reason headers and ownership", Run: userRoleCommands},
	}
}
func userRoleShapes(t *testing.T) {
	r := domain.ListUserRolesResponse{Items: []domain.UserRoleAssignment{{ID: 1, RoleID: 10, RoleCode: 3}}, Total: 1 << 32, UserID: 23, Version: "9007199254740993"}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	if err = json.Unmarshal(b, &m); err != nil || len(m) != 10 {
		t.Fatal(string(b), err)
	}
	var items []map[string]json.RawMessage
	_ = json.Unmarshal(m["items"], &items)
	if len(items) != 1 || len(items[0]) != 17 || string(m["version"]) != `"9007199254740993"` || string(m["total"]) != "4294967296" {
		t.Fatal(string(b))
	}
	for _, k := range []string{"application_access_status", "granted_by", "granted_at", "revoked_by", "revoked_at", "revoke_reason"} {
		if string(items[0][k]) != "null" {
			t.Fatal(k)
		}
	}
	if string(items[0]["is_system"]) != "false" {
		t.Fatal(string(b))
	}
	r.Items = []domain.UserRoleAssignment{}
	b, _ = json.Marshal(r)
	_ = json.Unmarshal(b, &m)
	if string(m["items"]) != "[]" {
		t.Fatal(string(b))
	}
	b, _ = json.Marshal(domain.UserRoleCommandResponse{UserID: 23, ChangedRoleIDs: []uint64{}, Version: "9223372036854775807"})
	// Unmarshal into a fresh map when checking the exact command shape.
	m = nil
	_ = json.Unmarshal(b, &m)
	if len(m) != 4 || string(m["changed_role_ids"]) != "[]" || string(m["changed"]) != "false" {
		t.Fatal(string(b))
	}
}
func userRoleQueryError(t *testing.T, input domain.UserRoleListInput, fields ...string) {
	t.Helper()
	_, err := domain.ParseUserRoleListQuery(input)
	var f cd.FieldErrors
	if !errors.As(err, &f) {
		t.Fatalf("%v: %v", input, err)
	}
	for _, lang := range []locale.Language{locale.English, locale.Indonesian, locale.Chinese} {
		m := f.Messages(lang)
		for _, key := range fields {
			if len(m[key]) == 0 || m[key][0] == "" {
				t.Fatal(lang, key, m)
			}
		}
	}
}
func userRoleQueries(t *testing.T) {
	q, err := domain.ParseUserRoleListQuery(nil)
	if err != nil || q.Page != 1 || q.Limit != 20 || len(q.Statuses) != 0 {
		t.Fatal(q, err)
	}
	input := domain.UserRoleListInput{"page": {"0002147483647"}, "limit": {"00100"}, "application_code": {"clm"}, "statuses": {" REVOKED,ACTIVE,ACTIVE "}, "keyword": {" a\t\r\nb  "}, "last_id": {" x+/= "}, "permission": {"evil"}}
	q, err = domain.ParseUserRoleListQuery(input)
	if err != nil || q.Keyword != "a b" || q.LastID != " x+/= " || !reflect.DeepEqual(q.Statuses, []string{"ACTIVE", "REVOKED"}) || q.Validate() != nil {
		t.Fatal(q, err)
	}
	q.Statuses[0] = "mutated"
	if input["statuses"][0] != " REVOKED,ACTIVE,ACTIVE " {
		t.Fatal("input mutated")
	}
	for _, field := range []string{"page", "limit", "keyword", "application_code", "statuses", "last_id"} {
		userRoleQueryError(t, domain.UserRoleListInput{field: {"1", "1"}}, field)
	}
	for _, field := range []string{"page", "limit"} {
		for _, raw := range []string{"", "0", "-1", "+1", " 1", "1 ", "1.2", "１", "2147483648", "99999999999999999999999999999"} {
			userRoleQueryError(t, domain.UserRoleListInput{field: {raw}}, field)
		}
	}
	userRoleQueryError(t, domain.UserRoleListInput{"limit": {"101"}}, "limit")
	for _, raw := range []string{"", "Clm", " clm", "clm ", "1clm", "cl-m", "é", strings.Repeat("a", 51)} {
		userRoleQueryError(t, domain.UserRoleListInput{"application_code": {raw}}, "application_code")
	}
	for _, raw := range []string{"", "ACTIVE,", "active", "DISABLED", " ACTIVE,\tREVOKED", "REVOKED,,ACTIVE"} {
		userRoleQueryError(t, domain.UserRoleListInput{"statuses": {raw}}, "statuses")
	}
	for _, raw := range []string{strings.Repeat("a", 201), "\x00a", "a\x7f", "é", "\x08a"} {
		userRoleQueryError(t, domain.UserRoleListInput{"keyword": {raw}}, "keyword")
	}
	for _, raw := range []string{"", " \t\r\n\v\f", strings.Repeat("a", 200)} {
		if _, err = domain.ParseUserRoleListQuery(domain.UserRoleListInput{"keyword": {raw}}); err != nil {
			t.Fatal(err)
		}
	}
	for _, field := range []string{"fields", "sort", "search", "search_fields", "search_mode", "created_from", "created_to", "updated_from", "updated_to", "code"} {
		userRoleQueryError(t, domain.UserRoleListInput{field: {""}}, field)
	}
	userRoleQueryError(t, domain.UserRoleListInput{"last_id": {strings.Repeat("x", 2049)}}, "last_id")
	userRoleQueryError(t, domain.UserRoleListInput{"page": {"0"}, "limit": {"0"}, "application_code": {"!"}, "statuses": {"DISABLED"}, "keyword": {"é"}}, "page", "limit", "application_code", "statuses", "keyword")
	for _, mutate := range []func(*domain.UserRoleListQuery){func(q *domain.UserRoleListQuery) { q.Page = 0 }, func(q *domain.UserRoleListQuery) { q.Limit = 101 }, func(q *domain.UserRoleListQuery) { q.Keyword = "é" }, func(q *domain.UserRoleListQuery) { q.ApplicationCode = "!" }, func(q *domain.UserRoleListQuery) { q.LastID = strings.Repeat("x", 2049) }, func(q *domain.UserRoleListQuery) { q.Statuses = []string{"DISABLED"} }} {
		q, _ = domain.ParseUserRoleListQuery(nil)
		mutate(&q)
		if q.Validate() == nil {
			t.Fatal(q)
		}
	}
}
func userRoleRaw() domain.UserRoleCommandRequest {
	reason := "  approved\r\nreason  "
	return domain.UserRoleCommandRequest{UserID: "00023", UserRoleCommandBody: domain.UserRoleCommandBody{RoleIDs: []string{"11", "00010"}, Reason: &reason}, Version: 9007199254740993, IdempotencyKey: " opaque, key "}
}
func userRoleCommands(t *testing.T) {
	r := userRoleRaw()
	got, err := r.NormalizeAndValidate()
	if err != nil || got.UserID != 23 || got.Version != 9007199254740993 || got.Reason != "approved\r\nreason" || !reflect.DeepEqual(got.RoleIDs, []uint64{10, 11}) || got.Validate() != nil {
		t.Fatal(got, err)
	}
	got.RoleIDs[0] = 99
	if r.RoleIDs[0] != "11" || *r.Reason != "  approved\r\nreason  " {
		t.Fatal("caller mutated")
	}
	for _, ids := range [][]string{nil, {}, {"10", "10"}, {"10", "00010"}, {"0"}, {"+1"}, {" 1"}, {"1 "}, {"１"}, {"2147483648"}, {"clm.reader"}, {""}} {
		r = userRoleRaw()
		r.RoleIDs = ids
		if _, err = r.NormalizeAndValidate(); err == nil {
			t.Fatal(ids)
		}
	}
	r = userRoleRaw()
	r.RoleIDs = []string{"0002147483647", "0001"}
	edge, e := r.NormalizeAndValidate()
	if e != nil || !reflect.DeepEqual(edge.RoleIDs, []uint64{1, 2147483647}) {
		t.Fatal(edge, e)
	}
	for _, n := range []int{1, 20, 21} {
		r = userRoleRaw()
		r.RoleIDs = nil
		for i := 1; i <= n; i++ {
			r.RoleIDs = append(r.RoleIDs, strconv.Itoa(i))
		}
		_, err = r.NormalizeAndValidate()
		if (err == nil) != (n <= 20) {
			t.Fatal(n, err)
		}
	}
	for _, reason := range []*string{nil, statusString(""), statusString(" ab "), statusString(strings.Repeat("a", 501)), statusString("\tabc"), statusString("abc\t"), statusString("éabc"), statusString("abc\x00"), statusString("abc\x7f")} {
		r = userRoleRaw()
		r.RoleIDs = []string{"0"}
		r.Reason = reason
		_, err = r.NormalizeAndValidate()
		var f cd.FieldErrors
		if !errors.As(err, &f) || len(f) != 2 {
			t.Fatal(f, err)
		}
		for _, lang := range []locale.Language{locale.English, locale.Indonesian, locale.Chinese} {
			if len(f.Messages(lang)["role_ids"]) == 0 {
				t.Fatal(lang)
			}
		}
	}
	for _, s := range []string{"abc", strings.Repeat("a", 500), " \r\n a  b! \r\n"} {
		r = userRoleRaw()
		r.Reason = &s
		if _, err = r.NormalizeAndValidate(); err != nil {
			t.Fatal(err)
		}
	}
	for _, version := range []string{"1", "0001", "9007199254740993", "9223372036854775807"} {
		for _, key := range []string{"x", " x, y ", strings.Repeat("x", 200)} {
			k, v, e := domain.ParseUserRoleHeaders([]string{key}, []string{version})
			if e != nil || k != key || v == 0 {
				t.Fatal(k, v, e)
			}
		}
	}
	for _, values := range [][]string{nil, {}, {"x", "x"}} {
		if _, _, e := domain.ParseUserRoleHeaders(values, []string{"1"}); e == nil {
			t.Fatal(values)
		}
		if _, _, e := domain.ParseUserRoleHeaders([]string{"x"}, values); e == nil {
			t.Fatal(values)
		}
	}
	for _, s := range []string{"", " ", "é", "\tx", "x\n", strings.Repeat("x", 201)} {
		if _, _, e := domain.ParseUserRoleHeaders([]string{s}, []string{"1"}); e == nil {
			t.Fatal("key", s)
		}
	}
	for _, s := range []string{"", "0", "+1", "-1", " 1", "1 ", "1.0", "１", "9223372036854775808"} {
		if _, _, e := domain.ParseUserRoleHeaders([]string{"x"}, []string{s}); e == nil {
			t.Fatal("version", s)
		}
	}
	for _, body := range []string{`{"role_ids":[10]}`, `{"role_ids":["10",11]}`, `{"role_ids":{}}`, `{"reason":1}`} {
		var b domain.UserRoleCommandBody
		if json.Unmarshal([]byte(body), &b) == nil {
			t.Fatal(body)
		}
	}
	for _, body := range []string{`{}`, `{"role_ids":null}`, `{"role_ids":[null],"reason":"abc"}`} {
		var b domain.UserRoleCommandBody
		if json.Unmarshal([]byte(body), &b) != nil {
			t.Fatal(body)
		}
		r = userRoleRaw()
		r.UserRoleCommandBody = b
		if _, err = r.NormalizeAndValidate(); err == nil {
			t.Fatal(body)
		}
	}
	for _, mutate := range []func(*domain.UserRoleCommand){func(c *domain.UserRoleCommand) { c.UserID = 0 }, func(c *domain.UserRoleCommand) { c.Version = 0 }, func(c *domain.UserRoleCommand) { c.Version = 1 << 63 }, func(c *domain.UserRoleCommand) { c.IdempotencyKey = "" }, func(c *domain.UserRoleCommand) { c.RoleIDs = []uint64{11, 10} }, func(c *domain.UserRoleCommand) { c.RoleIDs = []uint64{0} }, func(c *domain.UserRoleCommand) { c.Reason = "é" }} {
		c, _ := userRoleRaw().NormalizeAndValidate()
		mutate(&c)
		if c.Validate() == nil {
			t.Fatal(c)
		}
	}
}
