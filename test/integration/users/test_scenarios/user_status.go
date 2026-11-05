package test_scenarios

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	usererrors "github.com/siti-nabila/api-contracts/pkg/dictionary/user"
	"github.com/siti-nabila/api-contracts/pkg/grpcerror"
	"github.com/siti-nabila/api-contracts/pkg/grpcerror/mapping"
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
)

func UserStatusScenarios() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "status HTTP authority input outcome safety and early failures", Run: statusHTTPOutcomes},
		{Name: "status encoded user conflicts and existing safe error mapping", Run: statusHTTPFailures},
		{Name: "status separate detail permission version and replay current authority", Run: statusHTTPFlow},
	}
}
func statusRequest() *http.Request {
	req := httptest.NewRequest("PATCH", "/api/v1/users/00023/status?user_id=99&namespace=forged&version=1&status=DEACTIVATED", strings.NewReader(`{"status":"ACTIVE","reason":"  Approved\r\nreason  ","user_id":99,"version":1,"permissions":{"iam.user.status.update":true}}`))
	req.Header.Set("Authorization", authorizationValue)
	req.Header.Set("Version", "9007199254740993")
	req.Header.Set("Idempotency-Key", "opaque, key")
	req.Header.Set(locale.HTTPHeader, "id-ID")
	req.Header.Set("X-Service-Token", "browser-secret")
	req.AddCookie(&http.Cookie{Name: "__Secure-refresh", Value: "browser-refresh"})
	return req
}
func newStatusApp(t *testing.T, user authservice.Service, sdk authservice.UserStatusService, detail authservice.UserDetailService, before ...fiber.Handler) *fiber.App {
	t.Helper()
	if detail == nil {
		detail = &clientmocks.UserDetailService{}
	}
	client, err := authservice.NewWithUserStatusService(newTransport(t), user, &clientmocks.ProfileService{}, &clientmocks.PasswordService{}, &clientmocks.SessionService{}, &clientmocks.SelfSessionService{}, &clientmocks.AdminSessionService{}, detail, sdk)
	if err != nil {
		t.Fatal(err)
	}
	return shared.NewApp(t, client, before...)
}
func statusHTTPOutcomes(t *testing.T) {
	for _, mode := range []string{"success", "changed", "self", "PENDING", "SUSPENDED", "DEACTIVATED", "max version", "leading zero version", "nil result", "zero ID", "large ID", "wrong ID", "bad status", "zero version", "overflow version", "missing bearer", "cookie only", "Me failure", "invalid Me", "restricted actor", "role only", "wildcard", "prefix", "false permission", "read only", "session only", "invalid target", "literal me", "empty body", "null body", "array body", "scalar body", "wrong type", "missing fields", "null fields", "both invalid", "invalid raw edge", "missing key", "duplicate key", "missing version", "duplicate version", "preflight", "bad preflight target", "denied origin", "POST alias", "unversioned alias", "listusers"} {
		t.Run(mode, func(t *testing.T) {
			calls := []string{}
			target := uint64(23)
			state := "ACTIVE"
			version := uint64(9007199254740993)
			want := 200
			expected := []string{"Me", "ChangeUserStatus"}
			if mode == "self" {
				target = 17
			}
			if mode == "PENDING" || mode == "SUSPENDED" || mode == "DEACTIVATED" {
				state = mode
			}
			if mode == "max version" {
				version = 9223372036854775807
			}
			user := &mocks.Service{MeFunc: func(c context.Context, _ *emptypb.Empty, _ ...grpc.CallOption) (*userv1.UserData, error) {
				calls = append(calls, "Me")
				assertAuthorizationMetadata(t, c)
				if mode == "Me failure" {
					return nil, common.ErrUnauthorized
				}
				u := shared.MeUserData()
				u.Version = 1
				u.Permissions = map[string]bool{domain.UserStatusUpdatePermission: true}
				switch mode {
				case "invalid Me":
					u.Id = 0
				case "restricted actor":
					u.MustChangePassword = true
				case "role only":
					u.RoleCodes = []int32{2}
					u.RoleNames = []string{"admin"}
					u.Permissions = nil
				case "wildcard":
					u.Permissions = map[string]bool{"iam.user.*": true}
				case "prefix":
					u.Permissions = map[string]bool{"iam.user.status": true}
				case "false permission":
					u.Permissions[domain.UserStatusUpdatePermission] = false
				case "read only":
					u.Permissions = map[string]bool{domain.UserReadPermission: true}
				case "session only":
					u.Permissions = map[string]bool{domain.SessionRevokePermission: true}
				}
				return u, nil
			}}
			sdk := &clientmocks.UserStatusService{Change: func(c context.Context, r *userv1.ChangeUserStatusRequest) (*userv1.ChangeUserStatusResponse, error) {
				calls = append(calls, "ChangeUserStatus")
				assertAuthorizationMetadata(t, c)
				md, _ := metadata.FromOutgoingContext(c)
				if r.UserId != target || r.Status == nil || *r.Status != state || r.ExpectedUserVersion == nil || *r.ExpectedUserVersion != version || r.Reason != "Approved\r\nreason" || !reflect.DeepEqual(md.Get("idempotency-key"), []string{"opaque, key"}) || len(md.Get("version")) != 0 || md.Get("x-request-id")[0] != "status-http" || len(md.Get("namespace")) != 0 || len(md.Get("x-service-token")) != 0 {
					t.Error(r, md)
				}
				result := &userv1.ChangeUserStatusResponse{UserId: target, Status: state, Version: version, Changed: mode == "changed"}
				switch mode {
				case "nil result":
					return nil, nil
				case "zero ID":
					result.UserId = 0
				case "large ID":
					result.UserId = 2147483648
				case "wrong ID":
					result.UserId = 99
				case "bad status":
					result.Status = "active"
				case "zero version":
					result.Version = 0
				case "overflow version":
					result.Version = 1 << 63
				}
				return result, nil
			}}
			md := metadata.Pairs("version", "parent", "idempotency-key", "parent", "x-request-id", "status-http")
			parent := metadata.NewOutgoingContext(context.Background(), md)
			app := newStatusApp(t, user, sdk, nil, func(c fiber.Ctx) error { c.SetContext(parent); return c.Next() })
			req := statusRequest()
			setBody := func(raw string) {
				req.Body = http.NoBody
				req.ContentLength = 0
				if raw != "" {
					req.Body = io.NopCloser(strings.NewReader(raw))
					req.ContentLength = int64(len(raw))
				}
			}
			switch mode {
			case "self":
				req.URL.Path = "/api/v1/users/17/status"
			case "PENDING", "SUSPENDED", "DEACTIVATED":
				setBody(fmt.Sprintf(`{"status":%q,"reason":"  Approved\r\nreason  "}`, state))
			case "max version":
				req.Header.Set("Version", "9223372036854775807")
			case "leading zero version":
				req.Header.Set("Version", "0009007199254740993")
			case "nil result", "zero ID", "large ID", "wrong ID", "bad status", "zero version", "overflow version":
				want = 500
			case "missing bearer", "cookie only":
				req.Header.Del("Authorization")
				want = 401
				expected = nil
			case "Me failure", "invalid Me":
				want = 401
				expected = []string{"Me"}
			case "restricted actor", "role only", "wildcard", "prefix", "false permission", "read only", "session only", "listusers":
				want = 403
				expected = []string{"Me"}
				if mode == "listusers" {
					req.Method = "GET"
					req.URL.Path = "/api/v1/users"
					req.URL.RawQuery = ""
				}
			case "invalid target", "literal me":
				req.URL.Path = "/api/v1/users/2147483648/status"
				if mode == "literal me" {
					req.URL.Path = "/api/v1/users/me/status"
				}
				want = 400
				expected = []string{"Me"}
			case "empty body", "null body", "array body", "scalar body", "wrong type", "missing fields", "null fields", "both invalid", "invalid raw edge":
				raw := map[string]string{"null body": "null", "array body": "[]", "scalar body": "1", "wrong type": `{"status":true,"reason":"Approved"}`, "missing fields": "{}", "null fields": `{"status":null,"reason":null}`, "both invalid": `{"status":"active","reason":"ab"}`, "invalid raw edge": `{"status":"ACTIVE","reason":"\tApproved"}`}[mode]
				setBody(raw)
				want = 400
				expected = []string{"Me"}
			case "missing key", "duplicate key", "missing version", "duplicate version":
				want = 400
				expected = []string{"Me"}
				switch mode {
				case "missing key":
					req.Header.Del("Idempotency-Key")
				case "duplicate key":
					req.Header.Add("Idempotency-Key", "opaque, key")
				case "missing version":
					req.Header.Del("Version")
				case "duplicate version":
					req.Header.Add("Version", "9007199254740993")
				}
			case "preflight", "bad preflight target", "denied origin":
				req.Method = "OPTIONS"
				req.Header.Set("Origin", shared.BrowserOrigin)
				req.Header.Set("Access-Control-Request-Method", "PATCH")
				want = 204
				expected = nil
				if mode == "bad preflight target" {
					req.URL.Path = "/api/v1/users/bad/status"
				}
				if mode == "denied origin" {
					req.Header.Set("Origin", "https://evil.example.test")
					want = 403
				}
			case "POST alias":
				req.Method = "POST"
				want = 403
				expected = []string{"Me"}
			case "unversioned alias":
				req.URL.Path = "/users/23/status"
				want = 404
				expected = nil
			}
			res, err := app.Test(req)
			body := testutils.ResponseBody(t, res, err)
			if res.StatusCode != want || !equalCalls(calls, expected) || len(res.Cookies()) != 0 {
				t.Fatal(res.StatusCode, want, body, calls, expected)
			}
			if mode != "unversioned alias" && mode != "listusers" && res.Header.Get("Cache-Control") != "no-store" {
				t.Fatal("no-store", body)
			}
			if want == 200 {
				expectedBody := fmt.Sprintf(`{"code":"SS","message":"User status updated successfully.","data":{"user_id":%d,"status":%q,"version":"%d","changed":%t}}`, target, state, version, mode == "changed")
				if body != expectedBody {
					t.Fatal(body, expectedBody)
				}
			} else if want != 204 && !strings.Contains(body, `"data":[]`) {
				t.Fatal(body)
			}
			if mode == "both invalid" || mode == "missing fields" || mode == "null fields" {
				var env struct{ Errors map[string]json.RawMessage }
				_ = json.Unmarshal([]byte(body), &env)
				if len(env.Errors) != 2 || env.Errors["status"] == nil || env.Errors["reason"] == nil {
					t.Fatal(body)
				}
			}
			for _, secret := range []string{"browser-secret", "browser-refresh", "parent", "Approved"} {
				if want != 200 && strings.Contains(body, secret) {
					t.Fatal("leak", body)
				}
			}
			after, _ := metadata.FromOutgoingContext(parent)
			if !reflect.DeepEqual(after, md) {
				t.Fatal("parent mutated")
			}
		})
	}
}
func equalCalls(a, b []string) bool { return strings.Join(a, ",") == strings.Join(b, ",") }

