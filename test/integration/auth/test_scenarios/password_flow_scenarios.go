package test_scenarios

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	authv1 "github.com/siti-nabila/api-contracts/pb/auth/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	authdictionary "github.com/siti-nabila/api-contracts/pkg/dictionary/auth"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	"github.com/siti-nabila/rest-orc/internal/client/authservice"
	"github.com/siti-nabila/rest-orc/internal/client/grpcclient"
	clientmocks "github.com/siti-nabila/rest-orc/test/client/authservice/mocks"
	grpcfixtures "github.com/siti-nabila/rest-orc/test/client/grpcclient/fixtures"
	sharedfixtures "github.com/siti-nabila/rest-orc/test/shared/fixtures"
	sharedmocks "github.com/siti-nabila/rest-orc/test/shared/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

const passwordBearer = "Bearer password-user-sentinel"

func PasswordScenarios() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "password HTTP validates raw input and restricted eligibility with exact RPC counts", Run: passwordHTTPInputs},
		{Name: "password HTTP keeps backend flags and conditional browser cleanup authoritative", Run: passwordHTTPResults},
		{Name: "password HTTP aggregates localized errors without credential leakage", Run: passwordHTTPLocales},
		{Name: "password HTTP preserves authentication and mutation failures without retry or cleanup", Run: passwordHTTPFailures},
	}
}

