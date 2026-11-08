package test_scenarios

import (
	"net/http/httptest"
	"testing"

	authhandler "github.com/siti-nabila/rest-orc/internal/features/auth/handler"
	"github.com/siti-nabila/rest-orc/test/app/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc/connectivity"
)

func userCreationRoutes(t *testing.T) {
	app := newApplication(t, fixtures.ValidConfig())
	found := 0
	for _, r := range app.HTTPServer.GetRoutes(true) {
		if r.Method == "POST" && r.Path == "/api/v1/users" {
			found++
		}
		if r.Path == "/users" {
			t.Fatal("alias")
		}
	}
	if found != 1 {
		t.Fatal("creation route count", found)
	}
	for _, path := range []string{"/api/v1/users"} {
		for _, mode := range []string{"missing bearer", "allowed preflight", "denied preflight"} {
			req := httptest.NewRequest("POST", path, nil)
			want := 401
			if mode != "missing bearer" {
				req.Method = "OPTIONS"
				req.Header.Set("Origin", "https://app.example.test")
				req.Header.Set("Access-Control-Request-Method", "POST")
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
	res, err := app.HTTPServer.Test(httptest.NewRequest("POST", "/users", nil))
	readResponseBody(t, res, err)
	if res.StatusCode != 404 {
		t.Fatal("unversioned alias")
	}
	// Collection matching follows the supplied prefix and normalized path.
	standalone, _ := testutils.NewHTTPApp()
	standalone.Use(authhandler.LifecycleNoStore("/edge/v9"))
	for _, path := range []string{"/edge/v9/users", "/edge/v9/USERS/"} {
		res, err := standalone.Test(httptest.NewRequest("POST", path, nil))
		testutils.ResponseBody(t, res, err)
		if res.Header.Get("Cache-Control") != "no-store" {
			t.Fatal(path)
		}
	}
	// GET collection and its preflight retain their previous cache behavior.
	for _, method := range []string{"GET", "OPTIONS"} {
		req := httptest.NewRequest(method, "/api/v1/users", nil)
		if method == "OPTIONS" {
			req.Header.Set("Origin", "https://app.example.test")
			req.Header.Set("Access-Control-Request-Method", "GET")
		}
		res, err := app.HTTPServer.Test(req)
		readResponseBody(t, res, err)
		want := 401
		if method == "OPTIONS" {
			want = 204
		}
		if res.StatusCode != want || res.Header.Get("Cache-Control") != "" || len(res.Cookies()) != 0 {
			t.Fatal("GET cache changed")
		}
	}
}
