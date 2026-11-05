package test_scenarios

import (
	"context"
	"encoding/json"
	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	requestauth "github.com/siti-nabila/rest-orc/internal/auth"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	usershandler "github.com/siti-nabila/rest-orc/internal/features/users/handler"
	"github.com/siti-nabila/rest-orc/test/features/users/handler/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func ProfileScenarios() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "profile handler parses presence and forwards verified actor/context without business normalization", Run: profileHandlerParsing},
		{Name: "profile handler writes exact authoritative result and mapped errors", Run: profileHandlerResults},
	}
}
func profileHandlerParsing(t *testing.T) {
	for _, tc := range []struct {
		raw       string
		status    int
		principal bool
	}{
		{`{}`, 200, true}, {`null`, 200, true}, {`{"user_id":88,"field_mask":["address"]}`, 200, true},
		{`{"name":null,"phone":null,"country_code":null,"address":null}`, 200, true},
		{`{"name":"","phone":"","address":"","country_code":""}`, 200, true},
		{`{"name":"  nASA  o'neil ","phone":null,"user_id":88}`, 200, true},
		{`{`, 400, true}, {``, 400, true}, {`{"name":17}`, 400, true}, {`{"address":false}`, 400, true}, {`{"phone":[]}`, 400, true}, {`{"country_code":{}}`, 400, true},
		{`{}`, 401, false},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			var expected domain.UpdateProfileRequest
			if tc.status == 200 {
				if err := json.Unmarshal([]byte(tc.raw), &expected); err != nil {
					t.Fatal(err)
				}
			}
			var requestContext context.Context
			updater := &mocks.ProfileUseCase{ExecuteFunc: func(ctx context.Context, actor domain.ProfileActor, request domain.UpdateProfileRequest) (domain.UpdateProfileResponse, error) {
				if ctx != requestContext || actor.UserID != 17 || !actor.MustChangePassword {
					t.Fatalf("actor/context=%#v %v", actor, ctx)
				}
				got, _ := json.Marshal(request)
				want, _ := json.Marshal(expected)
				if string(got) != string(want) {
					t.Fatalf("parsed request=%s want %s", got, want)
				}
				return domain.UpdateProfileResponse{}, nil
			}}
			app, writer := testutils.NewHTTPApp()
			handler, err := usershandler.New(&mocks.ListUseCase{}, updater, writer)
			if err != nil {
				t.Fatal(err)
			}
			if tc.principal {
				app.Use(func(ctx fiber.Ctx) error {
					requestContext = requestauth.WithPrincipal(context.WithValue(ctx.Context(), requestIDKey, "profile-17"), requestauth.Principal{ID: 17, MustChangePassword: true})
					ctx.SetContext(requestContext)
					return ctx.Next()
				})
			}
			handler.RegisterAuthenticatedRoutes(app.Group("/me"))
			request := httptest.NewRequest(http.MethodPatch, "/me/profile?user_id=99", strings.NewReader(tc.raw))
			request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			response, err := app.Test(request)
			body := testutils.ResponseBody(t, response, err)
			wantCalls := 0
			if tc.status == 200 {
				wantCalls = 1
				if body != `{"code":"SS","message":"Profile updated successfully.","data":{"changed":false}}` {
					t.Fatalf("no-op body=%s", body)
				}
			}
			if response.StatusCode != tc.status || updater.Calls != wantCalls {
				t.Fatalf("status=%d body=%s calls=%d", response.StatusCode, body, updater.Calls)
			}
		})
	}
}
func profileHandlerResults(t *testing.T) {
	for _, failure := range []error{nil, common.ErrForbidden} {
		version := "9007199254740993"
		updater := &mocks.ProfileUseCase{ExecuteFunc: func(context.Context, domain.ProfileActor, domain.UpdateProfileRequest) (domain.UpdateProfileResponse, error) {
			return domain.UpdateProfileResponse{Changed: true, Version: &version, Profile: &domain.Profile{ID: 9, UserID: 17, Name: "NASA", Address: "Jakarta", Phone: "812345678901", CountryCode: "+62"}}, failure
		}}
		app, writer := testutils.NewHTTPApp()
		handler, err := usershandler.New(&mocks.ListUseCase{}, updater, writer)
		if err != nil {
			t.Fatal(err)
		}
		app.Use(func(ctx fiber.Ctx) error {
			ctx.SetContext(requestauth.WithPrincipal(ctx.Context(), requestauth.Principal{ID: 17}))
			return ctx.Next()
		})
		handler.RegisterAuthenticatedRoutes(app.Group("/me"))
		request := httptest.NewRequest(http.MethodPatch, "/me/profile", strings.NewReader(`{"name":"NASA"}`))
		request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		response, err := app.Test(request)
		body := testutils.ResponseBody(t, response, err)
		if failure == nil {
			if response.StatusCode != 200 || body != `{"code":"SS","message":"Profile updated successfully.","data":{"changed":true,"version":"9007199254740993","profile":{"id":9,"user_id":17,"name":"NASA","address":"Jakarta","phone":"812345678901","country_code":"+62"}}}` {
				t.Fatalf("result=%d %s", response.StatusCode, body)
			}
		} else if response.StatusCode != 403 || strings.Contains(body, "9007199254740993") {
			t.Fatalf("error=%d %s", response.StatusCode, body)
		}
		if updater.Calls != 1 {
			t.Fatal("call count")
		}
	}
}
