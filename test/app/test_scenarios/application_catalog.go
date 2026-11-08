package test_scenarios

import (
	"net/http/httptest"
	"strings"
	"testing"

	authhandler "github.com/siti-nabila/rest-orc/internal/features/auth/handler"
	"github.com/siti-nabila/rest-orc/test/app/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc/connectivity"
)

func applicationCatalogRoutes(t *testing.T) {
	app := newApplication(t, fixtures.ValidConfig())
	found := map[string]int{}
	for _, r := range app.HTTPServer.GetRoutes(true) {
		if strings.HasPrefix(r.Path, "/api/v1/applications") {
			if (r.Method != "GET" && r.Method != "HEAD") || (r.Path != "/api/v1/applications" && r.Path != "/api/v1/applications/options") {
				t.Fatal("extra route", r.Method, r.Path)
			}
			if r.Method == "GET" {
				found[r.Path]++
			}
		}
	}
	if len(found) != 2 {
		t.Fatal(found)
	}
	for _, n := range found {
		if n != 1 {
			t.Fatal(found)
		}
	}
	for _, path := range []string{"/api/v1/applications", "/api/v1/applications/options"} {
		for _, mode := range []string{"missing bearer", "HEAD", "preflight", "denied origin"} {
			req := httptest.NewRequest("GET", path, nil)
			want := 401
			switch mode {
			case "HEAD":
				req.Method = "HEAD"
			case "preflight":
				req.Method = "OPTIONS"
				req.Header.Set("Origin", "https://app.example.test")
				req.Header.Set("Access-Control-Request-Method", "GET")
				want = 204
			case "denied origin":
				req.Header.Set("Origin", "https://evil.test")
				want = 403
			}
			res, e := app.HTTPServer.Test(req)
			readResponseBody(t, res, e)
			if res.StatusCode != want || res.Header.Get("Cache-Control") != "no-store" || len(res.Cookies()) != 0 {
				t.Fatal(path, mode, res.StatusCode)
			}
		}
	}
	for _, path := range []string{"/applications", "/api/v1/applications/5", "/api/v1/applications/me"} {
		res, e := app.HTTPServer.Test(httptest.NewRequest("GET", path, nil))
		readResponseBody(t, res, e)
		if res.StatusCode == 200 {
			t.Fatal("alias", path)
		}
	}
	if app.AuthConnection.Connection().(interface{ GetState() connectivity.State }).GetState() != connectivity.Idle {
		t.Fatal("early RPC")
	}
	standalone, _ := testutils.NewHTTPApp()
	standalone.Use(authhandler.LifecycleNoStore("/edge/v9"))
	for _, path := range []string{"/edge/v9/applications", "/edge/v9/APPLICATIONS/OPTIONS/"} {
		res, e := standalone.Test(httptest.NewRequest("OPTIONS", path, nil))
		testutils.ResponseBody(t, res, e)
		if res.Header.Get("Cache-Control") != "no-store" {
			t.Fatal(path)
		}
	}
}
