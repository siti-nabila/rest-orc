package test_scenarios

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	cd "github.com/siti-nabila/api-contracts/pkg/dictionary"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func UserCreationScenarios() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "creation required inputs opaque key and initial status", Run: creationPresence},
		{Name: "creation credential profile boundaries and copying", Run: creationValidation},
	}
}
func creationBody() domain.CreateUserBody {
	var b domain.CreateUserBody
	_ = json.Unmarshal([]byte(`{"email":" User.Name@Example.COM ","password":"simple","name":"  NASA  o'neil-jean  ","phone":"0812345678901","country_code":"62"}`), &b)
	return b
}
func creationPresence(t *testing.T) {
	for _, keys := range [][]string{nil, {""}, {" "}, {"a", "a"}, {"a\t"}, {"a\n"}, {"é"}, {"\x7f"}, {strings.Repeat("a", 201)}} {
		if _, err := domain.ParseUserCreationKey(keys); err == nil {
			t.Fatal("invalid key", keys)
		}
	}
	for _, key := range []string{"x", "opaque, key", " x ", strings.Repeat("x", 200)} {
		got, err := domain.ParseUserCreationKey([]string{key})
		if err != nil || got != key {
			t.Fatal(got, err)
		}
	}
	for _, raw := range []string{`{}`, `{"email":null,"password":null,"name":null,"phone":null,"country_code":null}`, `{"permissions":{"iam.user.create":true},"field_mask":["name"]}`} {
		var b domain.CreateUserBody
		_ = json.Unmarshal([]byte(raw), &b)
		_, err := (domain.CreateUserRequest{Body: b, IdempotencyKey: "key"}).NormalizeAndValidate()
		var fields cd.FieldErrors
		if !errors.As(err, &fields) || len(fields) != 5 {
			t.Fatal(err)
		}
	}
	for _, status := range []string{"<nil>", "PENDING", "ACTIVE", "SUSPENDED", "DEACTIVATED", "", " PENDING", "pending", "PENDING "} {
		b := creationBody()
		if status != "<nil>" {
			b.Status = &status
		}
		got, err := (domain.CreateUserRequest{Body: b, IdempotencyKey: "key"}).NormalizeAndValidate()
		if status == "<nil>" || status == "PENDING" {
			if err != nil || got.Body.Status == nil || *got.Body.Status != "PENDING" {
				t.Fatal(got, err)
			}
		} else if err == nil {
			t.Fatal(status)
		}
	}
	for _, field := range []string{"email", "password", "name", "phone", "country_code", "address", "status"} {
		var b domain.CreateUserBody
		if json.Unmarshal([]byte(`{"`+field+`":123}`), &b) == nil {
			t.Fatal(field)
		}
	}
}
func creationValidation(t *testing.T) {
	type row struct {
		field, value string
		valid        bool
	}
	rows := []row{
		{"email", "a@b.c", false}, {"email", "ab@c.d", true}, {"email", strings.Repeat("a", 45) + "@b.co", true}, {"email", strings.Repeat("a", 46) + "@b.co", false},
		{"email", "a+b@example.com", false}, {"email", "a@localhost", false}, {"email", "\u00a0a@b.co", false}, {"email", "a@b.co\u00a0", false}, {"email", "a\n@b.co", false}, {"email", "a~b@c-d.co", true},
		{"password", "12345", false}, {"password", "123456", true}, {"password", strings.Repeat("!", 50), true}, {"password", strings.Repeat("!", 51), false},
		{"password", "abc def", false}, {"password", "abcdef\t", false}, {"password", "abcdef\x7f", false}, {"password", "abcéfg", false},
		{"name", "ab", false}, {"name", "abc", true}, {"name", strings.Repeat("a", 100), true}, {"name", strings.Repeat("a", 101), false}, {"name", "\tAlice", false}, {"name", "éAlice", false},
		{"phone", strings.Repeat("8", 11), false}, {"phone", strings.Repeat("8", 12), true}, {"phone", strings.Repeat("8", 20), true}, {"phone", strings.Repeat("8", 21), false}, {"phone", "81234567890x", false},
		{"address", "", false}, {"address", "ab", false}, {"address", "abc", true}, {"address", strings.Repeat("a", 200), true}, {"address", strings.Repeat("a", 201), false}, {"address", " X\r\nY ", true}, {"address", "abc\t", false}, {"address", "abcé", false},
		{"country_code", "+1", true}, {"country_code", "123", true}, {"country_code", "012", false}, {"country_code", " 62", false}, {"country_code", "+1234", false}, {"country_code", "", false},
	}
	for _, tc := range rows {
		t.Run(tc.field+"/"+tc.value, func(t *testing.T) {
			b := creationBody()
			raw, _ := json.Marshal(b)
			var values map[string]any
			_ = json.Unmarshal(raw, &values)
			values[tc.field] = tc.value
			raw, _ = json.Marshal(values)
			_ = json.Unmarshal(raw, &b)
			_, err := (domain.CreateUserRequest{Body: b, IdempotencyKey: "key"}).NormalizeAndValidate()
			if (err == nil) != tc.valid {
				t.Fatal(tc, err)
			}
		})
	}
	for _, phone := range []string{"+62812345678901", "62812345678901", "0812345678901", "812345678901", "62628123456789", "00628123456789"} {
		b := creationBody()
		b.Phone = &phone
		before, _ := json.Marshal(b)
		got, err := (domain.CreateUserRequest{Body: b, IdempotencyKey: "opaque, key"}).NormalizeAndValidate()
		want := "812345678901"
		if phone == "62628123456789" {
			want = "628123456789"
		}
		if phone == "00628123456789" {
			want = "0628123456789"
		}
		if err != nil || *got.Body.Name != "NASA O'Neil-Jean" || *got.Body.Phone != want || *got.Body.CountryCode != "+62" || *got.Body.Email != *b.Email || *got.Body.Password != *b.Password || got.Body.Address != nil {
			t.Fatal(got, err)
		}
		for _, p := range []*string{got.Body.Email, got.Body.Password, got.Body.Name, got.Body.Phone, got.Body.CountryCode, got.Body.Status} {
			*p = "dependency mutation"
		}
		after, _ := json.Marshal(b)
		if string(before) != string(after) {
			t.Fatal("alias")
		}
	}
	bad := "\tsecret-invalid"
	b := domain.CreateUserBody{Email: &bad, Password: &bad, Name: &bad, Phone: &bad, CountryCode: &bad, Address: &bad, Status: &bad}
	_, err := (domain.CreateUserRequest{Body: b, IdempotencyKey: "key"}).NormalizeAndValidate()
	var fields cd.FieldErrors
	if !errors.As(err, &fields) || len(fields) != 7 {
		t.Fatal(err)
	}
	for _, lang := range []locale.Language{locale.English, locale.Indonesian, locale.Chinese} {
		messages := fields.Messages(lang)
		raw, _ := json.Marshal(messages)
		if len(messages) != 7 || strings.Contains(string(raw), "secret-invalid") {
			t.Fatal(string(raw))
		}
	}
}
