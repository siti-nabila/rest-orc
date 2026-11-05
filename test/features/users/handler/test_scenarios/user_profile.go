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
	shared "github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

type targetProfileUseCase func(context.Context, domain.UserProfileActor, domain.UpdateUserProfileRequest) (domain.UpdateUserProfileResponse, error)

func (f targetProfileUseCase) Execute(c context.Context, a domain.UserProfileActor, r domain.UpdateUserProfileRequest) (domain.UpdateUserProfileResponse, error) {
	return f(c, a, r)
}
func UserProfileScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "target profile handler sources JSON presence headers exact envelopes and context", Run: targetProfileHandler}}
}
func targetProfileHandler(t *testing.T) {
	for _, mode := range []string{"success", "unkeyed", "lowercase headers", "null", "empty object", "null fields", "unknown fields", "empty body", "malformed", "array", "scalar", "unicode outer whitespace", "wrong name", "wrong address", "wrong phone", "wrong country", "missing principal", "zero principal", "invalid target", "missing version", "duplicate version", "case duplicate version", "empty key", "duplicate key", "case duplicate key", "forbidden", "internal"} {
		t.Run(mode, func(t *testing.T) {
			app, writer := testutils.NewHTTPApp()
			calls := 0
			var requestContext context.Context
			noOp := mode == "null" || mode == "empty object" || mode == "null fields" || mode == "unknown fields"
			var fixture struct{ Data domain.UserDetail }
			if json.Unmarshal([]byte(shared.UserDetailBody), &fixture) != nil {
				t.Fatal("fixture")
			}
			uc := targetProfileUseCase(func(c context.Context, a domain.UserProfileActor, r domain.UpdateUserProfileRequest) (domain.UpdateUserProfileResponse, error) {
				calls++
				if c != requestContext || a.UserID != 17 || !a.MustChangePassword || !a.Permissions[domain.UserUpdatePermission] || r.UserID != 23 || r.Version != 9007199254740993 {
					t.Fatal(a, r)
				}
				if mode == "unkeyed" {
					if r.IdempotencyKey != nil {
						t.Fatal("key fallback")
					}
				} else if r.IdempotencyKey == nil || *r.IdempotencyKey != "opaque, key" {
					t.Fatal("key")
				}
				if noOp {
					if r.Profile.HasFields() {
						t.Fatal("forged mask")
					}
					return domain.UpdateUserProfileResponse{}, nil
				}
				if r.Profile.Name == nil || *r.Profile.Name != "  NASA  " || r.Profile.Phone != nil || r.Profile.Address != nil || r.Profile.CountryCode != nil {
					t.Fatal(r)
				}
				if mode == "forbidden" {
					return domain.UpdateUserProfileResponse{}, common.ErrForbidden
				}
				if mode == "internal" {
					return domain.UpdateUserProfileResponse{}, common.ErrInternalServerError
				}
				return domain.UpdateUserProfileResponse{User: &fixture.Data, Changed: false}, nil
			})
			if _, err := handler.NewUserProfile(nil, writer); err == nil {
				t.Fatal("nil usecase")
			}
			if _, err := handler.NewUserProfile(uc, nil); err == nil {
				t.Fatal("nil writer")
			}
			h, err := handler.NewUserProfile(uc, writer)
			if err != nil {
				t.Fatal(err)
			}
			h.RegisterRoutes(app, func(c fiber.Ctx) error {
				if mode != "missing principal" {
					id := uint64(17)
					if mode == "zero principal" {
						id = 0
					}
					requestContext = requestauth.WithPrincipal(c.Context(), requestauth.Principal{ID: id, Version: 1, MustChangePassword: true, Permissions: map[string]bool{domain.UserUpdatePermission: true}})
					c.SetContext(requestContext)
				}
				return c.Next()
			})
			raw := `{"name":"  NASA  ","phone":null,"user_id":99,"version":7,"email":"forged","status":"ACTIVE","field_mask":["phone"]}`
			want, callsWant := 200, 1
			switch mode {
			case "null":
				raw = "null"
			case "empty object":
				raw = "{}"
			case "null fields":
				raw = `{"name":null,"phone":null,"address":null,"country_code":null}`
			case "unknown fields":
				raw = `{"email":"forged","password":"secret","user_id":99,"version":1,"field_mask":["phone"]}`
			case "empty body":
				raw = ""
			case "malformed":
				raw = "{"
			case "array":
				raw = "[]"
			case "scalar":
				raw = "1"
			case "unicode outer whitespace":
				raw = "\u00a0" + raw
			case "wrong name":
				raw = `{"name":true}`
			case "wrong address":
				raw = `{"address":{}}`
			case "wrong phone":
				raw = `{"phone":123}`
			case "wrong country":
				raw = `{"country_code":[]}`
			case "missing principal", "zero principal":
				want = 401
			case "forbidden":
				want = 403
			case "internal":
				want = 500
			}
			switch mode {
			case "empty body", "malformed", "array", "scalar", "unicode outer whitespace", "wrong name", "wrong address", "wrong phone", "wrong country", "invalid target", "missing version", "duplicate version", "case duplicate version", "empty key", "duplicate key", "case duplicate key":
				want = 400
				callsWant = 0
			case "missing principal", "zero principal":
				callsWant = 0
			}
			path := "/users/00023/profile?user_id=99&name=forged&version=1&idempotency_key=forged"
			if mode == "invalid target" {
				path = "/users/me/profile"
			}
			req := httptest.NewRequest("PATCH", path, strings.NewReader(raw))
			req.Header.Set("Version", "9007199254740993")
			req.Header.Set("Idempotency-Key", "opaque, key")
			switch mode {
			case "unkeyed":
				req.Header.Del("Idempotency-Key")
			case "lowercase headers":
				req.Header.Del("Version")
				req.Header.Del("Idempotency-Key")
				req.Header["version"] = []string{"9007199254740993"}
				req.Header["idempotency-key"] = []string{"opaque, key"}
			case "missing version":
				req.Header.Del("Version")
			case "duplicate version":
				req.Header.Add("Version", "9007199254740993")
			case "case duplicate version":
				req.Header["version"] = []string{"9007199254740993"}
			case "empty key":
				req.Header.Set("Idempotency-Key", "")
			case "duplicate key":
				req.Header.Add("Idempotency-Key", "opaque, key")
			case "case duplicate key":
				req.Header["idempotency-key"] = []string{"opaque, key"}
			}
			res, err := app.Test(req)
			body := testutils.ResponseBody(t, res, err)
			if res.StatusCode != want || calls != callsWant || res.Header.Get("Cache-Control") != "no-store" || len(res.Cookies()) != 0 {
				t.Fatal(res.StatusCode, want, body, calls, callsWant)
			}
			if want == 200 {
				expected := `{"code":"SS","message":"User profile updated successfully.","data":{"changed":false}}`
				if !noOp {
					data, _ := json.Marshal(domain.UpdateUserProfileResponse{User: &fixture.Data})
					expected = `{"code":"SS","message":"User profile updated successfully.","data":` + string(data) + `}`
				}
				if body != expected {
					t.Fatal(body, expected)
				}
			} else if !strings.Contains(body, `"data":[]`) {
				t.Fatal(body)
			}
		})
	}
}
