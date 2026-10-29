package test_scenarios

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/siti-nabila/api-contracts/pkg/dictionary"
	authdictionary "github.com/siti-nabila/api-contracts/pkg/dictionary/auth"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	pkgdictionary "github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func All() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "validates email with backend-compatible pattern without mutation", Run: emailValidation},
		{Name: "distinguishes register characters from legacy login credentials", Run: passwordCharacters},
		{Name: "validates required and byte length for both credentials", Run: credentialLengths},
	}
}

func emailValidation(t *testing.T) {
	cases := []struct {
		name, value string
		want        []error
	}{
		{"empty", "", []error{authdictionary.ErrRequired, authdictionary.ErrMinLength(6)}},
		{"whitespace only", " \t\r\n", []error{authdictionary.ErrRequired, authdictionary.ErrMinLength(6)}},
		{"five bytes", "a@b.c", []error{authdictionary.ErrMinLength(6)}},
		{"six bytes", "a@b.co", nil},
		{"fifty bytes", strings.Repeat("a", 45) + "@b.co", nil},
		{"fifty one bytes", strings.Repeat("a", 46) + "@b.co", []error{authdictionary.ErrMaxLength(50)}},
		{"length after trim", " " + strings.Repeat("a", 45) + "@b.co ", nil},
		{"uppercase and whitespace", " User.Name@Example.COM ", nil},
		{"allowed punctuation", "a._~-z@b._~-z.co", nil},
		{"plus addressing", "user+tag@example.com", []error{authdictionary.ErrInvalidEmail}},
		{"leading dot", ".user@example.com", []error{authdictionary.ErrInvalidEmail}},
		{"trailing dot", "user.@example.com", []error{authdictionary.ErrInvalidEmail}},
		{"missing domain dot", "user@example", []error{authdictionary.ErrInvalidEmail}},
		{"domain punctuation at edge", "user@-example.com", []error{authdictionary.ErrInvalidEmail}},
		{"embedded space", "user name@example.com", []error{authdictionary.ErrInvalidEmail}},
		{"unicode local part", "usér@example.com", []error{authdictionary.ErrInvalidEmail}},
		{"unicode whitespace", "\u00a0user@example.com\u00a0", []error{authdictionary.ErrInvalidEmail}},
		{"unicode case fold", "\u0130@example.com", []error{authdictionary.ErrInvalidEmail}},
		{"short malformed", "x", []error{authdictionary.ErrMinLength(6), authdictionary.ErrInvalidEmail}},
	}
	for _, tc := range cases {
		for _, mode := range []string{"register", "login"} {
			t.Run(tc.name+"/"+mode, func(t *testing.T) {
				request := domain.AuthRequest{Email: tc.value, Password: "valid-password"}
				before := request
				err := validate(request, mode)
				want := dictionary.FieldErrors{}
				if len(tc.want) > 0 {
					want["email"] = tc.want
				}
				assertFieldErrors(t, err, want)
				if request != before {
					t.Error("validation mutated the input")
				}
			})
		}
	}
}

func credentialLengths(t *testing.T) {
	for _, mode := range []string{"register", "login"} {
		for _, tc := range []struct {
			name, password string
			want           []error
		}{
			{"empty", "", []error{authdictionary.ErrRequired, authdictionary.ErrMinLength(6)}},
			{"five bytes", "aaaaa", []error{authdictionary.ErrMinLength(6)}},
			{"six bytes", "aaaaaa", nil},
			{"fifty bytes", strings.Repeat("a", 50), nil},
			{"fifty one bytes", strings.Repeat("a", 51), []error{authdictionary.ErrMaxLength(50)}},
		} {
			t.Run(mode+"/"+tc.name, func(t *testing.T) {
				request := domain.AuthRequest{Email: "a@b.co", Password: tc.password}
				before := request
				want := dictionary.FieldErrors{}
				if len(tc.want) > 0 {
					want["password"] = tc.want
				}
				assertFieldErrors(t, validate(request, mode), want)
				if request != before {
					t.Error("validation mutated the credential")
				}
			})
		}
		t.Run(mode+"/both empty", func(t *testing.T) {
			assertFieldErrors(t, validate(domain.AuthRequest{}, mode), dictionary.FieldErrors{
				"email":    {authdictionary.ErrRequired, authdictionary.ErrMinLength(6)},
				"password": {authdictionary.ErrRequired, authdictionary.ErrMinLength(6)},
			})
		})
	}
}

func validate(request domain.AuthRequest, mode string) error {
	if mode == "register" {
		return request.ValidateRegister()
	}
	return request.ValidateLogin()
}

func assertFieldErrors(t *testing.T, err error, want dictionary.FieldErrors) {
	t.Helper()
	if want.Empty() {
		if err != nil {
			t.Fatalf("valid input returned validation errors: %v", err)
		}
		return
	}
	got, ok := errors.AsType[dictionary.FieldErrors](err)
	if !ok {
		t.Fatalf("validation error type = %T, want FieldErrors", err)
	}
	if len(got) != len(want) {
		t.Fatalf("error field count = %d, want %d", len(got), len(want))
	}
	for field, list := range want {
		actual := got[field]
		if len(actual) != len(list) {
			t.Fatalf("%s error count = %d, want %d", field, len(actual), len(list))
		}
		for i, expected := range list {
			if !errors.Is(actual[i], expected) {
				t.Errorf("%s rule %d identity differs", field, i)
			}
		}
	}
	if !reflect.DeepEqual(got.Messages(locale.English), want.Messages(locale.English)) {
		t.Error("validation messages differ from expected catalog parameters")
	}
}

func passwordCharacters(t *testing.T) {
	cases := []struct {
		name, password string
		lengthError    error
		invalid        bool
	}{
		{"lowercase only", "aaaaaa", nil, false},
		{"digits only", "123456", nil, false},
		{"symbols only", "!!!!!!", nil, false},
		{"printable boundaries", "!aaaa~", nil, false},
		{"space", "abc de", nil, true},
		{"spaces only", "      ", nil, true},
		{"tab", "abc\tde", nil, true},
		{"CR", "abc\rde", nil, true},
		{"LF", "abc\nde", nil, true},
		{"NUL", "abc\x00de", nil, true},
		{"DEL", "abc\x7fde", nil, true},
		{"unicode six bytes", "ééé", nil, true},
		{"unicode fifty bytes", strings.Repeat("é", 25), nil, true},
		{"short invalid", "ab cd", authdictionary.ErrMinLength(6), true},
		{"long invalid", strings.Repeat(" ", 51), authdictionary.ErrMaxLength(50), true},
		{"multiple invalid characters", " \t\r\n\x00é", nil, true},
	}
	for _, tc := range cases {
		for _, mode := range []string{"register", "login"} {
			t.Run(tc.name+"/"+mode, func(t *testing.T) {
				req := domain.AuthRequest{Email: "a@b.co", Password: tc.password}
				before := req
				want := dictionary.FieldErrors{}
				if tc.lengthError != nil {
					want.Add("password", tc.lengthError)
				}
				if mode == "register" && tc.invalid {
					want.Add("password", pkgdictionary.ErrInvalidPasswordCharacters)
				}
				assertFieldErrors(t, validate(req, mode), want)
				if req != before {
					t.Error("credential changed during validation")
				}
			})
		}
	}
}
