package test_scenarios

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/healthcheck"
	"github.com/siti-nabila/rest-orc/internal/app"
	"github.com/siti-nabila/rest-orc/internal/config"
	"github.com/siti-nabila/rest-orc/tests/app/fixtures"
	"github.com/siti-nabila/rest-orc/tests/shared/testutils"
)

func All() []testutils.Scenario {
	return []testutils.Scenario{
		{
			Name: "valid config creates reusable clients and closes resources",
			Run:  validConfigCreatesAndClosesClients,
		},
		{
			Name: "server uses configured Fiber timeouts",
			Run:  serverUsesConfiguredFiberTimeouts,
		},
		{
			Name: "Fiber liveness endpoint returns healthy status",
			Run:  livenessEndpointReturnsHealthyStatus,
		},
		{
			Name: "feature panic returns safe error and application remains available",
			Run:  featurePanicReturnsSafeError,
		},
		{
			Name: "nil listen context is rejected",
			Run:  nilListenContextIsRejected,
		},
		{
			Name: "nil config is rejected",
			Run:  nilConfigIsRejected,
		},
		{
			Name: "backend creation failure returns dependency context",
			Run:  backendCreationFailureReturnsDependencyContext,
		},
	}
}

func validConfigCreatesAndClosesClients(t *testing.T) {
	// Arrange
	cfg := fixtures.ValidConfig()

	// Act
	application, err := app.New(cfg)

	// Assert
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if application.AuthConnection == nil {
		t.Error("AuthConnection = nil")
	}
	if application.BackendClient == nil {
		t.Error("BackendClient = nil")
	}
	if application.HTTPServer == nil {
		t.Error("HTTPServer = nil")
	}
	if err := application.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func serverUsesConfiguredFiberTimeouts(t *testing.T) {
	// Arrange
	cfg := fixtures.ValidConfig()
	application := newApplication(t, cfg)

	// Act
	serverConfig := application.HTTPServer.Config()

	// Assert
	if serverConfig.AppName != cfg.App.Name {
		t.Errorf("AppName = %q, want %q", serverConfig.AppName, cfg.App.Name)
	}
	if serverConfig.ReadTimeout != cfg.Server.ReadTimeout.Duration {
		t.Errorf(
			"ReadTimeout = %s, want %s",
			serverConfig.ReadTimeout,
			cfg.Server.ReadTimeout.Duration,
		)
	}
	if serverConfig.WriteTimeout != cfg.Server.WriteTimeout.Duration {
		t.Errorf(
			"WriteTimeout = %s, want %s",
			serverConfig.WriteTimeout,
			cfg.Server.WriteTimeout.Duration,
		)
	}
	if serverConfig.IdleTimeout != cfg.Server.IdleTimeout.Duration {
		t.Errorf(
			"IdleTimeout = %s, want %s",
			serverConfig.IdleTimeout,
			cfg.Server.IdleTimeout.Duration,
		)
	}
}

func livenessEndpointReturnsHealthyStatus(t *testing.T) {
	// Arrange
	application := newApplication(t, fixtures.ValidConfig())
	request, err := http.NewRequest(http.MethodGet, healthcheck.LivenessEndpoint, nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	// Act
	response, err := application.HTTPServer.Test(request)

	// Assert
	if err != nil {
		t.Fatalf("HTTPServer.Test() error = %v", err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close liveness response: %v", err)
		}
	}()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read liveness response: %v", err)
	}
	if response.StatusCode != http.StatusOK {
		t.Errorf("status code = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if string(body) != fixtures.HealthyResponseBody {
		t.Errorf("response body = %q, want %q", body, fixtures.HealthyResponseBody)
	}
}

func featurePanicReturnsSafeError(t *testing.T) {
	// Arrange
	application := newApplication(t, fixtures.ValidConfig())
	application.HTTPServer.Get("/feature-panic", func(fiber.Ctx) error {
		panic(fixtures.FeaturePanicMessage)
	})
	panicRequest, err := http.NewRequest(http.MethodGet, "/feature-panic", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	// Act
	panicResponse, err := application.HTTPServer.Test(panicRequest)

	// Assert
	panicBody := readResponseBody(t, panicResponse, err)
	if panicResponse.StatusCode != http.StatusInternalServerError {
		t.Errorf(
			"panic status code = %d, want %d",
			panicResponse.StatusCode,
			http.StatusInternalServerError,
		)
	}
	if panicBody != fixtures.InternalErrorResponseBody {
		t.Errorf(
			"panic response body = %q, want %q",
			panicBody,
			fixtures.InternalErrorResponseBody,
		)
	}
	if strings.Contains(panicBody, fixtures.FeaturePanicMessage) {
		t.Error("panic response exposes internal panic detail")
	}

	livenessRequest, err := http.NewRequest(
		http.MethodGet,
		healthcheck.LivenessEndpoint,
		nil,
	)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	livenessResponse, err := application.HTTPServer.Test(livenessRequest)
	livenessBody := readResponseBody(t, livenessResponse, err)
	if livenessResponse.StatusCode != http.StatusOK {
		t.Errorf(
			"liveness status after panic = %d, want %d",
			livenessResponse.StatusCode,
			http.StatusOK,
		)
	}
	if livenessBody != fixtures.HealthyResponseBody {
		t.Errorf(
			"liveness body after panic = %q, want %q",
			livenessBody,
			fixtures.HealthyResponseBody,
		)
	}
}

func nilListenContextIsRejected(t *testing.T) {
	// Arrange
	application := newApplication(t, fixtures.ValidConfig())

	// Act
	err := application.Listen(nil)

	// Assert
	if err == nil {
		t.Fatal("Listen(nil) error = nil, want error")
	}
}

func nilConfigIsRejected(t *testing.T) {
	// Act
	_, err := app.New(nil)

	// Assert
	if err == nil {
		t.Fatal("New(nil) error = nil, want error")
	}
}

func backendCreationFailureReturnsDependencyContext(t *testing.T) {
	// Arrange
	cfg := fixtures.ConfigWithUnreadableBackendCA()

	// Act
	_, err := app.New(cfg)

	// Assert
	if err == nil {
		t.Fatal("New() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "read TLS CA file") {
		t.Errorf("New() error = %q, want TLS CA context", err)
	}
}

func newApplication(t *testing.T, cfg *config.Config) *app.App {
	t.Helper()

	application, err := app.New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := application.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
	return application
}

func readResponseBody(t *testing.T, response *http.Response, err error) string {
	t.Helper()

	if err != nil {
		t.Fatalf("HTTPServer.Test() error = %v", err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close HTTP response: %v", err)
		}
	}()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read HTTP response: %v", err)
	}
	return string(body)
}