func passwordHTTPInputs(t *testing.T) {
	type example struct {
		name, payload       string
		status              int
		restricted, missing bool
		want                *authv1.ChangePasswordRequest
	}
	cases := []example{
		{name: "non-admin raw credentials and forged claims", payload: `{"current_password":" PaSs é ","new_password":"new-password","user_id":999,"email":"other@example.com","roles":["admin"],"permissions":{"all":true},"namespace":"iam","service_token":"forged","field_mask":["password"],"version":"bad","confirm_password":"ignored"}`, status: 200, want: &authv1.ChangePasswordRequest{CurrentPassword: " PaSs é ", NewPassword: "new-password"}},
		{name: "restricted", payload: `{"current_password":"legacy password","new_password":"new-password"}`, status: 200, restricted: true, want: &authv1.ChangePasswordRequest{CurrentPassword: "legacy password", NewPassword: "new-password"}},
		{name: "case-sensitive difference", payload: `{"current_password":"aaaaaa","new_password":"Aaaaaa"}`, status: 200, want: &authv1.ChangePasswordRequest{CurrentPassword: "aaaaaa", NewPassword: "Aaaaaa"}},
		{name: "empty object", payload: `{}`, status: 400},
		{name: "JSON null", payload: `null`, status: 400},
		{name: "null fields", payload: `{"current_password":null,"new_password":null}`, status: 400},
		{name: "unknown fields only", payload: `{"confirm_password":"ignored","user_id":999}`, status: 400},
		{name: "missing new", payload: `{"current_password":"legacy password"}`, status: 400},
		{name: "missing current", payload: `{"new_password":"new-password"}`, status: 400},
		{name: "empty fields", payload: `{"current_password":"","new_password":""}`, status: 400},
		{name: "equal", payload: `{"current_password":"same-password","new_password":"same-password"}`, status: 400},
		{name: "malformed", payload: `{`, status: 400},
		{name: "empty body", status: 400},
		{name: "missing bearer", payload: `{"current_password":"legacy password","new_password":"new-password"}`, status: 401, missing: true},
	}
	for _, field := range []string{"current_password", "new_password"} {
		for _, value := range []any{17, false, []any{}, map[string]any{}} {
			payload, _ := json.Marshal(map[string]any{field: value})
			cases = append(cases, example{name: field + " wrong type", payload: string(payload), status: 400})
		}
		for _, size := range []int{5, 6, 50, 51} {
			current, newPassword := "legacy password", "new-password"
			if field == "current_password" {
				current = strings.Repeat("a", size)
			} else {
				newPassword = strings.Repeat("a", size)
			}
			payload, _ := json.Marshal(map[string]string{"current_password": current, "new_password": newPassword})
			tc := example{name: fmt.Sprintf("%s length %d", field, size), payload: string(payload), status: 400}
			if size == 6 || size == 50 {
				tc.status = 200
				tc.want = &authv1.ChangePasswordRequest{CurrentPassword: current, NewPassword: newPassword}
			}
			cases = append(cases, tc)
		}
	}
	for _, value := range []string{"      ", "ééé", "ab\x00\t\r\n"} {
		payload, _ := json.Marshal(map[string]string{"current_password": value, "new_password": "new-password"})
		cases = append(cases, example{name: "legacy current characters", payload: string(payload), status: 200, want: &authv1.ChangePasswordRequest{CurrentPassword: value, NewPassword: "new-password"}})
	}
	for _, value := range []string{"new password", "new\tpassword", "new\rpassword", "new\npassword", "new\x00password", "new\x7fpassword", "ééé"} {
		payload, _ := json.Marshal(map[string]string{"current_password": "legacy password", "new_password": value})
		cases = append(cases, example{name: "invalid new characters", payload: string(payload), status: 400})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := []string{}
			user := &sharedmocks.Service{MeFunc: func(ctx context.Context, _ *emptypb.Empty, _ ...grpc.CallOption) (*userv1.UserData, error) {
				calls = append(calls, "Me")
				assertPasswordContext(t, ctx)
				result := sharedfixtures.MeUserData()
				result.MustChangePassword = tc.restricted
				return result, nil
			}}
			password := &clientmocks.PasswordService{ChangeFunc: func(ctx context.Context, request *authv1.ChangePasswordRequest, _ ...grpc.CallOption) (*authv1.ChangePasswordResponse, error) {
				calls = append(calls, "ChangePassword")
				assertPasswordContext(t, ctx)
				if !proto.Equal(request, tc.want) {
					t.Error("RPC credential payload changed or browser claims entered it")
				}
				return &authv1.ChangePasswordResponse{Changed: true, LoginRequired: true}, nil
			}}
			response, body := passwordHTTPRequest(t, newPasswordApp(t, user, password), tc.payload, "id-ID", !tc.missing)
			wantCalls := []string{}
			if !tc.missing {
				wantCalls = append(wantCalls, "Me")
			}
			if tc.want != nil {
				wantCalls = append(wantCalls, "ChangePassword")
			}
			if response.StatusCode != tc.status || !reflect.DeepEqual(calls, wantCalls) {
				t.Fatalf("status/calls=%d/%v, want %d/%v", response.StatusCode, calls, tc.status, wantCalls)
			}
			if tc.status == 200 {
				if body != `{"code":"SS","message":"Password changed successfully.","data":{"changed":true,"login_required":true}}` || response.Header.Get(fiber.HeaderCacheControl) != "no-store" {
					t.Fatal("success contract changed")
				}
			} else if !strings.Contains(body, `"data":[]`) {
				t.Fatal("error data must be empty")
			}
			testutils.AssertRefreshCookie(t, response, tc.status == 200)
			assertPasswordSecrets(t, body)
		})
	}
}

