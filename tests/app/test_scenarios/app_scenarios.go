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
	usershttp "github.com/siti-nabila/rest-orc/internal/features/users/handler/http"
	"github.com/siti-nabila/rest-orc/tests/app/fixtures"
	"github.com/siti-nabila/rest-orc/tests/shared/testutils"
)

const apiV1Prefix = "/api/v1"

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
			Name: "listen message contains application URL and port",
			Run:  listenMessageContainsURLAndPort,
		},
		{
			Name: "client message contains name target and available port",
			Run:  clientMessageContainsConnectionInformation,
		},
		{
			Name: "Fiber liveness endpoint returns healthy status",
			Run:  livenessEndpointReturnsHealthyStatus,
		},
		{
			Name: "ListUsers route rejects a request without authorization",
			Run:  listUsersRouteRejectsMissingAuthorization,
		},
		{
			Name: "Register route reaches public handler without authorization",
			Run:  registerRouteIsPublic,
		},
		{
			Name: "Login route reaches public handler without authorization",
			Run:  loginRouteIsPublic,
		},
		{
			Name: "unversioned users route is not registered",
			Run:  unversionedUsersRouteIsNotRegistered,
		},
		{
			Name: "admin route group does not protect later public routes",
			Run:  adminRouteGroupDoesNotLeakToPublicRoute,
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
	}
}

func clientMessageContainsConnectionInformation(t *testing.T) {
	// Arrange
	scenarios := []struct {
		name     string
		target   string
		expected string
	}{
		{
			name:     "host and port",
			target:   "localhost:9090",
			expected: fixtures.AuthGRPCClientMessage,
		},
		{
			name:     "gRPC resolver target",
			target:   "dns:///auth-service:50051",
			expected: fixtures.ResolverClientMessage,
		},
		{
			name:     "target without TCP port",
			target:   "unix:///tmp/auth.sock",
			expected: fixtures.PortlessClientMessage,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Act
			actual := app.FormatClientMessage("auth_grpc", scenario.target)

			// Assert
			if actual != scenario.expected {
				t.Errorf(
					"client message = %q, want %q",
					actual,
					scenario.expected,
				)
			}
		})
	}
}

func listenMessageContainsURLAndPort(t *testing.T) {
	// Arrange
	httpData := fiber.ListenData{
		AppName: "rest-orc",
		Host:    "0.0.0.0",
		Port:    "8080",
	}
	httpsData := fiber.ListenData{
		AppName: "rest-orc",
		Host:    "0.0.0.0",
		Port:    "8443",
		TLS:     true,
	}

	// Act
	httpMessage := app.FormatListenMessage(httpData)
	httpsMessage := app.FormatListenMessage(httpsData)

	// Assert
	if httpMessage != fixtures.HTTPListenMessage {
		t.Errorf(
			"HTTP listen message = %q, want %q",
			httpMessage,
			fixtures.HTTPListenMessage,
		)
	}
	if httpsMessage != fixtures.HTTPSListenMessage {
		t.Errorf(
			"HTTPS listen message = %q, want %q",
			httpsMessage,
			fixtures.HTTPSListenMessage,
		)
	}
}

func registerRouteIsPublic(t *testing.T) {
	assertMalformedRequestReachesPublicHandler(
		t,
		apiV1Prefix+usershttp.Route+usershttp.RegisterRoute,
	)
}

func loginRouteIsPublic(t *testing.T) {
	assertMalformedRequestReachesPublicHandler(
		t,
		apiV1Prefix+usershttp.Route+usershttp.LoginRoute,
	)
}

func unversionedUsersRouteIsNotRegistered(t *testing.T) {
	// Arrange
	application := newApplication(t, fixtures.ValidConfig())
	request, err := http.NewRequest(http.MethodGet, usershttp.Route, nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	// Act
	response, requestErr := application.HTTPServer.Test(request)

	// Assert
	responseBody := readResponseBody(t, response, requestErr)
	if response.StatusCode != http.StatusNotFound {
		t.Errorf(
			"status code = %d, want %d",
			response.StatusCode,
			http.StatusNotFound,
		)
	}
	if responseBody != fixtures.EndpointNotFoundBody {
		t.Errorf(
			"response body = %q, want %q",
			responseBody,
			fixtures.EndpointNotFoundBody,
		)
	}
}

func assertMalformedRequestReachesPublicHandler(t *testing.T, path string) {
	t.Helper()
	application := newApplication(t, fixtures.ValidConfig())
	request, err := http.NewRequest(
		http.MethodPost,
		path,
		strings.NewReader("{"),
	)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

	response, requestErr := application.HTTPServer.Test(request)
	readResponseBody(t, response, requestErr)
	if response.StatusCode != http.StatusBadRequest {
		t.Errorf(
			"status code = %d, want %d; public handler may be behind auth middleware",
			response.StatusCode,
			http.StatusBadRequest,
		)
	}
}

func adminRouteGroupDoesNotLeakToPublicRoute(t *testing.T) {
	// Arrange
	application := newApplication(t, fixtures.ValidConfig())
	application.HTTPServer.Get("/public-after-admin-groups", func(ctx fiber.Ctx) error {
		return ctx.SendStatus(http.StatusNoContent)
	})
	request, err := http.NewRequest(
		http.MethodGet,
		"/public-after-admin-groups",
		nil,
	)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	// Act
	response, requestErr := application.HTTPServer.Test(request)

	// Assert
	readResponseBody(t, response, requestErr)
	if response.StatusCode != http.StatusNoContent {
		t.Errorf(
			"status code = %d, want %d",
			response.StatusCode,
			http.StatusNoContent,
		)
	}
}

func listUsersRouteRejectsMissingAuthorization(t *testing.T) {
	// Arrange
	application := newApplication(t, fixtures.ValidConfig())
	request, err := http.NewRequest(
		http.MethodGet,
		apiV1Prefix+usershttp.Route,
		nil,
	)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	// Act
	response, requestErr := application.HTTPServer.Test(request)

	// Assert
	readResponseBody(t, response, requestErr)
	if response.StatusCode != http.StatusUnauthorized {
		t.Errorf(
			"status code = %d, want %d",
			response.StatusCode,
			http.StatusUnauthorized,
		)
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
	// if application.BackendClient == nil {
	// 	t.Error("BackendClient = nil")
	// }
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
