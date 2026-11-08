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
	handler "github.com/siti-nabila/rest-orc/internal/features/users/handler"
	"github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

type detailUseCase func(context.Context, domain.UserDetailActor, uint64) (domain.UserDetail, error)

func (f detailUseCase) Execute(c context.Context, a domain.UserDetailActor, id uint64) (domain.UserDetail, error) {
	return f(c, a, id)
}
func UserDetailScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "detail handler exact sources envelope context no-store and no cookies", Run: detailHandler}}
}
func detailHandler(t *testing.T) {
	for _, tc := range []struct {
		name, path, body string
		status           int
		missing, zero    bool
		failure          error
	}{
		{name: "path only empty body", path: "/users/00023?user_id=99", status: 200},
		{name: "malformed ignored body", path: "/users/23?permissions=forged&version=99&fields=password", body: "{", status: 200},
		{name: "forged body", path: "/users/23", body: `{"user_id":99,"roles":["admin"]}`, status: 200},
		{name: "me is not alias", path: "/users/me", status: 400},
		{name: "overflow", path: "/users/2147483648", status: 400},
		{name: "missing principal", path: "/users/23", missing: true, status: 401},
		{name: "zero principal", path: "/users/23", zero: true, status: 401},
		{name: "forbidden", path: "/users/23", failure: common.ErrForbidden, status: 403},
		{name: "invalid result error", path: "/users/23", failure: common.ErrInternalServerError, status: 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app, writer := testutils.NewHTTPApp()
			var requestContext context.Context
			calls := 0
			var expected struct{ Data domain.UserDetail }
			if json.Unmarshal([]byte(fixtures.UserDetailBody), &expected) != nil {
				t.Fatal("fixture")
			}
			uc := detailUseCase(func(c context.Context, a domain.UserDetailActor, id uint64) (domain.UserDetail, error) {
				calls++
				if c != requestContext || a.UserID != 17 || !a.MustChangePassword || !a.Permissions[domain.UserReadPermission] || id != 23 {
					t.Fatal(c, a, id)
				}
				return expected.Data, tc.failure
			})
			if _, err := handler.NewUserDetail(nil, writer); err == nil {
				t.Fatal("nil usecase")
			}
			if _, err := handler.NewUserDetail(uc, nil); err == nil {
				t.Fatal("nil writer")
			}
			h, err := handler.NewUserDetail(uc, writer)
			if err != nil {
				t.Fatal(err)
			}
			h.RegisterRoutes(app, func(c fiber.Ctx) error {
				if !tc.missing {
					p := requestauth.Principal{ID: 17, Version: 1, MustChangePassword: true, Permissions: map[string]bool{domain.UserReadPermission: true}}
					if tc.zero {
						p.ID = 0
					}
					requestContext = requestauth.WithPrincipal(c.Context(), p)
					c.SetContext(requestContext)
				}
				return c.Next()
			})
			req := httptest.NewRequest("GET", tc.path, strings.NewReader(tc.body))
			req.Header["Version"] = []string{"bad", "also bad"}
			req.Header["Idempotency-Key"] = []string{"é", "secret"}
			res, err := app.Test(req)
			body := testutils.ResponseBody(t, res, err)
			wantCalls := 0
			if tc.status == 200 || tc.failure != nil {
				wantCalls = 1
			}
			if res.StatusCode != tc.status || res.Header.Get("Cache-Control") != "no-store" || len(res.Cookies()) != 0 || calls != wantCalls {
				t.Fatal(res.StatusCode, body, calls)
			}
			if tc.status == 200 && body != fixtures.UserDetailBody {
				t.Fatal(body)
			}
			if tc.status != 200 && (!strings.Contains(body, `"data":[]`) || strings.Contains(body, "target@example.com")) {
				t.Fatal(body)
			}
		})
	}
}
