package test_scenarios

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/siti-nabila/rest-orc/internal/config"
	"github.com/siti-nabila/rest-orc/tests/config/fixtures"
	"github.com/siti-nabila/rest-orc/tests/shared/testutils"
)

func All() []testutils.Scenario {
	return []testutils.Scenario{
		{
			Name: "valid YAML is loaded into typed configuration",
			Run:  validYAMLIsLoaded,
		},
		{
			Name: "environment variables override YAML values",
			Run:  environmentOverridesYAML,
		},
		{
			Name: "unknown YAML field is rejected",
			Run:  unknownYAMLFieldIsRejected,
		},
		{
			Name: "malformed duration is rejected",
			Run:  malformedDurationIsRejected,
		},
		{
			Name: "empty required gRPC target is rejected",
			Run:  emptyGRPCTargetIsRejected,
		},
		{
			Name: "aggressive gRPC keepalive is rejected",
			Run:  aggressiveGRPCKeepaliveIsRejected,
		},
		{
			Name: "malformed environment override is rejected",
			Run:  malformedEnvironmentOverrideIsRejected,
		},
		{
			Name: "validation failure preserves typed error",
			Run:  validationFailurePreservesTypedError,
		},
	}
}

func validYAMLIsLoaded(t *testing.T) {
	// Arrange
	fixtures.ClearConfigurationEnvironment(t)
	path := fixtures.Write(t, fixtures.ValidConfiguration)

	// Act
	cfg, err := config.Load(path)

	// Assert
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.App.Port != 8080 {
		t.Errorf("App.Port = %d, want 8080", cfg.App.Port)
	}
	if cfg.Clients.AuthGRPC.RequestTimeout.Duration != 5*time.Second {
		t.Errorf(
			"AuthGRPC.RequestTimeout = %s, want 5s",
			cfg.Clients.AuthGRPC.RequestTimeout.Duration,
		)
	}
	if !cfg.Clients.BackendHTTP.Keepalive.Enabled {
		t.Error("BackendHTTP.Keepalive.Enabled = false, want true")
	}
}

func environmentOverridesYAML(t *testing.T) {
	// Arrange
	fixtures.ClearConfigurationEnvironment(t)
	t.Setenv("APP_PORT", "9090")
	t.Setenv("AUTH_GRPC_TARGET", "auth-load-balancer:8443")
	t.Setenv("AUTH_GRPC_KEEPALIVE_ENABLED", "true")
	t.Setenv("BACKEND_HTTP_BASE_URL", "http://backend-load-balancer:9000")
	t.Setenv("BACKEND_HTTP_MAX_CONNECTIONS_PER_HOST", "250")
	path := fixtures.Write(t, fixtures.ValidConfiguration)

	// Act
	cfg, err := config.Load(path)

	// Assert
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.App.Port != 9090 {
		t.Errorf("App.Port = %d, want 9090", cfg.App.Port)
	}
	if cfg.Clients.AuthGRPC.Target != "auth-load-balancer:8443" {
		t.Errorf("AuthGRPC.Target = %q, want auth-load-balancer:8443", cfg.Clients.AuthGRPC.Target)
	}
	if !cfg.Clients.AuthGRPC.Keepalive.Enabled {
		t.Error("AuthGRPC.Keepalive.Enabled = false, want true")
	}
	if cfg.Clients.BackendHTTP.BaseURL != "http://backend-load-balancer:9000" {
		t.Errorf(
			"BackendHTTP.BaseURL = %q, want http://backend-load-balancer:9000",
			cfg.Clients.BackendHTTP.BaseURL,
		)
	}
	if cfg.Clients.BackendHTTP.Keepalive.MaxConnectionsPerHost != 250 {
		t.Errorf(
			"MaxConnectionsPerHost = %d, want 250",
			cfg.Clients.BackendHTTP.Keepalive.MaxConnectionsPerHost,
		)
	}
}

func unknownYAMLFieldIsRejected(t *testing.T) {
	assertLoadErrorContains(t, fixtures.WithUnknownField(), "unknown")
}

func malformedDurationIsRejected(t *testing.T) {
	assertLoadErrorContains(t, fixtures.WithMalformedDuration(), "parse duration")
}

func emptyGRPCTargetIsRejected(t *testing.T) {
	assertLoadErrorContains(
		t,
		fixtures.WithEmptyGRPCTarget(),
		"clients.auth_grpc.target",
	)
}

func aggressiveGRPCKeepaliveIsRejected(t *testing.T) {
	assertLoadErrorContains(
		t,
		fixtures.WithAggressiveGRPCKeepalive(),
		"must be at least 5m0s",
	)
}

func malformedEnvironmentOverrideIsRejected(t *testing.T) {
	// Arrange
	fixtures.ClearConfigurationEnvironment(t)
	t.Setenv("APP_PORT", "not-a-port")
	path := fixtures.Write(t, fixtures.ValidConfiguration)

	// Act
	_, err := config.Load(path)

	// Assert
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "APP_PORT") {
		t.Errorf("Load() error = %q, want APP_PORT context", err)
	}
}

func validationFailurePreservesTypedError(t *testing.T) {
	// Arrange
	fixtures.ClearConfigurationEnvironment(t)
	path := fixtures.Write(t, fixtures.WithEmptyAppName())

	// Act
	_, err := config.Load(path)

	// Assert
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	var validationErr *config.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("Load() error type = %T, want *config.ValidationError", err)
	}
	if validationErr.Field != "app.name" {
		t.Errorf("ValidationError.Field = %q, want app.name", validationErr.Field)
	}
}

func assertLoadErrorContains(t *testing.T, content, expected string) {
	t.Helper()

	// Arrange
	fixtures.ClearConfigurationEnvironment(t)
	path := fixtures.Write(t, content)

	// Act
	_, err := config.Load(path)

	// Assert
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("Load() error = %q, want substring %q", err, expected)
	}
}