func statusHTTPFailures(t *testing.T) {
	encoder := grpcerror.NewEncoder(mapping.UserCodeMappings()...)
	cases := []struct {
		name string
		err  error
		code string
		http int
	}{
		{"version", encoder.Encode(usererrors.ErrVersionConflict, locale.Indonesian), "VC", 409},
		{"idempotency", encoder.Encode(usererrors.ErrIdempotencyConflict, locale.Indonesian), "IC", 409},
		{"transition", encoder.Encode(usererrors.ErrInvalidStatusTransition, locale.Indonesian), "IST", 409},
		{"busy", encoder.Encode(usererrors.ErrResourceBusy, locale.Indonesian), "RB", 409},
		{"self", encoder.Encode(usererrors.ErrSelfInactivationForbidden, locale.Indonesian), "SIF", 403},
		{"last admin", encoder.Encode(usererrors.ErrLastAdminRequired, locale.Indonesian), "LAR", 409},
		{"unauthorized", common.ErrUnauthorized, "", 401}, {"forbidden", status.Error(codes.PermissionDenied, "private-sentinel"), "", 403},
		{"not found", status.Error(codes.NotFound, "private-sentinel"), "", 404}, {"rate", testutils.ServiceFieldError(t, 429), "EMAIL_USED", 429}, {"unavailable", testutils.ServiceFieldError(t, 503), "EMAIL_USED", 503},
		{"timeout", context.DeadlineExceeded, "", 504}, {"cancel", context.Canceled, "", 499}, {"unknown", status.Error(codes.Internal, "private-sentinel"), "", 500},
	}
	for _, stage := range []string{"Me", "ChangeUserStatus"} {
		for _, tc := range cases {
			t.Run(stage+"/"+tc.name, func(t *testing.T) {
				calls := []string{}
				user := &mocks.Service{MeFunc: func(context.Context, *emptypb.Empty, ...grpc.CallOption) (*userv1.UserData, error) {
					calls = append(calls, "Me")
					if stage == "Me" {
						return nil, tc.err
					}
					u := shared.MeUserData()
					u.Permissions = map[string]bool{domain.UserStatusUpdatePermission: true}
					return u, nil
				}}
				sdk := &clientmocks.UserStatusService{Change: func(context.Context, *userv1.ChangeUserStatusRequest) (*userv1.ChangeUserStatusResponse, error) {
					calls = append(calls, "ChangeUserStatus")
					return &userv1.ChangeUserStatusResponse{UserId: 23, Status: "ACTIVE", Version: 1}, tc.err
				}}
				app := newStatusApp(t, user, sdk, nil)
				res, err := app.Test(statusRequest())
				body := testutils.ResponseBody(t, res, err)
				n := 1
				if stage != "Me" {
					n = 2
				}
				if res.StatusCode != tc.http || len(calls) != n || len(res.Cookies()) != 0 || res.Header.Get("Cache-Control") != "no-store" || strings.Contains(body, "private-sentinel") || !strings.Contains(body, `"data":[]`) {
					t.Fatal(res.StatusCode, body, calls)
				}
				if tc.code != "" && !strings.Contains(body, `"code":"`+tc.code+`"`) {
					t.Fatal(body)
				}
				if tc.code == "EMAIL_USED" {
					testutils.AssertServiceErrorResponse(t, res, body, tc.http)
				}
				if tc.code == "VC" || tc.code == "IC" || tc.code == "RB" {
					if !strings.Contains(body, common.ErrInternalServerError.Message(locale.Indonesian)) {
						t.Fatal("unregistered fallback", body)
					}
				}
				if tc.code == "IST" || tc.code == "LAR" {
					if !strings.Contains(body, common.ErrBadRequest.Message(locale.Indonesian)) {
						t.Fatal("unregistered fallback", body)
					}
				}
			})
		}
	}
}
func statusHTTPFlow(t *testing.T) {
	for _, mode := range []string{"success", "stale conflict", "read only", "status only", "replay", "revoked authority", "key conflict"} {
		t.Run(mode, func(t *testing.T) {
			calls := []string{}
			commands := 0
			user := &mocks.Service{MeFunc: func(context.Context, *emptypb.Empty, ...grpc.CallOption) (*userv1.UserData, error) {
				calls = append(calls, "Me")
				u := shared.MeUserData()
				u.Version = 1
				u.Permissions = map[string]bool{domain.UserReadPermission: mode != "status only", domain.UserStatusUpdatePermission: mode != "read only"}
				return u, nil
			}}
			detail := &clientmocks.UserDetailService{Get: func(_ context.Context, r *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
				calls = append(calls, "GetUser")
				return shared.UserDetailResponse(r.Id), nil
			}}
			sdk := &clientmocks.UserStatusService{Change: func(c context.Context, r *userv1.ChangeUserStatusRequest) (*userv1.ChangeUserStatusResponse, error) {
				commands++
				calls = append(calls, "ChangeUserStatus")
				if r.UserId != 23 || r.ExpectedUserVersion == nil || *r.ExpectedUserVersion != 9007199254740993 {
					t.Fatal(r)
				}
				encoder := grpcerror.NewEncoder(mapping.UserCodeMappings()...)
				if mode == "stale conflict" {
					return nil, encoder.Encode(usererrors.ErrVersionConflict, locale.Indonesian)
				}
				if commands == 2 {
					if mode == "revoked authority" {
						return nil, status.Error(codes.PermissionDenied, "private-sentinel")
					}
					if mode == "key conflict" {
						return nil, encoder.Encode(usererrors.ErrIdempotencyConflict, locale.Indonesian)
					}
				}
				// Stored authoritative replay may have no input-derived version growth.
				return &userv1.ChangeUserStatusResponse{UserId: 23, Status: "ACTIVE", Version: 1, Changed: false}, nil
			}}
			app := newStatusApp(t, user, sdk, detail)
			res, err := app.Test(detailRequest())
			body := testutils.ResponseBody(t, res, err)
			if mode == "status only" {
				if res.StatusCode != 403 || !equalCalls(calls, []string{"Me"}) {
					t.Fatal(body, calls)
				}
			} else {
				var env struct{ Data struct{ Version string } }
				if res.StatusCode != 200 || json.Unmarshal([]byte(body), &env) != nil || env.Data.Version != "9007199254740993" {
					t.Fatal(body)
				}
			}
			req := statusRequest()
			res, err = app.Test(req)
			body = testutils.ResponseBody(t, res, err)
			want := 200
			if mode == "read only" {
				want = 403
			}
			if mode == "stale conflict" {
				want = 409
			}
			if res.StatusCode != want || len(res.Cookies()) != 0 {
				t.Fatal(res.StatusCode, body)
			}
			expected := []string{"Me", "GetUser", "Me", "ChangeUserStatus"}
			if mode == "read only" {
				expected = expected[:3]
			}
			if mode == "status only" {
				expected = []string{"Me", "Me", "ChangeUserStatus"}
			}
			if !equalCalls(calls, expected) {
				t.Fatal(calls, expected)
			}
			if mode == "replay" || mode == "revoked authority" || mode == "key conflict" {
				res, err = app.Test(statusRequest())
				body = testutils.ResponseBody(t, res, err)
				want = 200
				if mode == "revoked authority" {
					want = 403
				}
				if mode == "key conflict" {
					want = 409
				}
				expected = append(expected, "Me", "ChangeUserStatus")
				if res.StatusCode != want || commands != 2 || !equalCalls(calls, expected) || len(res.Cookies()) != 0 {
					t.Fatal(res.StatusCode, body, calls)
				}
			}
		})
	}
}
