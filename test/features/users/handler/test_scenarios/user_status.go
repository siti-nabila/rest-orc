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

type statusUseCase func(context.Context, domain.UserStatusActor, domain.ChangeUserStatusRequest) (domain.ChangeUserStatusResponse, error)

func (f statusUseCase) Execute(c context.Context, a domain.UserStatusActor, r domain.ChangeUserStatusRequest) (domain.ChangeUserStatusResponse, error) {
	return f(c, a, r)
}
func UserStatusScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "status handler sources context constructors object body and header multiplicity", Run: statusHandler}}
}
func statusHandler(t *testing.T) {
	for _, mode := range []string{"success", "missing principal", "zero principal", "empty body", "null", "array", "scalar", "malformed", "wrong status type", "wrong reason type", "missing fields", "null fields", "missing key", "duplicate key", "case duplicate key", "missing version", "duplicate version", "case duplicate version", "invalid path"} {
		t.Run(mode, func(t *testing.T) {
			app, writer := testutils.NewHTTPApp()
			calls := 0
			var requestContext context.Context
			uc := statusUseCase(func(c context.Context, a domain.UserStatusActor, r domain.ChangeUserStatusRequest) (domain.ChangeUserStatusResponse, error) {
				calls++
				if c != requestContext || a.UserID != 17 || !a.MustChangePassword || !a.Permissions[domain.UserStatusUpdatePermission] || r.UserID != 23 || r.Version != 9007199254740993 || r.IdempotencyKey != "opaque, key" {
					t.Fatal(a, r)
				}
				if mode == "missing fields" || mode == "null fields" {
					if r.Status != nil || r.Reason != nil {
						t.Fatal("presence")
					}
				} else if r.Status == nil || *r.Status != "ACTIVE" || r.Reason == nil || *r.Reason != "  approved  " {
					t.Fatal(r)
				}
				return domain.ChangeUserStatusResponse{UserID: 23, Status: "ACTIVE", Version: "1", Changed: false}, nil
			})
			if _, err := handler.NewUserStatus(nil, writer); err == nil {
				t.Fatal("nil usecase")
			}
			if _, err := handler.NewUserStatus(uc, nil); err == nil {
				t.Fatal("nil writer")
			}
			h, _ := handler.NewUserStatus(uc, writer)
			h.RegisterRoutes(app, func(c fiber.Ctx) error {
				if mode != "missing principal" {
					id := uint64(17)
					if mode == "zero principal" {
						id = 0
					}
					requestContext = requestauth.WithPrincipal(c.Context(), requestauth.Principal{ID: id, Version: 1, MustChangePassword: true, Permissions: map[string]bool{domain.UserStatusUpdatePermission: true}})
					c.SetContext(requestContext)
				}
				return c.Next()
			})
			raw := `{"status":"ACTIVE","reason":"  approved  ","user_id":99,"version":7,"permissions":{"iam.user.status.update":true}}`
			want := 400
			expectedCalls := 0
			switch mode {
			case "success":
				want = 200
				expectedCalls = 1
			case "missing principal", "zero principal":
				want = 401
			case "empty body":
				raw = ""
			case "null":
				raw = "null"
			case "array":
				raw = "[]"
			case "scalar":
				raw = `"text"`
			case "malformed":
				raw = "{"
			case "wrong status type":
				raw = `{"status":1,"reason":"approved"}`
			case "wrong reason type":
				raw = `{"status":"ACTIVE","reason":true}`
			case "missing fields":
				raw = "{}"
				want = 200
				expectedCalls = 1
			case "null fields":
				raw = `{"status":null,"reason":null}`
				want = 200
				expectedCalls = 1
			}
			path := "/users/00023/status?user_id=99&status=DEACTIVATED&reason=forged&version=99"
			if mode == "invalid path" {
				path = "/users/me/status"
			}
			req := httptest.NewRequest("PATCH", path, strings.NewReader(raw))
			req.Header.Set("Idempotency-Key", "opaque, key")
			req.Header.Set("Version", "9007199254740993")
			switch mode {
			case "missing key":
				req.Header.Del("Idempotency-Key")
			case "duplicate key":
				req.Header.Add("Idempotency-Key", "opaque, key")
			case "case duplicate key":
				req.Header["idempotency-key"] = []string{"other"}
			case "missing version":
				req.Header.Del("Version")
			case "duplicate version":
				req.Header.Add("Version", "9007199254740993")
			case "case duplicate version":
				req.Header["version"] = []string{"1"}
			}
			res, err := app.Test(req)
			body := testutils.ResponseBody(t, res, err)
			if res.StatusCode != want || calls != expectedCalls || res.Header.Get("Cache-Control") != "no-store" || len(res.Cookies()) != 0 {
				t.Fatal(res.StatusCode, want, body, calls)
			}
			if want == 200 && body != `{"code":"SS","message":"User status updated successfully.","data":{"user_id":23,"status":"ACTIVE","version":"1","changed":false}}` {
				t.Fatal(body)
			}
		})
	}
}
