package test_scenarios

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	requestauth "github.com/siti-nabila/rest-orc/internal/auth"
	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	authhandler "github.com/siti-nabila/rest-orc/internal/features/auth/handler"
	"github.com/siti-nabila/rest-orc/test/features/auth/handler/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func PasswordScenarios() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "password handler parses credentials and forwards verified identity and context", Run: passwordHandlerParsing},
		{Name: "password handler preserves exact flags and only clears cookies on confirmed success", Run: passwordHandlerResults},
	}
}

func passwordHandlerParsing(t *testing.T) {
	for _, tc := range []struct {
		name, payload string
		principal     *requestauth.Principal
		status        int
	}{
		{"valid restricted", `{"current_password":" PaSs é ","new_password":"new-password","user_id":999,"confirm_password":"ignored"}`, &requestauth.Principal{ID: 17, MustChangePassword: true}, 200},
		{"null request", `null`, &requestauth.Principal{ID: 17}, 200},
		{"null fields", `{"current_password":null,"new_password":null}`, &requestauth.Principal{ID: 17}, 200},
		{"empty object", `{}`, &requestauth.Principal{ID: 17}, 200},
		{"empty fields", `{"current_password":"","new_password":""}`, &requestauth.Principal{ID: 17}, 200},
		{"malformed", `{`, &requestauth.Principal{ID: 17}, 400},
		{"empty body", ``, &requestauth.Principal{ID: 17}, 400},
		{"numeric current", `{"current_password":1}`, &requestauth.Principal{ID: 17}, 400},
		{"boolean new", `{"new_password":false}`, &requestauth.Principal{ID: 17}, 400},
		{"object current", `{"current_password":{}}`, &requestauth.Principal{ID: 17}, 400},
		{"array new", `{"new_password":[]}`, &requestauth.Principal{ID: 17}, 400},
		{"absent principal", `{}`, nil, 401},
		{"zero identity", `{}`, &requestauth.Principal{}, 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var expected domain.ChangePasswordRequest
			if tc.status == 200 {
				if err := json.Unmarshal([]byte(tc.payload), &expected); err != nil {
					t.Fatal(err)
				}
			}
			var requestContext context.Context
			usecase := &mocks.PasswordUseCase{ExecuteFunc: func(ctx context.Context, userID uint64, request domain.ChangePasswordRequest) (domain.ChangePasswordResponse, error) {
				if ctx != requestContext || userID != 17 || request != expected {
					t.Error("parsed credential, verified identity, or context changed")
				}
				return domain.ChangePasswordResponse{}, nil
			}}
			app, writer := testutils.NewHTTPApp()
			handler, err := authhandler.New(unexpectedAuthUseCase(t), unexpectedLoginUseCase(t), usecase, writer, fixtures.NewBrowserPolicy(t))
			if err != nil {
				t.Fatal(err)
			}
			app.Use(func(ctx fiber.Ctx) error {
				requestContext = context.WithValue(ctx.Context(), requestIDKey, "password-request")
				if tc.principal != nil {
					requestContext = requestauth.WithPrincipal(requestContext, *tc.principal)
				}
				ctx.SetContext(requestContext)
				return ctx.Next()
			})
			handler.RegisterAuthenticatedRoutes(app.Group("/me"))
			request := httptest.NewRequest(http.MethodPost, "/me/password?user_id=999", strings.NewReader(tc.payload))
			request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			response, err := app.Test(request)
			body := testutils.ResponseBody(t, response, err)
			wantCalls := 0
			if tc.status == 200 {
				wantCalls = 1
			}
			if response.StatusCode != tc.status || usecase.Calls != wantCalls {
				t.Fatalf("status/calls=%d/%d, want %d/%d; body=%s", response.StatusCode, usecase.Calls, tc.status, wantCalls, body)
			}
			if response.Header.Get(fiber.HeaderCacheControl) != "no-store" {
				t.Fatal("password handler response must not be cached")
			}
			testutils.AssertRefreshCookie(t, response, false)
		})
	}
}

func passwordHandlerResults(t *testing.T) {
	for _, changed := range []bool{false, true} {
		for _, loginRequired := range []bool{false, true} {
			for _, failure := range []error{nil, common.ErrForbidden} {
				for _, incoming := range []bool{false, true} {
					usecase := &mocks.PasswordUseCase{ExecuteFunc: func(context.Context, uint64, domain.ChangePasswordRequest) (domain.ChangePasswordResponse, error) {
						return domain.ChangePasswordResponse{Changed: changed, LoginRequired: loginRequired}, failure
					}}
					app, writer := testutils.NewHTTPApp()
					handler, err := authhandler.New(unexpectedAuthUseCase(t), unexpectedLoginUseCase(t), usecase, writer, fixtures.NewBrowserPolicy(t))
					if err != nil {
						t.Fatal(err)
					}
					app.Use(func(ctx fiber.Ctx) error {
						ctx.SetContext(requestauth.WithPrincipal(ctx.Context(), requestauth.Principal{ID: 17, MustChangePassword: true}))
						return ctx.Next()
					})
					handler.RegisterAuthenticatedRoutes(app.Group("/me"))
					request := httptest.NewRequest(http.MethodPost, "/me/password", strings.NewReader(`{"current_password":"legacy password","new_password":"new-password"}`))
					request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
					if incoming {
						request.AddCookie(&http.Cookie{Name: "__Secure-refresh", Value: "private-refresh-sentinel"})
					}
					response, err := app.Test(request)
					body := testutils.ResponseBody(t, response, err)
					if failure == nil {
						want := fmt.Sprintf(`{"code":"SS","message":"Password changed successfully.","data":{"changed":%t,"login_required":%t}}`, changed, loginRequired)
						if response.StatusCode != 200 || body != want {
							t.Fatal("success envelope or flags changed")
						}
					} else if response.StatusCode != 403 || strings.Contains(body, `"code":"SS"`) {
						t.Fatal("error became success")
					}
					testutils.AssertRefreshCookie(t, response, failure == nil && loginRequired)
					if usecase.Calls != 1 || response.Header.Get(fiber.HeaderCacheControl) != "no-store" || strings.Contains(body, "private-refresh-sentinel") {
						t.Fatal("call count, cache policy, or credential confidentiality changed")
					}
				}
			}
		}
	}
}
