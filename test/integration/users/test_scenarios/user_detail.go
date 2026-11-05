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
	sessionv1 "github.com/siti-nabila/api-contracts/pb/session/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	"github.com/siti-nabila/rest-orc/internal/client/authservice"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	clientmocks "github.com/siti-nabila/rest-orc/test/client/authservice/mocks"
	shared "github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func UserDetailScenarios() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "detail HTTP exact authority inputs safe results and no-store", Run: detailHTTPOutcomes},
		{Name: "detail HTTP preserves localized errors and dependency ordering", Run: detailHTTPFailures},
		{Name: "detail source feeds separate revoke-all with independent permissions and stale conflict", Run: detailHTTPVersionFlow},
	}
}

func detailRequest() *http.Request {
	req := httptest.NewRequest("GET", "/api/v1/users/00023?user_id=99&namespace=forged&fields=password", strings.NewReader("{"))
	req.Header.Set("Authorization", authorizationValue)
	req.Header["Version"] = []string{"bad", "duplicate"}
	req.Header["Idempotency-Key"] = []string{"é", "duplicate"}
	req.Header.Set(locale.HTTPHeader, "id-ID")
	req.Header.Set("X-Service-Token", "browser-secret")
	req.AddCookie(&http.Cookie{Name: "__Secure-refresh", Value: "browser-refresh"})
	return req
}
func newDetailApp(t *testing.T, user authservice.Service, detail authservice.UserDetailService, admin authservice.AdminSessionService, before ...fiber.Handler) *fiber.App {
	t.Helper()
	if admin == nil {
		admin = &clientmocks.AdminSessionService{}
	}
	client, err := authservice.NewWithUserDetailService(newTransport(t), user, &clientmocks.ProfileService{}, &clientmocks.PasswordService{}, &clientmocks.SessionService{}, &clientmocks.SelfSessionService{}, admin, detail)
	if err != nil {
		t.Fatal(err)
	}
	return shared.NewApp(t, client, before...)
}
func detailHTTPOutcomes(t *testing.T) {
	for _, mode := range []string{"success", "self", "target restricted", "PENDING", "SUSPENDED", "DEACTIVATED", "empty arrays", "nil result", "nil user", "wrong target", "zero version", "overflow version", "missing timestamp", "invalid timestamp", "invalid status", "nil role", "nil app", "missing bearer", "cookie only", "Me failure", "restricted actor", "role only", "wildcard", "false permission", "session only", "invalid target", "me target", "duplicate ignored headers", "listusers", "preflight", "denied origin", "empty body"} {
		t.Run(mode, func(t *testing.T) {
			sequence := []string{}
			target := uint64(23)
			if mode == "self" {
				target = 17
			}
			want := 200
			user := &mocks.Service{MeFunc: func(c context.Context, _ *emptypb.Empty, _ ...grpc.CallOption) (*userv1.UserData, error) {
				sequence = append(sequence, "Me")
				assertAuthorizationMetadata(t, c)
				if mode == "Me failure" {
					return nil, common.ErrUnauthorized
				}
				u := shared.MeUserData()
				u.Version = 1
				u.Permissions = map[string]bool{domain.UserReadPermission: true}
				switch mode {
				case "restricted actor":
					u.MustChangePassword = true
				case "role only":
					u.RoleCodes = []int32{2}
					u.RoleNames = []string{"admin"}
					u.Permissions = nil
				case "wildcard":
					u.Permissions = map[string]bool{"iam.user.*": true}
				case "false permission":
					u.Permissions[domain.UserReadPermission] = false
				case "session only":
					u.Permissions = map[string]bool{domain.SessionRevokePermission: true}
				}
				return u, nil
			}}
			sdk := &clientmocks.UserDetailService{Get: func(c context.Context, r *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
				sequence = append(sequence, "GetUser")
				assertAuthorizationMetadata(t, c)
				if r.Id != target {
					t.Error("wrong target", r)
				}
				md, _ := metadata.FromOutgoingContext(c)
				if len(md.Get("version")) != 0 || len(md.Get("idempotency-key")) != 0 || len(md.Get("namespace")) != 0 || len(md.Get("x-service-token")) != 0 {
					t.Error("authority leaked", md)
				}
				result := shared.UserDetailResponse(target)
				switch mode {
				case "nil result":
					return nil, nil
				case "nil user":
					result.User = nil
				case "wrong target":
					result.User.Id = 99
				case "zero version":
					result.User.Version = 0
				case "overflow version":
					result.User.Version = 1 << 63
				case "missing timestamp":
					result.User.CreatedAt = nil
				case "invalid timestamp":
					result.User.CreatedAt = &timestamppb.Timestamp{Seconds: 1 << 62}
				case "invalid status":
					result.User.Status = "UNKNOWN"
				case "nil role":
					result.User.Roles[0] = nil
				case "nil app":
					result.User.ApplicationAccess[0] = nil
				case "target restricted":
					result.User.MustChangePassword = true
				case "PENDING", "SUSPENDED", "DEACTIVATED":
					result.User.Status = mode
				case "empty arrays":
					result.User.Roles = nil
					result.User.ApplicationAccess = nil
				}
				return result, nil
			}}
			md := metadata.Pairs("version", "parent", "idempotency-key", "parent", "x-request-id", "detail-http")
			parent := metadata.NewOutgoingContext(context.Background(), md)
			app := newDetailApp(t, user, sdk, nil, func(c fiber.Ctx) error { c.SetContext(parent); return c.Next() })
			req := detailRequest()
			expected := []string{"Me", "GetUser"}
			switch mode {
			case "nil result", "nil user", "wrong target", "zero version", "overflow version", "missing timestamp", "invalid timestamp", "invalid status", "nil role", "nil app":
				want = 500
			case "self":
				req.URL.Path = "/api/v1/users/17"
			case "missing bearer", "cookie only":
				req.Header.Del("Authorization")
				want = 401
				expected = nil
			case "Me failure":
				want = 401
				expected = []string{"Me"}
			case "restricted actor", "role only", "wildcard", "false permission", "session only":
				want = 403
				expected = []string{"Me"}
			case "invalid target":
				req.URL.Path = "/api/v1/users/2147483648"
				want = 400
				expected = []string{"Me"}
			case "me target":
				req.URL.Path = "/api/v1/users/me"
				want = 400
				expected = []string{"Me"}
			case "listusers":
				req.URL.Path = "/api/v1/users"
				req.URL.RawQuery = ""
				want = 403
				expected = []string{"Me"}
			case "preflight", "denied origin":
				req.Method = "OPTIONS"
				req.Header.Set("Origin", shared.BrowserOrigin)
				req.Header.Set("Access-Control-Request-Method", "GET")
				want = 204
				expected = nil
				if mode == "denied origin" {
					req.Header.Set("Origin", "https://evil.example.test")
					want = 403
				}
			case "empty body":
				req.Body = http.NoBody
			}
			res, err := app.Test(req)
			body := testutils.ResponseBody(t, res, err)
			if res.StatusCode != want || len(sequence) != len(expected) || len(res.Cookies()) != 0 {
				t.Fatal(res.StatusCode, want, body, sequence)
			}
			if len(expected) > 0 && !reflect.DeepEqual(sequence, expected) {
				t.Fatal(sequence, expected)
			}
			if mode != "listusers" && res.Header.Get("Cache-Control") != "no-store" {
				t.Fatal("no-store")
			}
			if want == 200 {
				expectedBody := shared.UserDetailBody
				switch mode {
				case "self":
					expectedBody = strings.Replace(expectedBody, `"id":23`, `"id":17`, 1)
				case "target restricted":
					expectedBody = strings.Replace(expectedBody, `"must_change_password":false`, `"must_change_password":true`, 1)
				case "PENDING", "SUSPENDED", "DEACTIVATED":
					expectedBody = strings.Replace(expectedBody, `"status":"ACTIVE"`, `"status":"`+mode+`"`, 1)
				}
				if mode == "empty arrays" {
					var env struct{ Data map[string]json.RawMessage }
					if json.Unmarshal([]byte(body), &env) != nil || len(env.Data) != 12 || string(env.Data["roles"]) != "[]" || string(env.Data["application_access"]) != "[]" || string(env.Data["must_change_password"]) != "false" {
						t.Fatal(body)
					}
				} else if body != expectedBody {
					t.Fatal(body, expectedBody)
				}
			} else if want != 204 && !strings.Contains(body, `"data":[]`) {
				t.Fatal(body)
			}
			for _, secret := range []string{"private-password", "private-dob", "browser-secret", "browser-refresh"} {
				if strings.Contains(body, secret) {
					t.Fatal("secret leaked")
				}
			}
			after, _ := metadata.FromOutgoingContext(parent)
			if !reflect.DeepEqual(after, md) {
				t.Fatal("parent mutated")
			}
		})
	}
}
func detailHTTPFailures(t *testing.T) {
	for _, stage := range []string{"Me", "GetUser"} {
		for _, code := range []int{401, 403, 404, 409, 429, 503, 504, 499, 500} {
			t.Run(fmt.Sprint(stage, "/", code), func(t *testing.T) {
				var failure error
				switch code {
				case 409, 429, 503:
					failure = testutils.ServiceFieldError(t, code)
				case 401:
					failure = common.ErrUnauthorized
				case 403:
					failure = status.Error(codes.PermissionDenied, "private-sentinel")
				case 404:
					failure = status.Error(codes.NotFound, "private-sentinel")
				case 504:
					failure = context.DeadlineExceeded
				case 499:
					failure = context.Canceled
				default:
					failure = status.Error(codes.Internal, "private-sentinel")
				}
				calls := []string{}
				user := &mocks.Service{MeFunc: func(context.Context, *emptypb.Empty, ...grpc.CallOption) (*userv1.UserData, error) {
					calls = append(calls, "Me")
					if stage == "Me" {
						return nil, failure
					}
					u := shared.MeUserData()
					u.Permissions[domain.UserReadPermission] = true
					return u, nil
				}}
				sdk := &clientmocks.UserDetailService{Get: func(context.Context, *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
					calls = append(calls, "GetUser")
					return nil, failure
				}}
				app := newDetailApp(t, user, sdk, nil)
				res, err := app.Test(detailRequest())
				body := testutils.ResponseBody(t, res, err)
				n := 1
				if stage == "GetUser" {
					n = 2
				}
				if res.StatusCode != code || len(calls) != n || res.Header.Get("Cache-Control") != "no-store" || len(res.Cookies()) != 0 || strings.Contains(body, "private-sentinel") || !strings.Contains(body, `"data":[]`) {
					t.Fatal(res.StatusCode, body, calls)
				}
				if code == 409 || code == 429 || code == 503 {
					testutils.AssertServiceErrorResponse(t, res, body, code)
				}
			})
		}
	}
}
func detailHTTPVersionFlow(t *testing.T) {
	for _, mode := range []string{"success", "stale conflict", "read only", "revoke only", "fresh read"} {
		t.Run(mode, func(t *testing.T) {
			calls := []string{}
			reads := 0
			user := &mocks.Service{MeFunc: func(context.Context, *emptypb.Empty, ...grpc.CallOption) (*userv1.UserData, error) {
				calls = append(calls, "Me")
				u := shared.MeUserData()
				u.Version = 1
				u.Permissions = map[string]bool{domain.UserReadPermission: mode != "revoke only", domain.SessionRevokePermission: mode != "read only"}
				return u, nil
			}}
			sdk := &clientmocks.UserDetailService{Get: func(_ context.Context, r *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
				calls = append(calls, "GetUser")
				reads++
				if r.Id != 23 {
					t.Fatal(r)
				}
				result := shared.UserDetailResponse(23)
				if reads == 2 {
					result.User.Version = 9223372036854775807
					result.User.Name = "new profile"
				}
				return result, nil
			}}
			admin := &clientmocks.AdminSessionService{All: func(c context.Context, r *sessionv1.RevokeAllUserSessionsRequest) (*sessionv1.RevokeAllUserSessionsResponse, error) {
				calls = append(calls, "RevokeAllUserSessions")
				md, _ := metadata.FromOutgoingContext(c)
				if r.UserId != 23 || r.ExpectedUserVersion == nil || *r.ExpectedUserVersion != 9007199254740993 || !reflect.DeepEqual(md.Get("idempotency-key"), []string{"flow-key"}) {
					t.Fatal(r, md)
				}
				if mode == "stale conflict" {
					return nil, testutils.ServiceFieldError(t, 409)
				}
				return &sessionv1.RevokeAllUserSessionsResponse{Changed: true, Version: 9007199254740994}, nil
			}}
			app := newDetailApp(t, user, sdk, admin)
			res, err := app.Test(detailRequest())
			body := testutils.ResponseBody(t, res, err)
			var env struct {
				Data struct {
					Version string
					Name    string
				}
			}
			if mode == "revoke only" {
				if res.StatusCode != 403 || !reflect.DeepEqual(calls, []string{"Me"}) {
					t.Fatal(res.StatusCode, body, calls)
				}
				return
			}
			if res.StatusCode != 200 || json.Unmarshal([]byte(body), &env) != nil || env.Data.Version != "9007199254740993" {
				t.Fatal(body)
			}
			if mode == "fresh read" {
				res, err = app.Test(detailRequest())
				body = testutils.ResponseBody(t, res, err)
				if res.StatusCode != 200 || json.Unmarshal([]byte(body), &env) != nil || env.Data.Version != "9223372036854775807" || env.Data.Name != "new profile" || !reflect.DeepEqual(calls, []string{"Me", "GetUser", "Me", "GetUser"}) {
					t.Fatal(body, calls)
				}
				return
			}
			req := httptest.NewRequest("POST", "/api/v1/users/23/sessions/revoke-all", nil)
			req.Header.Set("Authorization", authorizationValue)
			req.Header.Set("Version", env.Data.Version)
			req.Header.Set("Idempotency-Key", "flow-key")
			res, err = app.Test(req)
			body = testutils.ResponseBody(t, res, err)
			want := 200
			expected := []string{"Me", "GetUser", "Me", "RevokeAllUserSessions"}
			if mode == "stale conflict" {
				want = 409
			}
			if mode == "read only" {
				want = 403
				expected = []string{"Me", "GetUser", "Me"}
			}
			if res.StatusCode != want || len(res.Cookies()) != 0 || !reflect.DeepEqual(calls, expected) {
				t.Fatal(res.StatusCode, body, calls)
			}
			if want == 409 {
				testutils.AssertServiceErrorResponse(t, res, body, want)
			}
		})
	}
}
