package test_scenarios

import (
	"net/http/httptest"
	"testing"

	authhandler "github.com/siti-nabila/rest-orc/internal/features/auth/handler"
	"github.com/siti-nabila/rest-orc/test/app/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc/connectivity"
)

func userRoleRoutes(t *testing.T) {
	app := newApplication(t, fixtures.ValidConfig())
	found := map[string]int{}
	for _, r := range app.HTTPServer.GetRoutes(true) {
		for _, entry := range []struct{ method, path string }{{"GET", "/api/v1/users/:user_id/roles"}, {"POST", "/api/v1/users/:user_id/roles/grant"}, {"POST", "/api/v1/users/:user_id/roles/revoke"}} {
			if r.Method == entry.method && r.Path == entry.path {
				found[r.Method+" "+r.Path]++
			}
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
	for _, target := range []string{"23", "bad", "2147483648", "me"} {
		for _, op := range []string{"list", "grant", "revoke"} {
			for _, mode := range []string{"missing bearer", "preflight", "denied origin", "HEAD"} {
				path := "/api/v1/users/" + target + "/roles"
				method := "GET"
				if op != "list" {
					path += "/" + op
					method = "POST"
				}
				if mode == "HEAD" && op != "list" {
					continue
				}
				req := httptest.NewRequest(method, path, nil)
				want := 401
				switch mode {
				case "preflight":
					req.Header.Set("Origin", "https://app.example.test")
					req.Header.Set("Access-Control-Request-Method", method)
					req.Method = "OPTIONS"
					want = 204
				case "denied origin":
					req.Header.Set("Origin", "https://evil.example.test")
					want = 403
				case "HEAD":
					req.Method = "HEAD"
				}
				res, e := app.HTTPServer.Test(req)
				readResponseBody(t, res, e)
				if res.StatusCode != want || res.Header.Get("Cache-Control") != "no-store" || len(res.Cookies()) != 0 {
					t.Fatal(path, mode, res.StatusCode)
				}
			}
		}
	}
	if app.AuthConnection.Connection().(interface{ GetState() connectivity.State }).GetState() != connectivity.Idle {
		t.Fatal("early RPC")
	}
	standalone, _ := testutils.NewHTTPApp()
	standalone.Use(authhandler.LifecycleNoStore("/edge/v9"))
	for _, path := range []string{"/edge/v9/users/bad/roles", "/edge/v9/USERS/me/ROLES/GRANT/", "/edge/v9/users/23/roles/revoke"} {
		res, e := standalone.Test(httptest.NewRequest("OPTIONS", path, nil))
		testutils.ResponseBody(t, res, e)
		if res.Header.Get("Cache-Control") != "no-store" {
			t.Fatal(path)
		}
	}
}
