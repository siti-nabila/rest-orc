package test_scenarios

import (
	"errors"
	"strings"
	"testing"

	cd "github.com/siti-nabila/api-contracts/pkg/dictionary"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func UserStatusScenarios() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "status header target boundaries and opaque keys", Run: statusHeaders},
		{Name: "status reason presence raw bytes normalization and safe localized fields", Run: statusBody},
	}
}
func statusString(s string) *string { return &s }
func statusInput() domain.ChangeUserStatusRequest {
	return domain.ChangeUserStatusRequest{UserID: 23, Version: 9007199254740993, IdempotencyKey: "opaque, key", Status: statusString("ACTIVE"), Reason: statusString("  approved\r\nreason  ")}
}
func statusHeaders(t *testing.T) {
	userDetailIDs(t)
	for _, key := range []string{"x", "opaque, key", strings.Repeat("x", 200), " x "} {
		for _, raw := range []string{"1", "00023", "9007199254740993", "9223372036854775807"} {
			got, v, err := domain.ParseUserStatusHeaders([]string{key}, []string{raw})
			if err != nil || got != key || v == 0 {
				t.Fatal(got, v, err)
			}
		}
	}
	for _, key := range []string{"", " ", "é", "x\t", "x\n", "x\x7f", strings.Repeat("x", 201)} {
		if _, _, err := domain.ParseUserStatusHeaders([]string{key}, []string{"1"}); err == nil {
			t.Fatal("key", key)
		}
	}
	for _, raw := range []string{"", "0", "+1", "-1", " 1", "1 ", "１", "0x1", "1.0", "9223372036854775808"} {
		if _, _, err := domain.ParseUserStatusHeaders([]string{"key"}, []string{raw}); err == nil {
			t.Fatal("version", raw)
		}
	}
	for _, values := range [][]string{nil, {}, {"x", "x"}} {
		if _, _, err := domain.ParseUserStatusHeaders(values, []string{"1"}); err == nil {
			t.Fatal("key multiplicity")
		}
		if _, _, err := domain.ParseUserStatusHeaders([]string{"key"}, values); err == nil {
			t.Fatal("version multiplicity")
		}
	}
	for _, mode := range []string{"id", "key", "zero version", "overflow"} {
		r := statusInput()
		switch mode {
		case "id":
			r.UserID = 0
		case "key":
			r.IdempotencyKey = ""
		case "zero version":
			r.Version = 0
		case "overflow":
			r.Version = 1 << 63
		}
		if _, err := r.NormalizeAndValidate(); err == nil {
			t.Fatal(mode)
		}
	}
}
func statusBody(t *testing.T) {
	for _, s := range []string{"PENDING", "ACTIVE", "SUSPENDED", "DEACTIVATED"} {
		for _, reason := range []string{"abc", strings.Repeat("a", 500), "  approved\r\nreason  ", "\r\n a  b! \r\n"} {
			r := statusInput()
			r.Status = statusString(s)
			r.Reason = statusString(reason)
			got, err := r.NormalizeAndValidate()
			if err != nil || *got.Status != s || *got.Reason != strings.Trim(reason, " \r\n") {
				t.Fatal(got, err)
			}
			*got.Status = "changed"
			*got.Reason = "changed"
			if *r.Status != s || *r.Reason != reason {
				t.Fatal("caller mutated")
			}
		}
	}
	for _, s := range []*string{nil, statusString(""), statusString("active"), statusString(" ACTIVE"), statusString("ACTIVE "), statusString("UNKNOWN")} {
		for _, reason := range []*string{nil, statusString(""), statusString("ab"), statusString("  ab "), statusString(strings.Repeat("a", 501)), statusString("\tabc"), statusString("abc\t"), statusString("éabc"), statusString("abc\x00"), statusString("abc\x7f"), statusString("abc\v")} {
			r := statusInput()
			r.Status = s
			r.Reason = reason
			_, err := r.NormalizeAndValidate()
			var fields cd.FieldErrors
			if !errors.As(err, &fields) || len(fields) != 2 {
				t.Fatal(fields, err)
			}
			for _, lang := range []locale.Language{locale.English, locale.Indonesian, locale.Chinese} {
				m := fields.Messages(lang)
				if len(m["status"]) == 0 || len(m["reason"]) == 0 {
					t.Fatal(m)
				}
			}
		}
	}
}
