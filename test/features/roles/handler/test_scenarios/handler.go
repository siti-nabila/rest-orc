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
	"github.com/siti-nabila/rest-orc/internal/features/roles/domain"
	"github.com/siti-nabila/rest-orc/internal/features/roles/handler"
	shared "github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

type listCase func(context.Context, domain.Actor, domain.ListInput) (domain.Page[domain.Role], error)

func (f listCase) Execute(c context.Context, a domain.Actor, r domain.ListInput) (domain.Page[domain.Role], error) {
	return f(c, a, r)
}

type getCase func(context.Context, domain.Actor, string) (domain.Detail, error)

func (f getCase) Execute(c context.Context, a domain.Actor, r string) (domain.Detail, error) {
	return f(c, a, r)
}
func All() []testutils.Scenario {
	return []testutils.Scenario{{Name: "role HTTP sources duplicate preservation options labels and exact envelopes", Run: routes}}
}
func routes(t *testing.T) {
	for _, view := range []string{"list", "options", "detail"} {
		for _, mode := range []string{"success", "null description", "empty description", "whitespace description", "empty page", "large total", "duplicate query", "missing principal", "zero principal", "forbidden", "internal"} {
			t.Run(view+"/"+mode, func(t *testing.T) {
				app, writer := testutils.NewHTTPApp()
				var ctx context.Context
				calls := 0
				var role domain.Role
				_ = json.Unmarshal([]byte(shared.RoleJSON), &role)
				desc := ""
				if mode == "null description" {
					role.Description = nil
				}
				if mode == "empty description" {
					role.Description = &desc
				}
				if mode == "whitespace description" {
					desc = " \r\n "
					role.Description = &desc
				}
				page := domain.Page[domain.Role]{Items: []domain.Role{role}, Total: 1, Page: 2, Limit: 20, TotalPages: 1, HasPrev: true, NextCursor: "opaque+/=%"}
				if mode == "empty page" {
					page.Items = []domain.Role{}
					page.Total = 7
				}
				if mode == "large total" {
					page.Total = 9223372036854775807
				}
				var permissions []domain.Permission
				_ = json.Unmarshal([]byte(shared.RolePermissionsJSON), &permissions)
				detail := domain.Detail{Role: role, Permissions: permissions}
				inspect := func(c context.Context, a domain.Actor) error {
					calls++
					if c != ctx || a.UserID != 17 || !a.MustChangePassword || !a.Permissions[domain.ReadPermission] {
						t.Fatal("authority/context")
					}
					if mode == "forbidden" {
						return common.ErrForbidden
					}
					if mode == "internal" {
						return common.ErrInternalServerError
					}
					return nil
				}
				list := listCase(func(c context.Context, a domain.Actor, r domain.ListInput) (domain.Page[domain.Role], error) {
					if view == "detail" {
						t.Fatal("wrong method")
					}
					n := 1
					if mode == "duplicate query" {
						n = 2
					}
					if len(r["page"]) != n || r["page"][0] != "0002" || r["keyword"][0] != "legal\treviewer" || r["last_id"][0] != "opaque+/=%" || r["namespace"][0] != "forged" {
						t.Fatal("query sources")
					}
					return page, inspect(c, a)
				})
				get := getCase(func(c context.Context, a domain.Actor, r string) (domain.Detail, error) {
					if view != "detail" || r != "00010" {
						t.Fatal("target source", r)
					}
					return detail, inspect(c, a)
				})
				if _, err := handler.New(nil, get, writer); err == nil {
					t.Fatal("nil list")
				}
				if _, err := handler.New(list, nil, writer); err == nil {
					t.Fatal("nil get")
				}
				if _, err := handler.New(list, get, nil); err == nil {
					t.Fatal("nil writer")
				}
				h, _ := handler.New(list, get, writer)
				h.RegisterRoutes(app, func(c fiber.Ctx) error {
					if mode != "missing principal" {
						id := uint64(17)
						if mode == "zero principal" {
							id = 0
						}
						ctx = requestauth.WithPrincipal(c.Context(), requestauth.Principal{ID: id, MustChangePassword: true, Permissions: map[string]bool{domain.ReadPermission: true}})
						c.SetContext(ctx)
					}
					return c.Next()
				})
				path := "/roles"
				message := "Roles retrieved successfully."
				var data any = page
				if view == "options" {
					path += "/options"
					message = "Role options retrieved successfully."
					label := role.Name
					if role.Description != nil && *role.Description != "" {
						label = *role.Description
					}
					items := []domain.Option{{Value: "10", Label: role.Code + " - " + label, Code: role.Code, RoleCode: "3"}}
					if mode == "empty page" {
						items = []domain.Option{}
					}
					data = domain.Page[domain.Option]{Items: items, Total: page.Total, Page: page.Page, Limit: page.Limit, TotalPages: page.TotalPages, HasPrev: true, NextCursor: page.NextCursor}
				}
				if view == "detail" {
					path += "/00010"
					message = "Role retrieved successfully."
					data = detail
				}
				path += "?page=0002&limit=020&keyword=legal%09reviewer&last_id=opaque%2B%2F%3D%25&application_code=clm&statuses=ACTIVE%2CDISABLED&user_id=99&role_id=99&namespace=forged"
				if mode == "duplicate query" {
					path += "&page=2"
				}
				req := httptest.NewRequest("GET", path, strings.NewReader(`{"role_id":999,"permissions":{"iam.role.read":false}}`))
				req.Header["Version"] = []string{"bad", "bad"}
				req.Header["Idempotency-Key"] = []string{"bad", "bad"}
				res, err := app.Test(req)
				body := testutils.ResponseBody(t, res, err)
				want, n := 200, 1
				if mode == "missing principal" || mode == "zero principal" {
					want = 401
					n = 0
				}
				if mode == "forbidden" {
					want = 403
				}
				if mode == "internal" {
					want = 500
				}
				if res.StatusCode != want || calls != n || res.Header.Get("Cache-Control") != "no-store" || len(res.Cookies()) != 0 {
					t.Fatal(view, mode, res.StatusCode, body, calls)
				}
				if want == 200 {
					raw, _ := json.Marshal(data)
					expected := `{"code":"SS","message":"` + message + `","data":` + string(raw) + `}`
					if body != expected {
						t.Fatal(body, expected)
					}
				} else if !strings.Contains(body, `"data":[]`) {
					t.Fatal(body)
				}
			})
		}
	}
}
