package test_scenarios

import (
	"context"
	"fmt"
	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	requestauth "github.com/siti-nabila/rest-orc/internal/auth"
	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	handler "github.com/siti-nabila/rest-orc/internal/features/auth/handler"
	"github.com/siti-nabila/rest-orc/test/features/auth/handler/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func MySessionsScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "self-session handler parses once and preserves context actor and safe flags", Run: mySessionsHandler}}
}
func mySessionsHandler(t *testing.T) {
	for _, op := range []string{"list", "one", "all"} {
		for _, changed := range []bool{false, true} {
			for _, login := range []bool{false, true} {
				for _, fail := range []bool{false, true} {
					for _, incoming := range []bool{false, true} {
						app, writer := testutils.NewHTTPApp()
						browser := fixtures.NewBrowserPolicy(t)
						calls := 0
						var requestCtx context.Context
						check := func(c context.Context, a domain.SessionActor) {
							calls++
							if c != requestCtx || a.UserID != 17 || !a.MustChangePassword {
								t.Fatal("context/actor changed")
							}
						}
						var failure error
						if fail {
							failure = common.ErrForbidden
						}
						list := &mocks.MySessionsList{Run: func(c context.Context, a domain.SessionActor, q domain.MySessionsQuery) (domain.MySessionsPage, error) {
							check(c, a)
							if q.LastID != "%2F+ " || q.Page != 1 || q.Limit != 20 {
								t.Fatal("double decode/defaults")
							}
							return domain.MySessionsPage{Items: []domain.MySession{}, Total: 2147483648, Page: 1, Limit: 20}, failure
						}}
						one := &mocks.MySessionsRevoke{Run: func(c context.Context, a domain.SessionActor, id string) (domain.RevokeMySessionResponse, error) {
							check(c, a)
							if id != fixtures.MySessionID {
								t.Fatal("path changed")
							}
							return domain.RevokeMySessionResponse{Changed: changed}, failure
						}}
						all := &mocks.MySessionsAll{Run: func(c context.Context, a domain.SessionActor) (domain.RevokeAllMySessionsResponse, error) {
							check(c, a)
							return domain.RevokeAllMySessionsResponse{Changed: changed, LoginRequired: login}, failure
						}}
						for _, invoke := range []func() error{func() error { _, e := handler.NewMySessions(nil, one, all, writer, browser); return e }, func() error { _, e := handler.NewMySessions(list, nil, all, writer, browser); return e }, func() error { _, e := handler.NewMySessions(list, one, nil, writer, browser); return e }, func() error { _, e := handler.NewMySessions(list, one, all, nil, browser); return e }, func() error { _, e := handler.NewMySessions(list, one, all, writer, nil); return e }} {
							if invoke() == nil {
								t.Fatal("constructor guard")
							}
						}
						h, e := handler.NewMySessions(list, one, all, writer, browser)
						if e != nil {
							t.Fatal(e)
						}
						app.Use(func(c fiber.Ctx) error {
							requestCtx = requestauth.WithPrincipal(c.Context(), requestauth.Principal{ID: 17, MustChangePassword: true})
							c.SetContext(requestCtx)
							return c.Next()
						})
						h.RegisterRoutes(app.Group("/me"))
						method, path := "GET", "/me/sessions?last_id=%252F%2B%20&user_id=99&namespace=forged"
						if op == "one" {
							method, path = "DELETE", "/me/sessions/"+fixtures.MySessionID
						}
						if op == "all" {
							method, path = "POST", "/me/sessions/revoke-all"
						}
						req := httptest.NewRequest(method, path, strings.NewReader(`{"user_id":999,"session_id":"forged","version":99}`))
						req.Header.Set("Version", "99")
						req.Header.Set("Idempotency-Key", "ignored")
						if incoming {
							req.AddCookie(&http.Cookie{Name: handler.RefreshCookie, Value: "private"})
						}
						res, e := app.Test(req)
						body := testutils.ResponseBody(t, res, e)
						want := 200
						if fail {
							want = 403
						}
						if res.StatusCode != want || calls != 1 || res.Header.Get("Cache-Control") != "no-store" {
							t.Fatal("response/calls", body)
						}
						if !fail {
							switch op {
							case "list":
								if !strings.Contains(body, `"message":"Sessions retrieved successfully."`) || !strings.Contains(body, `"items":[]`) || !strings.Contains(body, `"total":2147483648`) {
									t.Fatal(body)
								}
							case "one":
								if body != fmt.Sprintf(`{"code":"SS","message":"Session revoked successfully.","data":{"changed":%t}}`, changed) {
									t.Fatal(body)
								}
							case "all":
								if body != fmt.Sprintf(`{"code":"SS","message":"All sessions revoked successfully.","data":{"changed":%t,"login_required":%t}}`, changed, login) {
									t.Fatal(body)
								}
							}
						}
						cleanup := op == "all" && !fail && login
						testutils.AssertRefreshCookie(t, res, cleanup)
						if cleanup {
							if len(res.Cookies()) != 2 {
								t.Fatal("missing CSRF deletion")
							}
							for _, c := range res.Cookies() {
								if c.Value != "" || c.MaxAge != -1 || c.Path != "/api/v1/auth" || c.Domain != "" || !c.Secure || c.SameSite != http.SameSiteLaxMode || c.HttpOnly != (c.Name == handler.RefreshCookie) {
									t.Fatal("cookie attrs")
								}
							}
						}
					}
				}
			}
		}
	}
	for _, principal := range []*requestauth.Principal{nil, {}} {
		app, writer := testutils.NewHTTPApp()
		list := &mocks.MySessionsList{}
		one := &mocks.MySessionsRevoke{}
		all := &mocks.MySessionsAll{}
		h, _ := handler.NewMySessions(list, one, all, writer, fixtures.NewBrowserPolicy(t))
		if principal != nil {
			app.Use(func(c fiber.Ctx) error {
				c.SetContext(requestauth.WithPrincipal(c.Context(), *principal))
				return c.Next()
			})
		}
		h.RegisterRoutes(app.Group("/me"))
		for _, op := range []struct{ method, path string }{{"GET", "/me/sessions"}, {"DELETE", "/me/sessions/" + fixtures.MySessionID}, {"POST", "/me/sessions/revoke-all"}} {
			res, e := app.Test(httptest.NewRequest(op.method, op.path, nil))
			testutils.ResponseBody(t, res, e)
			if res.StatusCode != 401 || res.Header.Get("Cache-Control") != "no-store" || len(res.Cookies()) != 0 {
				t.Fatal("principal guard")
			}
		}
	}
}
