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

type creationUseCase func(context.Context, domain.UserCreationActor, domain.CreateUserRequest) (domain.CreateUserResponse, error)

func (f creationUseCase) Execute(c context.Context, a domain.UserCreationActor, r domain.CreateUserRequest) (domain.CreateUserResponse, error) {
	return f(c, a, r)
}
func UserCreationScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "creation HTTP parsing sources exact response context and no browser effects", Run: creationHandler}}
}
func creationHandler(t *testing.T) {
	modes := []string{"success", "lowercase key", "unused version", "empty object", "null fields", "null address", "empty address", "missing principal", "zero principal", "missing key", "duplicate key", "case duplicate key", "empty key", "forbidden", "internal", "empty body", "malformed", "null", "array", "scalar", "unicode whitespace"}
	for _, f := range []string{"email", "password", "name", "phone", "country_code", "address", "status"} {
		modes = append(modes, "wrong "+f)
	}
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			app, writer := testutils.NewHTTPApp()
			calls := 0
			var ctx context.Context
			var fixture struct{ Data domain.UserDetail }
			_ = json.Unmarshal([]byte(shared.UserDetailBody), &fixture)
			uc := creationUseCase(func(c context.Context, a domain.UserCreationActor, r domain.CreateUserRequest) (domain.CreateUserResponse, error) {
				calls++
				if c != ctx || a.UserID != 17 || !a.MustChangePassword || !a.Permissions[domain.UserCreatePermission] || r.IdempotencyKey != "opaque, key" {
					t.Fatal("authority/context")
				}
				if mode == "empty object" || mode == "null fields" {
					if r.Body.Email != nil || r.Body.Password != nil || r.Body.Name != nil {
						t.Fatal("defaults")
					}
				} else if r.Body.Email == nil || *r.Body.Email != " Raw@Example.COM " || r.Body.Password == nil || *r.Body.Password != "simple" {
					t.Fatal("body sources")
				}
				if mode == "null address" && r.Body.Address != nil {
					t.Fatal("null presence")
				}
				if mode == "empty address" && (r.Body.Address == nil || *r.Body.Address != "") {
					t.Fatal("empty presence")
				}
				if mode == "forbidden" {
					return domain.CreateUserResponse{}, common.ErrForbidden
				}
				if mode == "internal" {
					return domain.CreateUserResponse{}, common.ErrInternalServerError
				}
				return domain.CreateUserResponse{User: &fixture.Data}, nil
			})
			if _, err := handler.NewUserCreation(nil, writer); err == nil {
				t.Fatal("nil usecase")
			}
			if _, err := handler.NewUserCreation(uc, nil); err == nil {
				t.Fatal("nil writer")
			}
			h, _ := handler.NewUserCreation(uc, writer)
			h.RegisterRoutes(app, func(c fiber.Ctx) error {
				if mode != "missing principal" {
					id := uint64(17)
					if mode == "zero principal" {
						id = 0
					}
					ctx = requestauth.WithPrincipal(c.Context(), requestauth.Principal{ID: id, MustChangePassword: true, Permissions: map[string]bool{domain.UserCreatePermission: true}})
					c.SetContext(ctx)
				}
				return c.Next()
			})
			raw := `{"email":" Raw@Example.COM ","password":"simple","name":" raw name ","phone":"0812345678901","country_code":"62","id":99,"user_id":99,"profile_id":999,"permissions":{"iam.user.create":false},"roles":["admin"],"application_access":[1],"must_change_password":false,"field_mask":["password"],"idempotency_key":"forged","version":1,"confirm_password":false}`
			want, n := 200, 1
			switch mode {
			case "empty object":
				raw = `{}`
			case "null fields":
				raw = `{"email":null,"password":null,"name":null,"phone":null,"country_code":null}`
			case "null address":
				raw = strings.TrimSuffix(raw, "}") + `,"address":null}`
			case "empty address":
				raw = strings.TrimSuffix(raw, "}") + `,"address":""}`
			case "missing principal", "zero principal":
				want = 401
				n = 0
			case "forbidden":
				want = 403
			case "internal":
				want = 500
			case "empty body":
				raw = ""
			case "malformed":
				raw = "{"
			case "null":
				raw = "null"
			case "array":
				raw = "[]"
			case "scalar":
				raw = "true"
			case "unicode whitespace":
				raw = "\u00a0" + raw
			}
			if strings.HasPrefix(mode, "wrong ") {
				raw = `{"` + strings.TrimPrefix(mode, "wrong ") + `":123}`
			}
			if strings.HasPrefix(mode, "wrong ") || mode == "empty body" || mode == "malformed" || mode == "null" || mode == "array" || mode == "scalar" || mode == "unicode whitespace" {
				want = 400
				n = 0
			}
			req := httptest.NewRequest("POST", "/users?email=forged&password=forged&idempotency_key=forged&version=1", strings.NewReader(raw))
			req.Header.Set("Idempotency-Key", "opaque, key")
			switch mode {
			case "lowercase key":
				req.Header.Del("Idempotency-Key")
				req.Header["idempotency-key"] = []string{"opaque, key"}
			case "unused version":
				req.Header["Version"] = []string{"bad", "also bad"}
				req.Header["version"] = []string{"-1"}
			case "missing key":
				req.Header.Del("Idempotency-Key")
				want = 400
				n = 0
			case "empty key":
				req.Header.Set("Idempotency-Key", "")
				want = 400
				n = 0
			case "duplicate key":
				req.Header.Add("Idempotency-Key", "opaque, key")
				want = 400
				n = 0
			case "case duplicate key":
				req.Header["idempotency-key"] = []string{"opaque, key"}
				want = 400
				n = 0
			}
			res, err := app.Test(req)
			body := testutils.ResponseBody(t, res, err)
			if res.StatusCode != want || calls != n || res.Header.Get("Cache-Control") != "no-store" || len(res.Cookies()) != 0 {
				t.Fatal(mode, res.StatusCode, body, calls, n)
			}
			if want == 200 {
				data, _ := json.Marshal(domain.CreateUserResponse{User: &fixture.Data})
				expected := `{"code":"SS","message":"User created successfully.","data":` + string(data) + `}`
				if body != expected {
					t.Fatal(body)
				}
			} else if !strings.Contains(body, `"data":[]`) {
				t.Fatal(body)
			}
		})
	}
}
