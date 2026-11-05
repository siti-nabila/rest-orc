package test_scenarios

import (
	"net/http/httptest"
	"testing"

	authhandler "github.com/siti-nabila/rest-orc/internal/features/auth/handler"
	"github.com/siti-nabila/rest-orc/test/app/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc/connectivity"
)

func userStatusRoutes(t *testing.T) {
	app := newApplication(t, fixtures.ValidConfig())
	found := 0
	for _, r := range app.HTTPServer.GetRoutes(true) {
		if r.Method == "PATCH" && r.Path == "/api/v1/users/:user_id/status" {
			found++
		}
		if r.Path == "/users/:user_id/status" || r.Path == "/api/v1/users/me/status" {
			t.Fatal("alias")
		}
	}
	if found != 1 {
		t.Fatal("detail route count", found)
	}
	for _, path := range []string{"/api/v1/users/23/status", "/api/v1/users/me/status", "/api/v1/users/2147483648/status", "/api/v1/users/bad/status"} {
		for _, mode := range []string{"missing bearer", "allowed preflight", "denied preflight"} {
			req := httptest.NewRequest("PATCH", path, nil)
			want := 401
			if mode != "missing bearer" {
				req.Method = "OPTIONS"
				req.Header.Set("Origin", "https://app.example.test")
				req.Header.Set("Access-Control-Request-Method", "PATCH")
				want = 204
				if mode == "denied preflight" {
					req.Header.Set("Origin", "https://evil.example.test")
					want = 403
				}
			}
			res, err := app.HTTPServer.Test(req)
			readResponseBody(t, res, err)
			if res.StatusCode != want || res.Header.Get("Cache-Control") != "no-store" || len(res.Cookies()) != 0 {
				t.Fatal(mode, path, res.StatusCode)
			}
		}
	}
	if app.AuthConnection.Connection().(interface{ GetState() connectivity.State }).GetState() != connectivity.Idle {
		t.Fatal("early failure dispatched")
	}
	res, err := app.HTTPServer.Test(httptest.NewRequest("PATCH", "/users/23/status", nil))
	readResponseBody(t, res, err)
	if res.StatusCode != 404 {
		t.Fatal("unversioned alias")
	}
	// The matcher follows its supplied API prefix and does not depend on a valid ID.
	standalone, _ := testutils.NewHTTPApp()
	standalone.Use(authhandler.LifecycleNoStore("/edge/v9"))
	for _, path := range []string{"/edge/v9/users/23/status", "/edge/v9/users/me/status", "/edge/v9/users/+1/status"} {
		res, err := standalone.Test(httptest.NewRequest("OPTIONS", path, nil))
		testutils.ResponseBody(t, res, err)
		if res.Header.Get("Cache-Control") != "no-store" {
			t.Fatal(path)
		}
	}
}
