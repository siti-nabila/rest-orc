package test_scenarios

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	requestauth "github.com/siti-nabila/rest-orc/internal/auth"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/internal/features/users/handler"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

type accessListUC func(context.Context, domain.UserApplicationAccessActor, domain.ListUserApplicationAccessRequest) (domain.ListUserApplicationAccessResponse, error)

func (f accessListUC) Execute(c context.Context, a domain.UserApplicationAccessActor, r domain.ListUserApplicationAccessRequest) (domain.ListUserApplicationAccessResponse, error) {
	return f(c, a, r)
}

type accessGrantUC func(context.Context, domain.UserApplicationAccessActor, domain.GrantApplicationAccessRequest) (domain.ApplicationAccessCommandResponse, error)

func (f accessGrantUC) Execute(c context.Context, a domain.UserApplicationAccessActor, r domain.GrantApplicationAccessRequest) (domain.ApplicationAccessCommandResponse, error) {
	return f(c, a, r)
}

type accessRevokeUC func(context.Context, domain.UserApplicationAccessActor, domain.RevokeApplicationAccessRequest) (domain.ApplicationAccessCommandResponse, error)

func (f accessRevokeUC) Execute(c context.Context, a domain.UserApplicationAccessActor, r domain.RevokeApplicationAccessRequest) (domain.ApplicationAccessCommandResponse, error) {
	return f(c, a, r)
}
func UserApplicationAccessScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "assignment handler sources multiplicity exact messages and standalone browser policy", Run: userApplicationAccessHandler}}
}
func userApplicationAccessHandler(t *testing.T) {
	for _, op := range []string{"list", "grant", "revoke"} {
		for _, mode := range []string{"success", "missing principal", "zero principal", "failure", "empty", "null", "array", "scalar", "malformed", "wrong IDs type", "mixed IDs", "wrong reason", "missing key", "duplicate key", "case duplicate key", "missing version", "duplicate version", "case duplicate version", "empty fields", "query duplicates"} {
			t.Run(op+"/"+mode, func(t *testing.T) {
				app, w := testutils.NewHTTPApp()
				calls := 0
				var original context.Context
				inspect := func(c context.Context, a domain.UserApplicationAccessActor) {
					calls++
					if c != original || a.UserID != 17 || !a.MustChangePassword || !a.Permissions[domain.UserRoleGrantPermission] {
						t.Fatal(a)
					}
				}
				list := accessListUC(func(c context.Context, a domain.UserApplicationAccessActor, r domain.ListUserApplicationAccessRequest) (domain.ListUserApplicationAccessResponse, error) {
					inspect(c, a)
					if op != "list" || r.UserID != "00023" {
						t.Fatal(r)
					}
					if mode == "query duplicates" && len(r.Input["page"]) != 2 {
						t.Fatal(r)
					}
					if mode == "failure" {
						return domain.ListUserApplicationAccessResponse{}, common.ErrForbidden
					}
					return domain.ListUserApplicationAccessResponse{Items: []domain.UserApplicationAccess{{ApplicationID: 5, GrantedAt: "2026-10-20T08:09:10.123456789Z"}}, UserID: 23, Version: "1"}, nil
				})
				inspectCommand := func(c context.Context, a domain.UserApplicationAccessActor, r domain.ApplicationAccessCommandRequest, reason *string) (domain.ApplicationAccessCommandResponse, error) {
					inspect(c, a)
					if r.UserID != "00023" || r.ApplicationCode != "clm" || r.IdempotencyKey != "opaque, key" || r.Version != 9007199254740993 {
						t.Fatal(r)
					}
					if mode == "failure" {
						return domain.ApplicationAccessCommandResponse{}, common.ErrForbidden
					}
					if mode == "empty fields" {
						if reason != nil {
							t.Fatal("reason")
						}
					} else if reason == nil || *reason != "  approved  " {
						t.Fatal(reason)
					}
					return domain.ApplicationAccessCommandResponse{UserID: 23, ApplicationID: 5, ChangedRoleIDs: []uint64{}, Version: "1"}, nil
				}
				grant := accessGrantUC(func(c context.Context, a domain.UserApplicationAccessActor, r domain.GrantApplicationAccessRequest) (domain.ApplicationAccessCommandResponse, error) {
					if op != "grant" {
						t.Fatal("grant")
					}
					if mode != "empty fields" && (len(r.RoleIDs) != 2 || r.RoleIDs[0] != "11") {
						t.Fatal(r)
					}
					return inspectCommand(c, a, r.ApplicationAccessCommandRequest, r.Reason)
				})
				revoke := accessRevokeUC(func(c context.Context, a domain.UserApplicationAccessActor, r domain.RevokeApplicationAccessRequest) (domain.ApplicationAccessCommandResponse, error) {
					if op != "revoke" {
						t.Fatal("revoke")
					}
					return inspectCommand(c, a, r.ApplicationAccessCommandRequest, r.Reason)
				})
				if _, e := handler.NewUserApplicationAccess(nil, grant, revoke, w); e == nil {
					t.Fatal("nil list")
				}
				if _, e := handler.NewUserApplicationAccess(list, nil, revoke, w); e == nil {
					t.Fatal("nil grant")
				}
				if _, e := handler.NewUserApplicationAccess(list, grant, nil, w); e == nil {
					t.Fatal("nil revoke")
				}
				if _, e := handler.NewUserApplicationAccess(list, grant, revoke, nil); e == nil {
					t.Fatal("nil writer")
				}
				h, _ := handler.NewUserApplicationAccess(list, grant, revoke, w)
				h.RegisterRoutes(app, func(c fiber.Ctx) error {
					if mode != "missing principal" {
						id := uint64(17)
						if mode == "zero principal" {
							id = 0
						}
						original = requestauth.WithPrincipal(c.Context(), requestauth.Principal{ID: id, MustChangePassword: true, Permissions: map[string]bool{domain.UserRoleGrantPermission: true}})
						c.SetContext(original)
					}
					return c.Next()
				})
				raw := `{"role_ids":["11","00010"],"reason":"  approved  ","user_id":99,"version":7}`
				switch mode {
				case "empty":
					raw = ""
				case "null":
					raw = "null"
				case "array":
					raw = "[]"
				case "scalar":
					raw = `"text"`
				case "malformed":
					raw = "{"
				case "wrong IDs type":
					raw = `{"role_ids":10,"reason":"  approved  "}`
				case "mixed IDs":
					raw = `{"role_ids":["10",11],"reason":"  approved  "}`
				case "wrong reason":
					raw = `{"role_ids":["10"],"reason":1}`
				case "empty fields":
					raw = "{}"
				}
				method, path := "POST", "/users/00023/applications/clm/"+op
				if op == "list" {
					method = "GET"
					path = "/users/00023/applications"
				}
				path += "?user_id=99&page=1"
				if mode == "query duplicates" {
					path += "&page=1"
				}
				req := httptest.NewRequest(method, path, strings.NewReader(raw))
				req.Header.Set("Version", "9007199254740993")
				req.Header.Set("Idempotency-Key", "opaque, key")
				switch mode {
				case "missing key":
					req.Header.Del("Idempotency-Key")
				case "duplicate key":
					req.Header.Add("Idempotency-Key", "opaque, key")
				case "case duplicate key":
					req.Header["iDeMpOtEnCy-KeY"] = []string{"opaque, key"}
				case "missing version":
					req.Header.Del("Version")
				case "duplicate version":
					req.Header.Add("Version", "9007199254740993")
				case "case duplicate version":
					req.Header["vErSiOn"] = []string{"9007199254740993"}
				}
				want, n := 200, 1
				if mode == "missing principal" || mode == "zero principal" {
					want, n = 401, 0
				} else if mode == "failure" {
					want = 403
				} else if op != "list" {
					switch mode {
					case "wrong IDs type", "mixed IDs":
						if op == "grant" {
							want, n = 400, 0
						}
					case "empty", "null", "array", "scalar", "malformed", "wrong reason", "missing key", "duplicate key", "case duplicate key", "missing version", "duplicate version", "case duplicate version":
						want, n = 400, 0
					}
				}
				res, e := app.Test(req)
				b := testutils.ResponseBody(t, res, e)
				if res.StatusCode != want || calls != n || res.Header.Get("Cache-Control") != "no-store" || len(res.Cookies()) != 0 {
					t.Fatal(res.StatusCode, calls, string(b))
				}
				if want == 200 {
					message := "User application access retrieved successfully."
					if op == "grant" {
						message = "Application access granted successfully."
					}
					if op == "revoke" {
						message = "Application access revoked successfully."
					}
					var env struct{ Data map[string]json.RawMessage }
					if json.Unmarshal([]byte(b), &env) != nil {
						t.Fatal(string(b))
					}
					size := 6
					if op == "list" {
						size = 10
						var items []map[string]json.RawMessage
						_ = json.Unmarshal(env.Data["items"], &items)
						if len(items) != 1 || len(items[0]) != 11 {
							t.Fatal(string(b))
						}
					}
					if len(env.Data) != size {
						t.Fatal(string(b))
					}
					if !strings.Contains(string(b), message) || !strings.Contains(string(b), `"code":"SS"`) {
						t.Fatal(string(b))
					}
				}
			})
		}
	}
}
