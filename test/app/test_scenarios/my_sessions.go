package test_scenarios

import (
	"github.com/siti-nabila/rest-orc/test/app/fixtures"
	"google.golang.org/grpc/connectivity"
	"net/http/httptest"
	"testing"
)

func mySessionRoutes(t *testing.T) {
	application := newApplication(t, fixtures.ValidConfig())
	want := map[string]bool{"GET /api/v1/me/sessions": false, "DELETE /api/v1/me/sessions/:session_id": false, "POST /api/v1/me/sessions/revoke-all": false}
	for _, r := range application.HTTPServer.GetRoutes(true) {
		key := r.Method + " " + r.Path
		if _, ok := want[key]; ok {
			if want[key] {
				t.Fatal("duplicate route")
			}
			want[key] = true
		}
	}
	for path, seen := range want {
		if !seen {
			t.Fatal("missing", path)
		}
	}
	for _, tc := range []struct {
		method, path string
		status       int
	}{{"GET", "/api/v1/me/sessions", 401}, {"DELETE", "/api/v1/me/sessions/00000000-0000-0000-0000-000000000000", 401}, {"POST", "/api/v1/me/sessions/revoke-all", 401}, {"GET", "/me/sessions", 404}, {"GET", "/api/v1/users/me/sessions", 401}, {"GET", "/api/v1/users/17/sessions", 401}} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		res, e := application.HTTPServer.Test(req)
		readResponseBody(t, res, e)
		if res.StatusCode != tc.status {
			t.Fatalf("%s %d", tc.path, res.StatusCode)
		}
		if tc.status == 401 && tc.path[:10] == "/api/v1/me" && res.Header.Get("Cache-Control") != "no-store" {
			t.Fatal("early missing no-store")
		}
	}
	for _, r := range application.HTTPServer.GetRoutes(true) {
		if r.Path == "/me/sessions" || r.Path == "/api/v1/users/me/sessions" || r.Path == "/api/v1/users/:user_id/sessions" {
			t.Fatal("alias registered")
		}
	}
	for _, path := range []string{"/api/v1/me/sessions", "/api/v1/me/sessions/00000000-0000-0000-0000-000000000000", "/api/v1/me/sessions/revoke-all"} {
		for _, origin := range []string{"https://app.example.test", "https://evil.example.test"} {
			req := httptest.NewRequest("OPTIONS", path, nil)
			req.Header.Set("Origin", origin)
			req.Header.Set("Access-Control-Request-Method", "DELETE")
			res, e := application.HTTPServer.Test(req)
			readResponseBody(t, res, e)
			want := 204
			if origin == "https://evil.example.test" {
				want = 403
			}
			if res.StatusCode != want || res.Header.Get("Cache-Control") != "no-store" || len(res.Cookies()) != 0 {
				t.Fatal("preflight", res.StatusCode)
			}
		}
	}
	if application.AuthConnection.Connection().(interface{ GetState() connectivity.State }).GetState() != connectivity.Idle {
		t.Fatal("early denial dialed")
	}
}
