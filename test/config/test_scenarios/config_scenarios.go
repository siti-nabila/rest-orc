package test_scenarios

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/siti-nabila/rest-orc/internal/config"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/test/config/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func All() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "browser config is required and safe", Run: browserConfiguration},
		{Name: "service credentials are required canonical and never echoed", Run: serviceCredentials},
		{Name: "valid environment is loaded into typed configuration", Run: validEnvironmentIsLoaded},
		{Name: "malformed duration is rejected", Run: malformedDurationIsRejected},
		{Name: "empty required gRPC target is rejected", Run: emptyGRPCTargetIsRejected},
		{Name: "aggressive gRPC keepalive is rejected", Run: aggressiveGRPCKeepaliveIsRejected},
		{Name: "malformed integer environment value is rejected", Run: malformedEnvironmentValueIsRejected},
		{Name: "validation failure preserves typed error", Run: validationFailurePreservesTypedError},
		{Name: "validation failure preserves registered problem", Run: validationFailurePreservesRegisteredProblem},
		{Name: "keepalive validation preserves parameterized minimum", Run: keepaliveValidationPreservesParameterizedMinimum},
	}
}

func validEnvironmentIsLoaded(t *testing.T) {
	fixtures.SetValidEnvironment(t)

	cfg, err := config.Load()

	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.App.Port != 8080 {
		t.Errorf("App.Port = %d, want 8080", cfg.App.Port)
	}
	if cfg.Clients.AuthGRPC.RequestTimeout.Duration != 5*time.Second {
		t.Errorf("AuthGRPC.RequestTimeout = %s, want 5s", cfg.Clients.AuthGRPC.RequestTimeout.Duration)
	}
}

func malformedDurationIsRejected(t *testing.T) {
	fixtures.SetValidEnvironment(t)
	t.Setenv("SERVER_READ_TIMEOUT", "sometime")
	assertLoadErrorContains(t, "SERVER_READ_TIMEOUT")
}

func emptyGRPCTargetIsRejected(t *testing.T) {
	fixtures.SetValidEnvironment(t)
	t.Setenv("AUTH_GRPC_TARGET", "")
	assertLoadErrorContains(t, "clients.auth_grpc.target")
}

func aggressiveGRPCKeepaliveIsRejected(t *testing.T) {
	fixtures.SetValidEnvironment(t)
	t.Setenv("AUTH_GRPC_KEEPALIVE_ENABLED", "true")
	t.Setenv("AUTH_GRPC_KEEPALIVE_TIME", "1m")
	assertLoadErrorContains(t, "must be at least 5m0s")
}

func malformedEnvironmentValueIsRejected(t *testing.T) {
	fixtures.SetValidEnvironment(t)
	t.Setenv("APP_PORT", "not-a-port")
	assertLoadErrorContains(t, "APP_PORT")
}

func validationFailurePreservesTypedError(t *testing.T) {
	fixtures.SetValidEnvironment(t)
	t.Setenv("APP_NAME", "")

	_, err := config.Load()

	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	validationErr, ok := errors.AsType[*dictionary.ValidationError](err)
	if !ok {
		t.Fatalf("Load() error type = %T, want *dictionary.ValidationError", err)
	}
	if validationErr.Field != "app.name" {
		t.Errorf("ValidationError.Field = %q, want app.name", validationErr.Field)
	}
}

func validationFailurePreservesRegisteredProblem(t *testing.T) {
	fixtures.SetValidEnvironment(t)
	t.Setenv("APP_NAME", "")

	_, err := config.Load()

	if !errors.Is(err, dictionary.ErrValueRequired) {
		t.Errorf("Load() error = %v, want ErrValueRequired in error chain", err)
	}
}

func keepaliveValidationPreservesParameterizedMinimum(t *testing.T) {
	fixtures.SetValidEnvironment(t)
	t.Setenv("AUTH_GRPC_KEEPALIVE_ENABLED", "true")
	t.Setenv("AUTH_GRPC_KEEPALIVE_TIME", "1m")

	_, err := config.Load()

	if !errors.Is(err, dictionary.ErrMinimumDurationWhenEnabled) {
		t.Errorf("Load() error = %v, want ErrMinimumDurationWhenEnabled in error chain", err)
	}
}

func assertLoadErrorContains(t *testing.T, expected string) {
	t.Helper()

	_, err := config.Load()

	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("Load() error = %q, want substring %q", err, expected)
	}
}
