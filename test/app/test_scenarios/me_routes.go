package test_scenarios

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/siti-nabila/rest-orc/test/app/fixtures"
	"google.golang.org/grpc/connectivity"
)

func meRouteRejectsMissingAuthorization(t *testing.T) {
	application := newApplication(t, fixtures.ValidConfig())
	for _, bearer := range []string{"", "   "} {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
		request.Header.Set("Authorization", bearer)
		response, err := application.HTTPServer.Test(request)
		body := readResponseBody(t, response, err)
		if response.StatusCode != http.StatusUnauthorized {
			t.Fatalf("Me status=%d, want 401; body=%s", response.StatusCode, body)
		}
	}
	state := application.AuthConnection.Connection().(interface{ GetState() connectivity.State })
	if state.GetState() != connectivity.Idle {
		t.Fatal("missing authorization dispatched a downstream RPC")
	}
}

func meRouteHasNoAliases(t *testing.T) {
	application := newApplication(t, fixtures.ValidConfig())
	found := false
	for _, route := range application.HTTPServer.GetRoutes() {
		if route.Method != http.MethodGet {
			continue
		}
		switch route.Path {
		case "/api/v1/me":
			found = true
		case "/me", "/api/v1/users/me", "/api/v1/auth/me":
			t.Errorf("unexpected Me alias %s", route.Path)
		}
	}
	if !found {
		t.Fatal("GET /api/v1/me is not registered")
	}
}
