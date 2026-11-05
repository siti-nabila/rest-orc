package test_scenarios

import (
	"errors"
	contractdictionary "github.com/siti-nabila/api-contracts/pkg/dictionary"
	"strings"
	"testing"

	"github.com/siti-nabila/api-contracts/pkg/locale"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func All() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "password difference error preserves catalog identity and locales", Run: passwordDifferenceMessages},
		{Name: "profile catalog translates every supported locale", Run: profileMessages},
		{Name: "validation catalog preserves legacy identity and translations", Run: validationMessages},
		{
			Name: "registered static error exposes stable message",
			Run:  registeredStaticError,
		},
		{
			Name: "parameterized error fills message and preserves identity",
			Run:  parameterizedError,
		},
		{
			Name: "wrapped error preserves catalog and cause identity",
			Run:  wrappedError,
		},
		{
			Name: "validation error preserves field problem and catalog identity",
			Run:  validationError,
		},
	}
}

func passwordDifferenceMessages(t *testing.T) {
	err := dictionary.ErrPasswordMustDiffer
	if err.Key() != "users.password_must_differ" {
		t.Fatal("password difference error has an unexpected namespace")
	}
	for _, tc := range []struct {
		language locale.Language
		want     string
	}{
		{locale.English, "must differ from the current password"},
		{locale.Indonesian, "harus berbeda dari password saat ini"},
		{locale.Chinese, "必须与当前密码不同"},
	} {
		if err.Message(tc.language) != tc.want {
			t.Errorf("password difference translation differs for %s", tc.language)
		}
	}
}

func registeredStaticError(t *testing.T) {
	err := dictionary.ErrHTTPRequestRequired

	if err.Error() != "execute HTTP request: request must not be nil" {
		t.Fatalf("unexpected message: %q", err.Error())
	}
}

func parameterizedError(t *testing.T) {
	err := dictionary.UnsupportedHTTPClientBaseURLScheme("ftp://backend")

	if !errors.Is(err, dictionary.ErrUnsupportedHTTPClientBaseURLScheme) {
		t.Fatal("expected parameterized error to preserve catalog identity")
	}
	if err.Error() != `parse HTTP client base URL "ftp://backend": scheme must be http or https` {
		t.Fatalf("unexpected message: %q", err.Error())
	}
}

func wrappedError(t *testing.T) {
	cause := errors.New("connection refused")
	err := dictionary.CreateGRPCClient("auth-proxy:50051", cause)

	if !errors.Is(err, dictionary.ErrCreateGRPCClient) {
		t.Fatal("expected wrapped error to preserve catalog identity")
	}
	if !errors.Is(err, cause) {
		t.Fatal("expected wrapped error to preserve cause identity")
	}
	if err.Error() != `create gRPC client for target "auth-proxy:50051": connection refused` {
		t.Fatalf("unexpected message: %q", err.Error())
	}
}

func validationError(t *testing.T) {
	problem := dictionary.ErrValueRequired
	err := dictionary.NewValidationError("app.name", problem)

	if err.Field != "app.name" || err.Problem != problem {
		t.Fatalf("unexpected validation details: %#v", err)
	}
	if !errors.Is(err, dictionary.ErrInvalidConfigurationField) {
		t.Fatal("expected validation error to preserve catalog identity")
	}
	if !errors.Is(err, problem) {
		t.Fatal("expected validation error to preserve problem identity")
	}
	if err.Error() != "invalid configuration field app.name: must not be empty" {
		t.Fatalf("unexpected message: %q", err.Error())
	}
}

func validationMessages(t *testing.T) {
	err := dictionary.ErrInvalidPasswordCharacters
	if err.Key() != "users.invalid_password_characters" {
		t.Fatalf("validation key = %q, want legacy key", err.Key())
	}
	for _, tc := range []struct {
		language locale.Language
		want     string
	}{
		{locale.English, "contains invalid characters"},
		{locale.Indonesian, "mengandung karakter tidak valid"},
		{locale.Chinese, "包含无效字符"},
	} {
		if got := err.Message(tc.language); got != tc.want {
			t.Errorf("validation message (%s) = %q, want %q", tc.language, got, tc.want)
		}
	}
}

func profileMessages(t *testing.T) {
	for _, err := range []*contractdictionary.Error{dictionary.ErrInvalidProfileCharacters, dictionary.ErrInvalidCountryCode, dictionary.ErrInvalidPhone} {
		if !strings.HasPrefix(err.Key(), "users.") {
			t.Fatalf("unexpected namespace: %s", err.Key())
		}
		messages := map[string]bool{}
		for _, language := range []locale.Language{locale.English, locale.Indonesian, locale.Chinese} {
			value := err.Message(language)
			if value == "" || value == err.Key() || messages[value] {
				t.Fatalf("invalid translation: %s", value)
			}
			messages[value] = true
		}
	}
}
