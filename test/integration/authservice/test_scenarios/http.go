package test_scenarios

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/gofiber/fiber/v3"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	authdictionary "github.com/siti-nabila/api-contracts/pkg/dictionary/auth"
	commondictionary "github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	usersfixtures "github.com/siti-nabila/rest-orc/test/integration/users/fixtures"
	sharedfixtures "github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
)

const browserToken = "untrusted-browser-service-sentinel"
const userBearer = "Bearer verified-user-test"

func secureHTTPOutcomes(t *testing.T) {
	for _, tc := range []struct {
		name, method, path, payload, wantBody string
		bearer                                bool
		status                                int
		methods                               []string
	}{
		{"register", http.MethodPost, "/api/v1/users/create", `{"email":" User.Name@Example.COM ","password":"valid-password","x-service-token":"ignored","namespace":"iam","user_id":"999"}`, sharedfixtures.RegisterSuccessBody, false, 200, []string{userv1.UserService_Register_FullMethodName}},
		{"login", http.MethodPost, "/api/v1/users/login", `{"email":" User.Name@Example.COM ","password":"legacy password","x-service-token":"ignored","namespace":"iam","user_id":"999"}`, sharedfixtures.LoginSuccessBody, false, 200, []string{userv1.UserService_Login_FullMethodName}},
		{"register validation", http.MethodPost, "/api/v1/users/create", `{}`, `{"code":"BR","errors":{"email":["is required","minimum length is 6"],"password":["is required","minimum length is 6"]},"data":[]}`, false, 400, nil},
		{"login validation", http.MethodPost, "/api/v1/users/login", `{}`, `{"code":"BR","errors":{"email":["is required","minimum length is 6"],"password":["is required","minimum length is 6"]},"data":[]}`, false, 400, nil},
		{"protected list", http.MethodGet, "/api/v1/users?page=2&limit=20", ``, usersfixtures.SuccessBody, true, 200, []string{userv1.UserService_Me_FullMethodName, userv1.UserService_ListUsers_FullMethodName}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				cfg := secureConfig(t)
				h := newTransport(t, cfg, defaultServerTLS(t), nil)
				defer h.close(t)
				app := sharedfixtures.NewApp(t, h.client)
				response, body := secureHTTPRequest(t, app, tc.method, tc.path, tc.payload, tc.bearer)
				if response.StatusCode != tc.status || body != tc.wantBody {
					t.Errorf("existing HTTP contract changed: status %d, expected %d", response.StatusCode, tc.status)
				}
				wantCookies := 0
				if tc.name == "login" {
					wantCookies = 2
				}
				if len(response.Header.Values(fiber.HeaderSetCookie)) != wantCookies {
					t.Error("service authentication added a browser cookie")
				}
				calls, _ := h.snapshot()
				assertHTTPCalls(t, calls, tc.methods, cfg.ServiceToken, tc.bearer)
				wantPassword := "valid-password"
				if tc.name == "login" {
					wantPassword = "legacy password"
				}
				for _, got := range calls {
					if req, ok := got.request.(*userv1.AuthRequest); ok {
						if req.GetEmail() != " User.Name@Example.COM " || req.GetPassword() != wantPassword {
							t.Error("HTTP credential input was changed before RPC")
						}
					}
				}
				assertNoSecrets(t, body, cfg.ServiceToken)
			})
		})
	}
}