func passwordHTTPResults(t *testing.T) {
	for _, changed := range []bool{false, true} {
		for _, loginRequired := range []bool{false, true} {
			for _, nilResult := range []bool{false, true} {
				calls := []string{}
				user := &sharedmocks.Service{MeFunc: func(context.Context, *emptypb.Empty, ...grpc.CallOption) (*userv1.UserData, error) {
					calls = append(calls, "Me")
					return sharedfixtures.MeUserData(), nil
				}}
				password := &clientmocks.PasswordService{ChangeFunc: func(context.Context, *authv1.ChangePasswordRequest, ...grpc.CallOption) (*authv1.ChangePasswordResponse, error) {
					calls = append(calls, "ChangePassword")
					if nilResult {
						return nil, nil
					}
					return &authv1.ChangePasswordResponse{Changed: changed, LoginRequired: loginRequired}, nil
				}}
				response, body := passwordHTTPRequest(t, newPasswordApp(t, user, password), `{"current_password":"legacy password","new_password":"new-password"}`, "en", true)
				if nilResult {
					if response.StatusCode != 500 || !strings.Contains(body, `"data":[]`) {
						t.Fatal("nil result became success")
					}
				} else {
					want := fmt.Sprintf(`{"code":"SS","message":"Password changed successfully.","data":{"changed":%t,"login_required":%t}}`, changed, loginRequired)
					if response.StatusCode != 200 || body != want {
						t.Fatal("authoritative flags changed")
					}
				}
				if !reflect.DeepEqual(calls, []string{"Me", "ChangePassword"}) || response.Header.Get(fiber.HeaderCacheControl) != "no-store" {
					t.Fatal("call order/count or cache policy changed")
				}
				testutils.AssertRefreshCookie(t, response, !nilResult && loginRequired)
				assertPasswordSecrets(t, body)
			}
		}
	}
}

func passwordHTTPLocales(t *testing.T) {
	for _, tc := range []struct{ language, difference, characters string }{
		{"en", "must differ from the current password", "contains invalid characters"},
		{"id-ID", "harus berbeda dari password saat ini", "mengandung karakter tidak valid"},
		{"zh-CN", "必须与当前密码不同", "包含无效字符"},
	} {
		for _, equal := range []bool{false, true} {
			calls := 0
			user := &sharedmocks.Service{MeFunc: func(context.Context, *emptypb.Empty, ...grpc.CallOption) (*userv1.UserData, error) {
				calls++
				return sharedfixtures.MeUserData(), nil
			}}
			password := &clientmocks.PasswordService{}
			payload := `{"current_password":"short","new_password":" "}`
			if equal {
				payload = `{"current_password":"same-password","new_password":"same-password"}`
			}
			response, body := passwordHTTPRequest(t, newPasswordApp(t, user, password), payload, tc.language, true)
			var result struct {
				Code   string
				Errors map[string][]string
				Data   []any
			}
			if err := json.Unmarshal([]byte(body), &result); err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != 400 || result.Code != "BR" || len(result.Data) != 0 || calls != 1 || password.Calls != 0 {
				t.Fatal("local validation mapping or call count differs")
			}
			if equal {
				if len(result.Errors) != 1 || !reflect.DeepEqual(result.Errors["new_password"], []string{tc.difference}) {
					t.Fatal("localized equality error differs")
				}
			} else if len(result.Errors) != 2 || len(result.Errors["current_password"]) != 1 || len(result.Errors["new_password"]) != 2 || !strings.Contains(body, tc.characters) {
				t.Fatal("localized field errors did not aggregate")
			}
			testutils.AssertRefreshCookie(t, response, false)
			assertPasswordSecrets(t, body)
		}
	}
}

