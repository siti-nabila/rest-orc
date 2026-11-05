package test_scenarios

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	requestauth "github.com/siti-nabila/rest-orc/internal/auth"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	handler "github.com/siti-nabila/rest-orc/internal/features/users/handler"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

type adminList func(context.Context, domain.AdminSessionActor, uint64, domain.AdminSessionsQuery) (domain.AdminSessionsPage, error)

func (f adminList) Execute(c context.Context, a domain.AdminSessionActor, id uint64, q domain.AdminSessionsQuery) (domain.AdminSessionsPage, error) {
	return f(c, a, id, q)
}

type adminRevoke func(context.Context, domain.AdminSessionActor, domain.RevokeUserSessionRequest) (domain.RevokeUserSessionResponse, error)

func (f adminRevoke) Execute(c context.Context, a domain.AdminSessionActor, r domain.RevokeUserSessionRequest) (domain.RevokeUserSessionResponse, error) {
	return f(c, a, r)
}

type adminAll func(context.Context, domain.AdminSessionActor, domain.RevokeAllUserSessionsRequest) (domain.RevokeAllUserSessionsResponse, error)

func (f adminAll) Execute(c context.Context, a domain.AdminSessionActor, r domain.RevokeAllUserSessionsRequest) (domain.RevokeAllUserSessionsResponse, error) {
	return f(c, a, r)
}

func AdminSessionScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "admin handler authoritative sources duplicate headers and exact envelopes", Run: adminHandler}}
}