func secureHTTPErrors(t *testing.T) {
	for _, tc := range []struct {
		name, path, method, payload, reason, code, description string
		status                                                 int
		rpcCode                                                codes.Code
		bearer, conflict                                       bool
	}{
		{"unknown credential", "/api/v1/users/login", http.MethodPost, `{"email":"a@b.co","password":"valid-password"}`, authdictionary.ErrServiceAuthenticationFailed.Key(), "SF", "Service authentication failed.", 401, codes.Unauthenticated, false, false},
		{"expired credential", "/api/v1/users/create", http.MethodPost, `{"email":"a@b.co","password":"valid-password"}`, authdictionary.ErrServiceAuthenticationFailed.Key(), "SF", "Service authentication failed.", 401, codes.Unauthenticated, false, false},
		{"method denied", "/api/v1/users?page=2", http.MethodGet, "", commondictionary.ErrForbidden.Key(), "FB", "Forbidden.", 403, codes.PermissionDenied, true, false},
		{"namespace denied", "/api/v1/users?page=2", http.MethodGet, "", commondictionary.ErrForbidden.Key(), "FB", "Forbidden.", 403, codes.PermissionDenied, true, false},
		{"email conflict", "/api/v1/users/create", http.MethodPost, `{"email":"a@b.co","password":"valid-password"}`, authdictionary.ErrDataExists.Key(), "EMAIL_USED", "", 409, codes.Internal, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				s, err := status.New(tc.rpcCode, "private-backend-sentinel").WithDetails(&errdetails.ErrorInfo{Reason: tc.reason, Metadata: map[string]string{
					"http_status": strconv.Itoa(tc.status), "code": tc.code,
				}})
				if err != nil {
					t.Fatal(err)
				}
				fields := any(map[string]string{"description": tc.description})
				if tc.conflict {
					s, err = s.WithDetails(
						&errdetails.BadRequest{FieldViolations: []*errdetails.BadRequest_FieldViolation{{Field: "email", Description: "Email is already used."}}},
						&errdetails.RetryInfo{RetryDelay: &durationpb.Duration{Seconds: 1, Nanos: 200000000}},
					)
					if err != nil {
						t.Fatal(err)
					}
					fields = map[string][]string{"email": {"Email is already used."}}
				}
				cfg := secureConfig(t)
				// Return controlled backend outcomes; do not reimplement its
				// credential verifier, expiry clock or permission database.
				h := newTransport(t, cfg, defaultServerTLS(t), func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
					if tc.bearer && info.FullMethod == userv1.UserService_Me_FullMethodName {
						return handler(ctx, req)
					}
					return nil, s.Err()
				})
				defer h.close(t)
				response, body := secureHTTPRequest(t, sharedfixtures.NewApp(t, h.client), tc.method, tc.path, tc.payload, tc.bearer)
				want, err := json.Marshal(map[string]any{"code": tc.code, "errors": fields, "data": []any{}})
				if err != nil {
					t.Fatal(err)
				}
				var actual, expected any
				if err := json.Unmarshal([]byte(body), &actual); err != nil {
					t.Fatal("HTTP error response is not JSON")
				}
				if err := json.Unmarshal(want, &expected); err != nil {
					t.Fatal(err)
				}
				if response.StatusCode != tc.status || !reflect.DeepEqual(actual, expected) {
					t.Errorf("downstream HTTP status/code/safe body changed: status %d, want %d", response.StatusCode, tc.status)
				}
				wantRetry := ""
				if tc.conflict {
					wantRetry = "2"
				}
				if response.Header.Get(fiber.HeaderRetryAfter) != wantRetry {
					t.Error("downstream retry guidance changed")
				}
				methods := []string{userv1.UserService_Register_FullMethodName}
				if tc.path == "/api/v1/users/login" {
					methods = []string{userv1.UserService_Login_FullMethodName}
				}
				if tc.bearer {
					methods = []string{userv1.UserService_Me_FullMethodName, userv1.UserService_ListUsers_FullMethodName}
				}
				calls, _ := h.snapshot()
				assertHTTPCalls(t, calls, methods, cfg.ServiceToken, tc.bearer)
				assertNoSecrets(t, body, cfg.ServiceToken)
			})
		})
	}
}

func secureHTTPRequest(t *testing.T, app *fiber.App, method, path, payload string, bearer bool) (*http.Response, string) {
	t.Helper()
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	request := httptest.NewRequest(method, path+separator+"x-service-token=ignored&namespace=clm&user_id=999", strings.NewReader(payload))
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	request.Header.Set("Origin", sharedfixtures.BrowserOrigin)
	request.Header.Set(locale.HTTPHeader, "en")
	request.Header.Set("X-Service-Token", browserToken)
	request.Header.Set("X-Service-Name", "untrusted-service")
	request.Header.Set("Namespace", "iam")
	request.Header.Set("User_ID", "999")
	if bearer {
		request.Header.Set(fiber.HeaderAuthorization, userBearer)
	}
	response, err := app.Test(request)
	return response, testutils.ResponseBody(t, response, err)
}

func assertHTTPCalls(t *testing.T, calls []call, methods []string, token string, bearer bool) {
	t.Helper()
	if len(calls) != len(methods) {
		t.Fatalf("RPC count = %d, want %d without retry", len(calls), len(methods))
	}
	for i, got := range calls {
		if got.method != methods[i] {
			t.Error("HTTP pipeline changed RPC ordering")
		}
		assertMetadata(t, got.metadata, "x-service-token", token)
		assertMetadata(t, got.metadata, locale.MetadataKey, "en")
		if bearer {
			assertMetadata(t, got.metadata, "authorization", userBearer)
		} else if len(got.metadata.Get("authorization")) != 0 {
			t.Error("public route invented a user bearer")
		}
		for _, key := range []string{"x-service-name", "namespace", "user_id"} {
			if len(got.metadata.Get(key)) != 0 {
				t.Error("browser identity or namespace claim reached the backend")
			}
		}
	}
}

func assertNoSecrets(t *testing.T, body, token string) {
	t.Helper()
	for _, secret := range []string{token, browserToken, userBearer, "private-backend-sentinel", "valid-password", "legacy password"} {
		if strings.Contains(body, secret) {
			t.Error("HTTP body exposed a credential or private detail")
		}
	}
}
