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
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func UserApplicationAccessScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "access JSON schemas", Run: accessShapes}, {Name: "access query bounds and ownership", Run: accessQueries}, {Name: "access grant and preconditions", Run: accessCommands}, {Name: "access revoke identity and reason", Run: accessRevoke}}
}
func accessQueryError(t *testing.T, input domain.UserApplicationAccessListInput, fields ...string) {
	t.Helper()
	_, err := domain.ParseUserApplicationAccessListQuery(input)
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
func accessQueries(t *testing.T) {
	q, err := domain.ParseUserApplicationAccessListQuery(nil)
	if err != nil || q.Page != 1 || q.Limit != 20 || len(q.Statuses) != 0 {
		t.Fatal(q, err)
	}
	input := domain.UserApplicationAccessListInput{"page": {"0002147483647"}, "limit": {"00100"}, "application_code": {"clm"}, "statuses": {" REVOKED,ACTIVE,ACTIVE "}, "keyword": {" a\t\r\nb  "}, "last_id": {" x+/= "}, "permission": {"evil"}}
	q, err = domain.ParseUserApplicationAccessListQuery(input)
	if err != nil || q.Keyword != "a b" || q.LastID != " x+/= " || !reflect.DeepEqual(q.Statuses, []string{"ACTIVE", "REVOKED"}) || q.Validate() != nil {
		t.Fatal(q, err)
	}
	q.Statuses[0] = "mutated"
	if input["statuses"][0] != " REVOKED,ACTIVE,ACTIVE " {
		t.Fatal("input mutated")
	}
	for _, field := range []string{"page", "limit", "keyword", "application_code", "statuses", "last_id"} {
		accessQueryError(t, domain.UserApplicationAccessListInput{field: {"1", "1"}}, field)
	}
	for _, field := range []string{"page", "limit"} {
		for _, raw := range []string{"", "0", "-1", "+1", " 1", "1 ", "1.2", "１", "2147483648", "99999999999999999999999999999"} {
			accessQueryError(t, domain.UserApplicationAccessListInput{field: {raw}}, field)
		}
	}
	accessQueryError(t, domain.UserApplicationAccessListInput{"limit": {"101"}}, "limit")
	for _, raw := range []string{"", "Clm", " clm", "clm ", "1clm", "cl-m", "é", strings.Repeat("a", 51)} {
		accessQueryError(t, domain.UserApplicationAccessListInput{"application_code": {raw}}, "application_code")
	}
	for _, raw := range []string{"", "ACTIVE,", "active", "DISABLED", " ACTIVE,\tREVOKED", "REVOKED,,ACTIVE"} {
		accessQueryError(t, domain.UserApplicationAccessListInput{"statuses": {raw}}, "statuses")
	}
	for _, raw := range []string{strings.Repeat("a", 201), "\x00a", "a\x7f", "é", "\x08a"} {
		accessQueryError(t, domain.UserApplicationAccessListInput{"keyword": {raw}}, "keyword")
	}
	for _, raw := range []string{"", " \t\r\n\v\f", strings.Repeat("a", 200)} {
		if _, err = domain.ParseUserApplicationAccessListQuery(domain.UserApplicationAccessListInput{"keyword": {raw}}); err != nil {
			t.Fatal(err)
		}
	}
	for _, field := range []string{"fields", "sort", "search", "search_fields", "search_mode", "created_from", "created_to", "updated_from", "updated_to", "code"} {
		accessQueryError(t, domain.UserApplicationAccessListInput{field: {""}}, field)
	}
	accessQueryError(t, domain.UserApplicationAccessListInput{"last_id": {strings.Repeat("x", 2049)}}, "last_id")
	accessQueryError(t, domain.UserApplicationAccessListInput{"page": {"0"}, "limit": {"0"}, "application_code": {"!"}, "statuses": {"DISABLED"}, "keyword": {"é"}}, "page", "limit", "application_code", "statuses", "keyword")
	for _, mutate := range []func(*domain.UserApplicationAccessListQuery){func(q *domain.UserApplicationAccessListQuery) { q.Page = 0 }, func(q *domain.UserApplicationAccessListQuery) { q.Limit = 101 }, func(q *domain.UserApplicationAccessListQuery) { q.Keyword = "é" }, func(q *domain.UserApplicationAccessListQuery) { q.ApplicationCode = "!" }, func(q *domain.UserApplicationAccessListQuery) { q.LastID = strings.Repeat("x", 2049) }, func(q *domain.UserApplicationAccessListQuery) { q.Statuses = []string{"DISABLED"} }} {
		q, _ = domain.ParseUserApplicationAccessListQuery(nil)
		mutate(&q)
		if q.Validate() == nil {
			t.Fatal(q)
		}
	}
}
func accessRaw() domain.GrantApplicationAccessRequest {
	reason := "  approved\r\nreason  "
	return domain.GrantApplicationAccessRequest{ApplicationAccessCommandRequest: domain.ApplicationAccessCommandRequest{UserID: "00023", ApplicationCode: "clm", Version: 9007199254740993, IdempotencyKey: " opaque, key "}, GrantApplicationAccessBody: domain.GrantApplicationAccessBody{RoleIDs: []string{"11", "00010"}, Reason: &reason}}
}
func accessCommands(t *testing.T) {
	r := accessRaw()
	got, err := r.NormalizeAndValidate()
	if err != nil || got.UserID != 23 || got.Version != 9007199254740993 || got.Reason != "approved\r\nreason" || !reflect.DeepEqual(got.RoleIDs, []uint64{10, 11}) || got.Validate() != nil {
		t.Fatal(got, err)
	}
	got.RoleIDs[0] = 99
	if r.RoleIDs[0] != "11" || *r.Reason != "  approved\r\nreason  " {
		t.Fatal("caller mutated")
	}
	for _, ids := range [][]string{nil, {}, {"10", "10"}, {"10", "00010"}, {"0"}, {"+1"}, {" 1"}, {"1 "}, {"１"}, {"2147483648"}, {"clm.reader"}, {""}} {
		r = accessRaw()
		r.RoleIDs = ids
		if _, err = r.NormalizeAndValidate(); err == nil {
			t.Fatal(ids)
		}
	}
	r = accessRaw()
	r.RoleIDs = []string{"0002147483647", "0001"}
	edge, e := r.NormalizeAndValidate()
	if e != nil || !reflect.DeepEqual(edge.RoleIDs, []uint64{1, 2147483647}) {
		t.Fatal(edge, e)
	}
	for _, n := range []int{1, 20, 21} {
		r = accessRaw()
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
		r = accessRaw()
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
		r = accessRaw()
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
		var b domain.GrantApplicationAccessBody
		if json.Unmarshal([]byte(body), &b) == nil {
			t.Fatal(body)
		}
	}
	for _, body := range []string{`{}`, `{"role_ids":null}`, `{"role_ids":[null],"reason":"abc"}`} {
		var b domain.GrantApplicationAccessBody
		if json.Unmarshal([]byte(body), &b) != nil {
			t.Fatal(body)
		}
		r = accessRaw()
		r.GrantApplicationAccessBody = b
		if _, err = r.NormalizeAndValidate(); err == nil {
			t.Fatal(body)
		}
	}
	for _, mutate := range []func(*domain.GrantApplicationAccessCommand){func(c *domain.GrantApplicationAccessCommand) { c.UserID = 0 }, func(c *domain.GrantApplicationAccessCommand) { c.Version = 0 }, func(c *domain.GrantApplicationAccessCommand) { c.Version = 1 << 63 }, func(c *domain.GrantApplicationAccessCommand) { c.IdempotencyKey = "" }, func(c *domain.GrantApplicationAccessCommand) { c.RoleIDs = []uint64{11, 10} }, func(c *domain.GrantApplicationAccessCommand) { c.RoleIDs = []uint64{0} }, func(c *domain.GrantApplicationAccessCommand) { c.Reason = "é" }} {
		c, _ := accessRaw().NormalizeAndValidate()
		mutate(&c)
		if c.Validate() == nil {
			t.Fatal(c)
		}
	}
}

func accessShapes(t *testing.T) {
	r := domain.ListUserApplicationAccessResponse{Items: []domain.UserApplicationAccess{{ApplicationID: 5, GrantedAt: "2026-10-20T08:09:10.123456789Z"}}, Total: 1 << 32, UserID: 23, Version: "9007199254740993"}
	b, _ := json.Marshal(r)
	var m map[string]json.RawMessage
	_ = json.Unmarshal(b, &m)
	var items []map[string]json.RawMessage
	_ = json.Unmarshal(m["items"], &items)
	if len(m) != 10 || len(items[0]) != 11 || string(m["total"]) != "4294967296" || string(m["version"]) != `"9007199254740993"` || string(items[0]["is_system"]) != "false" {
		t.Fatal(string(b))
	}
	for _, k := range []string{"granted_by", "revoked_by", "revoked_at", "revoke_reason"} {
		if string(items[0][k]) != "null" {
			t.Fatal(k)
		}
	}
	b, _ = json.Marshal(domain.ApplicationAccessCommandResponse{UserID: 23, ApplicationID: 5, ChangedRoleIDs: []uint64{}, Version: "9223372036854775807"})
	m = nil
	_ = json.Unmarshal(b, &m)
	if len(m) != 6 || string(m["changed_role_ids"]) != "[]" || string(m["changed"]) != "false" || string(m["access_status"]) != "null" {
		t.Fatal(string(b))
	}
}
func accessRevoke(t *testing.T) {
	_, err := domain.ParseUserApplicationAccessListQuery(domain.UserApplicationAccessListInput{"sort": {""}})
	var fields cd.FieldErrors
	if !errors.As(err, &fields) {
		t.Fatal(err)
	}
	for _, language := range []locale.Language{locale.English, locale.Indonesian, locale.Chinese} {
		if got := fields.Messages(language)["sort"]; len(got) != 1 || got[0] != dictionary.ErrUnsupportedAccessControl.Message(language) {
			t.Fatal(got)
		}
	}

	base := accessRaw()
	r := domain.RevokeApplicationAccessRequest{ApplicationAccessCommandRequest: base.ApplicationAccessCommandRequest, RevokeApplicationAccessBody: domain.RevokeApplicationAccessBody{Reason: base.Reason}}
	c, e := r.NormalizeAndValidate()
	if e != nil || c.Validate() != nil || c.ApplicationCode != "clm" || c.Reason != "approved\r\nreason" {
		t.Fatal(c, e)
	}
	for _, code := range []string{"", "Clm", " clm", "clm ", "1clm", "cl-m", "é", strings.Repeat("a", 51)} {
		r.ApplicationCode = code
		if _, e = r.NormalizeAndValidate(); e == nil {
			t.Fatal(code)
		}
	}
	for _, code := range []string{"a", "iam", strings.Repeat("a", 50)} {
		r.ApplicationCode = code
		if _, e = r.NormalizeAndValidate(); e != nil {
			t.Fatal(code, e)
		}
	}
	for _, id := range []string{"", "me", "0", "+1", " 1", "1.0", "１", "2147483648"} {
		r.UserID = id
		if _, e = r.NormalizeAndValidate(); e == nil {
			t.Fatal(id)
		}
	}
	for _, raw := range []string{`{"reason":"abc","role_ids":["10"]}`, `{"reason":"abc","role_ids":{}}`, `{"reason":"abc","role_ids":false}`} {
		var body domain.RevokeApplicationAccessBody
		if json.Unmarshal([]byte(raw), &body) != nil || body.Reason == nil {
			t.Fatal(raw)
		}
	}
	q, e := domain.ParseUserApplicationAccessListQuery(domain.UserApplicationAccessListInput{"last_id": {strings.Repeat("x", 2048)}, "application_code": {strings.Repeat("a", 50)}})
	if e != nil || len(q.LastID) != 2048 {
		t.Fatal(e)
	}
}
