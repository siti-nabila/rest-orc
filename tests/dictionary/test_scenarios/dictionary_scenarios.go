package test_scenarios

import (
	"errors"
	"testing"

	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/tests/shared/testutils"
)

func All() []testutils.Scenario {
	return []testutils.Scenario{
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
