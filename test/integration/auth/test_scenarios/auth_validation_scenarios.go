package test_scenarios

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	authhandler "github.com/siti-nabila/rest-orc/internal/features/auth/handler"
	"github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

func localAuthErrors(t *testing.T) {
	for _, translation := range []struct {
		locale, required, minimum, invalidEmail, invalidCharacters string
	}{
		{"en", "is required", "minimum length is 6", "email is invalid", "contains invalid characters"},
		{"id", "harus diisi", "minimal panjang karakter adalah 6", "email tidak valid", "mengandung karakter tidak valid"},
		{"zh-CN", "必填", "最小长度为 6", "邮箱格式无效", "包含无效字符"},
	} {
		for _, route := range []string{authhandler.RegisterRoute, authhandler.LoginRoute} {
			t.Run(translation.locale+route, func(t *testing.T) {
				service, calls := rejectingAuthService(t)
				app := newAuthApp(t, service)
				for _, tc := range []struct {
					name, payload string
					fields        map[string][]string
				}{
					{"omitted", `{}`, map[string][]string{
						"email": {translation.required, translation.minimum}, "password": {translation.required, translation.minimum},
					}},
					{"null fields", `{"email":null,"password":null}`, map[string][]string{
						"email": {translation.required, translation.minimum}, "password": {translation.required, translation.minimum},
					}},
					{"null body", `null`, map[string][]string{
						"email": {translation.required, translation.minimum}, "password": {translation.required, translation.minimum},
					}},
					{"both invalid", `{"email":"x","password":"ab cd"}`, map[string][]string{
						"email": {translation.minimum, translation.invalidEmail}, "password": {translation.minimum},
					}},
				} {
					t.Run(tc.name, func(t *testing.T) {
						if tc.name == "both invalid" && route == authhandler.RegisterRoute {
							tc.fields["password"] = append(tc.fields["password"], translation.invalidCharacters)
						}
						response, body := authHTTP(t, app, route, tc.payload, translation.locale)
						assertLocalAuthResponse(t, response, body, tc.fields)
						if strings.Contains(body, "ab cd") || strings.Contains(body, `"x"`) {
							t.Error("validation response echoed a credential")
						}
					})
				}
				if *calls != 0 {
					t.Errorf("invalid requests made %d RPC calls, want 0", *calls)
				}
			})
		}
	}
}

func authParsingErrors(t *testing.T) {
	for _, route := range []string{authhandler.RegisterRoute, authhandler.LoginRoute} {
		t.Run(route, func(t *testing.T) {
			service, calls := rejectingAuthService(t)
			app := newAuthApp(t, service)
			for _, payload := range []string{
				`{"email":`, `{"email":17,"password":"secret-password"}`,
				`{"email":"a@b.co","password":false}`, `{"email":[],"password":{}}`,
			} {
				response, body := authHTTP(t, app, route, payload, "en")
				if response.StatusCode != 400 || body != `{"code":"BR","errors":{"description":"Bad request."},"data":[]}` {
					t.Error("malformed or incompatible known field did not retain the bad request envelope")
				}
			}
			if *calls != 0 {
				t.Errorf("parsing failures made %d RPC calls, want 0", *calls)
			}
		})
	}
}

func registerCharacterErrors(t *testing.T) {
	service, calls := rejectingAuthService(t)
	app := newAuthApp(t, service)
	for _, password := range []string{"abc de", "abc\tde", "abc\rde", "abc\nde", "abc\x00de", "abc\x7fde", "ééé"} {
		payload, err := json.Marshal(map[string]string{"email": "a@b.co", "password": password})
		if err != nil {
			t.Fatal(err)
		}
		response, body := authHTTP(t, app, authhandler.RegisterRoute, string(payload), "en")
		assertLocalAuthResponse(t, response, body, map[string][]string{"password": {"contains invalid characters"}})
	}
	if *calls != 0 {
		t.Errorf("invalid characters made %d RPC calls, want 0", *calls)
	}
}

func rejectingAuthService(t *testing.T) (*mocks.Service, *int) {
	t.Helper()
	calls := 0
	reject := func(context.Context, *userv1.AuthRequest, ...grpc.CallOption) (*userv1.UserTokenResponse, error) {
		calls++
		t.Error("invalid public input unexpectedly invoked a business RPC")
		return nil, nil
	}
	return &mocks.Service{
		RegisterFunc: reject, LoginFunc: reject,
		MeFunc: func(context.Context, *emptypb.Empty, ...grpc.CallOption) (*userv1.UserData, error) {
			calls++
			t.Error("public auth request unexpectedly invoked Me")
			return nil, nil
		},
	}, &calls
}

func authHTTP(t *testing.T, app *fiber.App, route, payload, language string) (*http.Response, string) {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, apiV1Prefix+authhandler.Route+route, strings.NewReader(payload))
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	request.Header.Set("Origin", fixtures.BrowserOrigin)
	request.Header.Set(locale.HTTPHeader, language)
	response, err := app.Test(request)
	return response, testutils.ResponseBody(t, response, err)
}

func assertLocalAuthResponse(t *testing.T, response *http.Response, body string, fields map[string][]string) {
	t.Helper()
	if response.StatusCode != http.StatusBadRequest || response.Header.Get(fiber.HeaderRetryAfter) != "" {
		t.Error("local validation must return 400 without Retry-After")
	}
	want, err := json.Marshal(map[string]any{"code": "BR", "errors": fields, "data": []any{}})
	if err != nil {
		t.Fatal(err)
	}
	var actual, expected any
	if err := json.Unmarshal([]byte(body), &actual); err != nil {
		t.Fatal("validation response is not JSON")
	}
	if err := json.Unmarshal(want, &expected); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Error("local validation envelope or translated message lists differ")
	}
}
