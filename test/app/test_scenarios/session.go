package test_scenarios

import (
	"github.com/siti-nabila/rest-orc/test/app/fixtures"
	"google.golang.org/grpc/connectivity"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func sessionRoutesAndCORS(t *testing.T) {
	application := newApplication(t, fixtures.ValidConfig())
	origin := "https://app.example.test"
	routes := map[string]bool{}
	for _, route := range application.HTTPServer.GetRoutes(true) {
		if strings.Contains(route.Path, "/auth/") && (route.Method == "GET" || route.Method == "POST") {
			routes[route.Method+" "+route.Path] = true
		}
	}
	if len(routes) != 3 || !routes["POST /api/v1/auth/refresh"] || !routes["POST /api/v1/auth/logout"] || !routes["GET /api/v1/auth/csrf"] {
		t.Fatalf("unexpected session routes %v", routes)
	}
	for _, tc := range []struct {
		method, path, origin string
		status               int
		preflight            bool
	}{
		{"POST", "/api/v1/auth/refresh", origin, 401, false}, {"POST", "/api/v1/auth/logout", origin, 401, false}, {"GET", "/api/v1/auth/csrf", origin, 401, false},
		{"POST", "/api/v1/users/login", "", 403, false}, {"POST", "/api/v1/auth/refresh", "null", 403, false},
		{"POST", "/api/v1/auth/refresh/", "null", 403, false}, {"POST", "/api/v1/auth/refresh", origin + ".evil", 403, false},
		{"OPTIONS", "/api/v1/auth/refresh", origin, 204, true}, {"OPTIONS", "/api/v1/me/profile", origin, 204, true}, {"OPTIONS", "/api/v1/auth/refresh", "https://evil.test", 403, true},
		{"POST", "/auth/refresh", origin, 404, false},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		if tc.origin != "" {
			req.Header.Set("Origin", tc.origin)
		}
		if tc.preflight {
			req.Header.Set("Access-Control-Request-Method", "POST")
			req.Header.Set("Access-Control-Request-Headers", "Content-Type, X-CSRF-Token, Authorization")
		}
		res, err := application.HTTPServer.Test(req)
		readResponseBody(t, res, err)
		if res.StatusCode != tc.status {
			t.Fatalf("%s %s status %d want %d", tc.method, tc.path, res.StatusCode, tc.status)
		}
		if strings.HasPrefix(tc.path, "/api/v1/auth/") && res.Header.Get("Cache-Control") != "no-store" {
			t.Fatal("early rejection/preflight cached")
		}
		if tc.status == 204 {
			if res.Header.Get("Access-Control-Allow-Origin") != origin || res.Header.Get("Access-Control-Allow-Credentials") != "true" || !strings.Contains(strings.ToLower(res.Header.Get("Vary")), "origin") {
				t.Fatal("credentialed CORS incorrect")
			}
		}
		if tc.status == 403 && res.Header.Get("Access-Control-Allow-Origin") != "" {
			t.Fatal("denied Origin reflected")
		}
		if len(res.Cookies()) != 0 {
			t.Fatal("preflight/error changed cookies")
		}
	}
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/refresh", nil)
	req.Header["Origin"] = []string{origin, origin}
	req.Header.Set("Access-Control-Request-Method", "POST")
	res, err := application.HTTPServer.Test(req)
	readResponseBody(t, res, err)
	if res.StatusCode != 403 {
		t.Fatal("duplicate Origin accepted")
	}
	state := application.AuthConnection.Connection().(interface{ GetState() connectivity.State })
	if state.GetState() != connectivity.Idle {
		t.Fatal("local checks/preflight dispatched RPC")
	}
}