func adminHandler(t *testing.T) {
	for _, tc := range []struct {
		name, method, path, key, version, body  string
		duplicateKey, duplicateVersion, missing bool
		status                                  int
	}{
		{name: "list decoded cursor", method: "GET", path: "/users/00023/sessions?last_id=%252F%2B%20", key: "ignored", version: "invalid", body: "{", status: 200},
		{name: "single opaque key and malformed ignored body", method: "DELETE", path: "/users/23/sessions/00000000-0000-0000-0000-000000000000?user_id=99&version=99", key: "a, b", body: "{", status: 200},
		{name: "all precise target version", method: "POST", path: "/users/23/sessions/revoke-all?version=99&user_id=99", key: "a, b", version: "09007199254740993", body: `{"version":99,"user_id":99}`, status: 200},
		{name: "empty body", method: "POST", path: "/users/23/sessions/revoke-all", key: "a, b", version: "9007199254740993", status: 200},
		{name: "missing key", method: "DELETE", path: "/users/23/sessions/id", status: 400},
		{name: "duplicate key", method: "DELETE", path: "/users/23/sessions/id", key: "a, b", duplicateKey: true, status: 400},
		{name: "blank key", method: "DELETE", path: "/users/23/sessions/id", key: "   ", status: 400},
		{name: "unicode key", method: "DELETE", path: "/users/23/sessions/id", key: "é-secret", status: 400},
		{name: "oversized key", method: "DELETE", path: "/users/23/sessions/id", key: strings.Repeat("x", 201), status: 400},
		{name: "version missing", method: "POST", path: "/users/23/sessions/revoke-all?version=9", key: "a, b", body: `{"version":9}`, status: 400},
		{name: "version duplicate", method: "POST", path: "/users/23/sessions/revoke-all", key: "a, b", version: "1", duplicateVersion: true, status: 400},
		{name: "version overflow", method: "POST", path: "/users/23/sessions/revoke-all", key: "a, b", version: "9223372036854775808", status: 400},
		{name: "target me invalid", method: "GET", path: "/users/me/sessions", status: 400},
		{name: "duplicate pagination", method: "GET", path: "/users/23/sessions?page=1&page=1", status: 400},
		{name: "unsupported control", method: "GET", path: "/users/23/sessions?sort=", status: 400},
		{name: "missing principal", method: "GET", path: "/users/23/sessions", missing: true, status: 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app, writer := testutils.NewHTTPApp()
			calls := 0
			var requestContext context.Context
			check := func(c context.Context, a domain.AdminSessionActor) {
				t.Helper()
				calls++
				if c != requestContext || a.UserID != 17 || !a.Permissions[domain.SessionReadPermission] {
					t.Fatal("authority/context changed")
				}
			}
			l := adminList(func(c context.Context, a domain.AdminSessionActor, id uint64, q domain.AdminSessionsQuery) (domain.AdminSessionsPage, error) {
				check(c, a)
				if id != 23 || q.Page != 1 || q.Limit != 20 || q.LastID != "%2F+ " {
					t.Fatal(id, q)
				}
				return domain.AdminSessionsPage{Items: []domain.AdminSession{}, Total: 2147483648, Page: 1, Limit: 20}, nil
			})
			r := adminRevoke(func(c context.Context, a domain.AdminSessionActor, r domain.RevokeUserSessionRequest) (domain.RevokeUserSessionResponse, error) {
				check(c, a)
				if r.UserID != 23 || r.IdempotencyKey != "a, b" || r.SessionID != "00000000-0000-0000-0000-000000000000" {
					t.Fatal(r)
				}
				return domain.RevokeUserSessionResponse{}, nil
			})
			all := adminAll(func(c context.Context, a domain.AdminSessionActor, r domain.RevokeAllUserSessionsRequest) (domain.RevokeAllUserSessionsResponse, error) {
				check(c, a)
				if r.UserID != 23 || r.IdempotencyKey != "a, b" || r.Version != 9007199254740993 {
					t.Fatal(r)
				}
				return domain.RevokeAllUserSessionsResponse{Version: "9007199254740993"}, nil
			})
			if _, err := handler.NewAdminSessions(nil, r, all, writer); err == nil {
				t.Fatal("nil list")
			}
			if _, err := handler.NewAdminSessions(l, nil, all, writer); err == nil {
				t.Fatal("nil revoke")
			}
			if _, err := handler.NewAdminSessions(l, r, nil, writer); err == nil {
				t.Fatal("nil all")
			}
			if _, err := handler.NewAdminSessions(l, r, all, nil); err == nil {
				t.Fatal("nil writer")
			}
			h, err := handler.NewAdminSessions(l, r, all, writer)
			if err != nil {
				t.Fatal(err)
			}
			h.RegisterRoutes(app, func(c fiber.Ctx) error {
				if !tc.missing {
					requestContext = requestauth.WithPrincipal(c.Context(), requestauth.Principal{ID: 17, Version: 999, Permissions: map[string]bool{domain.SessionReadPermission: true}})
					c.SetContext(requestContext)
				}
				return c.Next()
			})
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			if tc.key != "" {
				req.Header["iDeMpOtEnCy-KeY"] = []string{tc.key}
				if tc.duplicateKey {
					req.Header["Idempotency-Key"] = []string{tc.key}
				}
			}
			if tc.version != "" {
				req.Header["vErSiOn"] = []string{tc.version}
				if tc.duplicateVersion {
					req.Header["Version"] = []string{tc.version}
				}
			}
			res, err := app.Test(req)
			body := testutils.ResponseBody(t, res, err)
			wantCalls := 0
			if tc.status == 200 {
				wantCalls = 1
			}
			if res.StatusCode != tc.status || calls != wantCalls || res.Header.Get("Cache-Control") != "no-store" || len(res.Cookies()) != 0 {
				t.Fatalf("%d %s calls=%d", res.StatusCode, body, calls)
			}
			if tc.status == 200 {
				switch tc.method {
				case "DELETE":
					if body != `{"code":"SS","message":"User session revoked successfully.","data":{"changed":false}}` {
						t.Fatal(body)
					}
				case "POST":
					if body != `{"code":"SS","message":"All user sessions revoked successfully.","data":{"changed":false,"version":"9007199254740993"}}` {
						t.Fatal(body)
					}
				case "GET":
					if body != `{"code":"SS","message":"User sessions retrieved successfully.","data":{"items":[],"total":2147483648,"page":1,"limit":20,"total_pages":0,"has_next":false,"has_prev":false,"next_cursor":""}}` {
						t.Fatal(body)
					}
				}
			} else if strings.Contains(body, "secret") || strings.Contains(body, "login_required") {
				t.Fatal(body)
			}
		})
	}
}
