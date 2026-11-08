package test_scenarios

import (
	"net/http/httptest"
	"testing"

	authhandler "github.com/siti-nabila/rest-orc/internal/features/auth/handler"
	"github.com/siti-nabila/rest-orc/test/app/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc/connectivity"
)

func roleCatalogRoutes(t *testing.T) {
	app := newApplication(t, fixtures.ValidConfig())
	found := map[string]int{}
	for _, r := range app.HTTPServer.GetRoutes(true) {
		if r.Method == "GET" && (r.Path == "/api/v1/roles" || r.Path == "/api/v1/roles/options" || r.Path == "/api/v1/roles/:role_id") {
			found[r.Path]++
		}
		if r.Path == "/roles" || r.Path == "/roles/options" {
			t.Fatal("alias")
		}
	}
	if len(found) != 3 {
		t.Fatal(found)
	}
	for _, n := range found {
		if n != 1 {
			t.Fatal(found)
		}
	}
	for _, path := range []string{"/api/v1/roles", "/api/v1/roles/options", "/api/v1/roles/10", "/api/v1/roles/bad", "/api/v1/roles/2147483648"} {
		for _, mode := range []string{"GET", "HEAD", "preflight", "denied origin"} {
			req := httptest.NewRequest("GET", path, nil)
			want := 401
			if mode == "HEAD" {
				req.Method = "HEAD"
			}
			if mode == "preflight" {
				req.Method = "OPTIONS"
				req.Header.Set("Origin", "https://app.example.test")
				req.Header.Set("Access-Control-Request-Method", "GET")
				want = 204
			}
			if mode == "denied origin" {
				req.Header.Set("Origin", "https://evil.example.test")
				want = 403
			}
			res, err := app.HTTPServer.Test(req)
			readResponseBody(t, res, err)
			if res.StatusCode != want || res.Header.Get("Cache-Control") != "no-store" || len(res.Cookies()) != 0 {
				t.Fatal(path, mode, res.StatusCode)
			}
		}
	}
	if app.AuthConnection.Connection().(interface{ GetState() connectivity.State }).GetState() != connectivity.Idle {
		t.Fatal("early failure dispatched")
	}
	for _, method := range []string{"GET", "OPTIONS"} {
		req := httptest.NewRequest(method, "/api/v1/users", nil)
		want := 401
		if method == "OPTIONS" {
			req.Header.Set("Origin", "https://app.example.test")
			req.Header.Set("Access-Control-Request-Method", "GET")
			want = 204
		}
		res, err := app.HTTPServer.Test(req)
		readResponseBody(t, res, err)
		if res.StatusCode != want || res.Header.Get("Cache-Control") != "" {
			t.Fatal("users GET cache changed")
		}
	}
	standalone, _ := testutils.NewHTTPApp()
	standalone.Use(authhandler.LifecycleNoStore("/edge/v9"))
	for _, path := range []string{"/edge/v9/roles", "/edge/v9/ROLES/OPTIONS/", "/edge/v9/roles/bad"} {
		res, err := standalone.Test(httptest.NewRequest("OPTIONS", path, nil))
		testutils.ResponseBody(t, res, err)
		if res.Header.Get("Cache-Control") != "no-store" {
			t.Fatal(path)
		}
	}
}