func passwordHTTPFailures(t *testing.T) {
	for _, stage := range []string{"Me", "ChangePassword"} {
		for _, mode := range []string{"unauthorized", "forbidden", "conflict", "rate limit", "unavailable fields", "wrong current", "deadline", "cancelled", "internal"} {
			t.Run(stage+"/"+mode, func(t *testing.T) {
				var failure error
				wantStatus := 500
				switch mode {
				case "unauthorized":
					failure = status.Error(codes.Unauthenticated, "private-password-backend-sentinel")
					wantStatus = 401
				case "forbidden":
					failure = status.Error(codes.PermissionDenied, "private-password-backend-sentinel")
					wantStatus = 403
				case "conflict":
					failure = testutils.PasswordFieldError(t, 409)
					wantStatus = 409
				case "rate limit":
					failure = testutils.PasswordFieldError(t, 429)
					wantStatus = 429
				case "unavailable fields":
					failure = testutils.PasswordFieldError(t, 503)
					wantStatus = 503
				case "wrong current":
					failure = authdictionary.ErrPasswordMismatch
					wantStatus = authdictionary.ErrPasswordMismatch.HTTPStatus()
				case "deadline":
					failure = context.DeadlineExceeded
					wantStatus = 504
				case "cancelled":
					failure = context.Canceled
					wantStatus = 499
				default:
					failure = status.Error(codes.Internal, "private-password-backend-sentinel")
				}
				calls := []string{}
				user := &sharedmocks.Service{MeFunc: func(context.Context, *emptypb.Empty, ...grpc.CallOption) (*userv1.UserData, error) {
					calls = append(calls, "Me")
					if stage == "Me" {
						return nil, failure
					}
					return sharedfixtures.MeUserData(), nil
				}}
				password := &clientmocks.PasswordService{ChangeFunc: func(context.Context, *authv1.ChangePasswordRequest, ...grpc.CallOption) (*authv1.ChangePasswordResponse, error) {
					calls = append(calls, "ChangePassword")
					return nil, failure
				}}
				response, body := passwordHTTPRequest(t, newPasswordApp(t, user, password), `{"current_password":"legacy password","new_password":"new-password"}`, "id-ID", true)
				wantCalls := []string{"Me"}
				if stage == "ChangePassword" {
					wantCalls = append(wantCalls, stage)
				}
				if response.StatusCode != wantStatus || !reflect.DeepEqual(calls, wantCalls) || !strings.Contains(body, `"data":[]`) {
					t.Fatal("downstream mapping, RPC order, or no-retry contract differs")
				}
				if mode == "conflict" || mode == "rate limit" || mode == "unavailable fields" {
					testutils.AssertPasswordErrorResponse(t, response, body, wantStatus)
				}
				testutils.AssertRefreshCookie(t, response, false)
				assertPasswordSecrets(t, body)
			})
		}
	}
}

func newPasswordApp(t *testing.T, user authservice.Service, password authservice.PasswordService) *fiber.App {
	t.Helper()
	transport, err := grpcclient.New(grpcfixtures.ValidConfig())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := transport.Close(); err != nil {
			t.Error(err)
		}
	})
	client, err := authservice.NewWithPasswordService(transport, user, &clientmocks.ProfileService{}, password)
	if err != nil {
		t.Fatal(err)
	}
	return sharedfixtures.NewApp(t, client)
}

func passwordHTTPRequest(t *testing.T, app *fiber.App, payload, language string, bearer bool) (*http.Response, string) {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/me/password?user_id=999&namespace=iam", strings.NewReader(payload))
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	request.Header.Set(locale.HTTPHeader, language)
	request.Header.Set("X-Service-Token", "forged-service-sentinel")
	request.Header.Set("Namespace", "iam")
	request.Header.Set("User_ID", "999")
	request.Header.Set("Version", "invalid-ignored")
	request.Header.Set("Idempotency-Key", "ignored")
	request.AddCookie(&http.Cookie{Name: "__Secure-refresh", Value: "private-refresh-sentinel"})
	if bearer {
		request.Header.Set(fiber.HeaderAuthorization, passwordBearer)
	}
	response, err := app.Test(request)
	return response, testutils.ResponseBody(t, response, err)
}

func assertPasswordContext(t *testing.T, ctx context.Context) {
	t.Helper()
	testutils.AssertServiceContext(t, ctx)
	md, _ := metadata.FromOutgoingContext(ctx)
	if !reflect.DeepEqual(md.Get("authorization"), []string{passwordBearer}) {
		t.Error("user bearer changed")
	}
	for _, key := range []string{"namespace", "user_id", "x-service-token", "cookie", "refresh_token", "idempotency-key", "version"} {
		if len(md.Get(key)) != 0 {
			t.Error("browser claim or cookie entered outbound metadata")
		}
	}
}

func assertPasswordSecrets(t *testing.T, body string) {
	t.Helper()
	for _, secret := range []string{passwordBearer, "forged-service-sentinel", "private-refresh-sentinel", "private-password-backend-sentinel", "same-password", "legacy password", "new-password"} {
		if strings.Contains(body, secret) {
			t.Fatal("HTTP response exposed a credential or private dependency detail")
		}
	}
}
