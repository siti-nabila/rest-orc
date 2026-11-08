package test_scenarios

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	requestauth "github.com/siti-nabila/rest-orc/internal/auth"
	"github.com/siti-nabila/rest-orc/internal/features/applications/domain"
	"github.com/siti-nabila/rest-orc/internal/features/applications/handler"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

type applicationListUC func(context.Context, domain.ApplicationActor, domain.ApplicationListInput) (domain.ApplicationPage[domain.Application], error)

func (f applicationListUC) Execute(c context.Context, a domain.ApplicationActor, in domain.ApplicationListInput) (domain.ApplicationPage[domain.Application], error) {
	return f(c, a, in)
}
func All() []testutils.Scenario {
	return []testutils.Scenario{{Name: "application handler exact views query sources context constructors and standalone browser boundary", Run: views}}
}
func views(t *testing.T) {
	for _, view := range []string{"list", "options"} {
		for _, mode := range []string{"success", "empty", "empty name", "historical code", "failure", "missing principal", "zero principal"} {
			t.Run(view+"/"+mode, func(t *testing.T) {
				app, w := testutils.NewHTTPApp()
				calls := 0
				var original context.Context
				code, name := "clm", " Legacy NAME\té\r\n "
				if mode == "empty name" {
					name = ""
				}
				if mode == "historical code" {
					code = " legacy-code\té "
				}
				uc := applicationListUC(func(c context.Context, a domain.ApplicationActor, in domain.ApplicationListInput) (domain.ApplicationPage[domain.Application], error) {
					calls++
					if c != original || a.UserID != 17 || !a.MustChangePassword || !a.Permissions[domain.ReadPermission] || !reflect.DeepEqual(in["page"], []string{"1", "2"}) || in["user_id"][0] != "99" {
						t.Fatal(a, in)
					}
					if mode == "failure" {
						return domain.ApplicationPage[domain.Application]{}, common.ErrForbidden
					}
					items := []domain.Application{{ID: 5, Code: code, Name: name, Status: "DISABLED"}}
					if mode == "empty" {
						items = []domain.Application{}
					}
					return domain.ApplicationPage[domain.Application]{Items: items, Total: 1 << 32, Page: 99, Limit: 20, TotalPages: 3, HasPrev: true, NextCursor: "next+/="}, nil
				})
				if _, e := handler.New(nil, w); e == nil {
					t.Fatal("nil usecase")
				}
				if _, e := handler.New(uc, nil); e == nil {
					t.Fatal("nil writer")
				}
				h, _ := handler.New(uc, w)
				h.RegisterRoutes(app, func(c fiber.Ctx) error {
					if mode != "missing principal" {
						id := uint64(17)
						if mode == "zero principal" {
							id = 0
						}
						original = requestauth.WithPrincipal(c.Context(), requestauth.Principal{ID: id, MustChangePassword: true, Permissions: map[string]bool{domain.ReadPermission: true}})
						c.SetContext(original)
					}
					return c.Next()
				})
				path := "/applications"
				if view == "options" {
					path += "/options"
				}
				req := httptest.NewRequest("GET", path+"?page=1&page=2&user_id=99", strings.NewReader("{"))
				req.Header["Version"] = []string{"bad", "bad"}
				req.Header["Idempotency-Key"] = []string{"bad", "bad"}
				res, e := app.Test(req)
				b := testutils.ResponseBody(t, res, e)
				want, n := 200, 1
				if mode == "failure" {
					want = 403
				}
				if mode == "missing principal" || mode == "zero principal" {
					want, n = 401, 0
				}
				if res.StatusCode != want || calls != n || res.Header.Get("Cache-Control") != "no-store" || len(res.Cookies()) != 0 {
					t.Fatal(res.StatusCode, b, calls)
				}
				if want == 200 {
					var env struct {
						Code, Message string
						Data          map[string]json.RawMessage
					}
					if json.Unmarshal([]byte(b), &env) != nil || env.Code != "SS" || len(env.Data) != 8 || string(env.Data["total"]) != "4294967296" || string(env.Data["page"]) != "99" || string(env.Data["next_cursor"]) != `"next+/="` {
						t.Fatal(b)
					}
					message := "Applications retrieved successfully."
					if view == "options" {
						message = "Application options retrieved successfully."
					}
					if env.Message != message {
						t.Fatal(b)
					}
					var items []map[string]json.RawMessage
					_ = json.Unmarshal(env.Data["items"], &items)
					if mode == "empty" {
						if string(env.Data["items"]) != "[]" {
							t.Fatal(b)
						}
					} else if view == "options" {
						v, _ := json.Marshal(code)
						label, _ := json.Marshal(code + " - " + name)
						if len(items) != 1 || len(items[0]) != 2 || string(items[0]["value"]) != string(v) || string(items[0]["label"]) != string(label) {
							t.Fatal(b)
						}
					} else if len(items[0]) != 5 {
						t.Fatal(b)
					}
				}
			})
		}
	}
}
