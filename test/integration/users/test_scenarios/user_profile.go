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

const targetProfileJSON = `{"name":"  NASA  o'neil-jean  ","address":" X\r\nY ","country_code":"62","phone":"0812345678901","user_id":99,"version":1,"email":"forged","status":"DEACTIVATED","field_mask":["password"]}`

func targetProfileRequest() *http.Request {
	req := httptest.NewRequest("PATCH", "/api/v1/users/00023/profile?user_id=99&version=1&name=forged&idempotency_key=forged", strings.NewReader(targetProfileJSON))
	req.Header.Set("Authorization", authorizationValue)
	req.Header.Set("Version", "9007199254740993")
	req.Header.Set("Idempotency-Key", "opaque, key")
	req.Header.Set(locale.HTTPHeader, "id-ID")
	req.Header.Set("X-Service-Token", "browser-secret")
	req.AddCookie(&http.Cookie{Name: "__Secure-refresh", Value: "browser-refresh"})
	return req
}
func setTargetProfileBody(req *http.Request, raw string) {
	req.Body = io.NopCloser(strings.NewReader(raw))
	req.ContentLength = int64(len(raw))
}
func newTargetProfileApp(t *testing.T, user authservice.Service, sdk authservice.UserProfileService, detail authservice.UserDetailService, before ...fiber.Handler) *fiber.App {
	t.Helper()
	if detail == nil {
		detail = &clientmocks.UserDetailService{}
	}
	client, err := authservice.NewWithUserProfileService(newTransport(t), user, &clientmocks.ProfileService{}, &clientmocks.PasswordService{}, &clientmocks.SessionService{}, &clientmocks.SelfSessionService{}, &clientmocks.AdminSessionService{}, detail, &clientmocks.UserStatusService{}, sdk)
	if err != nil {
		t.Fatal(err)
	}
	return shared.NewApp(t, client, before...)
}
func UserProfileScenarios() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "target profile HTTP eligibility inputs no-op and safe results", Run: targetProfileHTTPOutcomes},
		{Name: "target profile HTTP encoded conflicts and safe error mapping", Run: targetProfileHTTPFailures},
		{Name: "target profile separate detail version optional replay and current authority", Run: targetProfileHTTPFlow},
	}
}
func targetProfileHTTPOutcomes(t *testing.T) {
	modes := []string{"success", "changed", "unkeyed", "lowercase headers", "self", "prefix preserved", "leading zero version", "max version", "no growth", "target restricted", "PENDING", "SUSPENDED", "DEACTIVATED", "empty arrays", "local null", "local object", "local null fields", "local unknown fields", "unverified local target", "restricted no-op", "read-only no-op", "missing bearer", "cookie only", "Me failure", "invalid Me", "restricted actor", "role only", "wildcard", "prefix", "false permission", "read only", "status only", "session only", "invalid target", "literal me", "empty body", "malformed", "array", "scalar", "wrong type", "empty name", "all invalid", "missing version", "duplicate version", "case duplicate version", "absent key no-op", "empty key", "duplicate key", "case duplicate key", "invalid key no-op", "missing version no-op", "nil result", "nil user", "wrong target", "zero version", "overflow version", "bad status", "missing timestamp", "nil role", "preflight", "bad preflight target", "denied origin", "POST alias", "old PATCH alias", "unversioned alias", "listusers"}
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			calls := []string{}
			target, version := uint64(23), uint64(9007199254740993)
			want := 200
			expected := []string{"Me", "UpdateUser"}
			noOp := strings.HasPrefix(mode, "local ") || mode == "unverified local target" || mode == "absent key no-op" || mode == "restricted no-op" || mode == "read-only no-op" || mode == "invalid key no-op" || mode == "missing version no-op"
			if mode == "self" {
				target = 17
			}
			if mode == "unverified local target" {
				target = 2147483647
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
				u.Fullname = "NASA O'Neil-Jean"
				u.Permissions = map[string]bool{domain.UserUpdatePermission: true}
				switch mode {
				case "invalid Me":
					u.Id = 0
				case "restricted actor", "restricted no-op":
					u.MustChangePassword = true
				case "role only":
					u.RoleCodes = []int32{2}
					u.RoleNames = []string{"admin"}
					u.Permissions = nil
				case "wildcard":
					u.Permissions = map[string]bool{"iam.user.*": true}
				case "prefix":
					u.Permissions = map[string]bool{"iam.user": true}
				case "false permission":
					u.Permissions[domain.UserUpdatePermission] = false
				case "read only", "read-only no-op":
					u.Permissions = map[string]bool{domain.UserReadPermission: true}
				case "status only":
					u.Permissions = map[string]bool{domain.UserStatusUpdatePermission: true}
				case "session only":
					u.Permissions = map[string]bool{domain.SessionRevokePermission: true}
				}
				return u, nil
			}}
			sdk := &clientmocks.UserProfileService{Update: func(c context.Context, r *userv1.UpdateUserRequest) (*userv1.UpdateUserResponse, error) {
				calls = append(calls, "UpdateUser")
				assertAuthorizationMetadata(t, c)
				md, _ := metadata.FromOutgoingContext(c)
				keyWant := []string{"opaque, key"}
				if mode == "unkeyed" {
					keyWant = nil
				}
				phone := "812345678901"
				if mode == "prefix preserved" {
					phone = "628123456789"
				}
				if r.UserId != target || r.ExpectedUserVersion == nil || *r.ExpectedUserVersion != version || r.Name == nil || *r.Name != "NASA O'Neil-Jean" || r.Address == nil || *r.Address != " X\r\nY " || r.Phone == nil || *r.Phone != phone || r.CountryCode == nil || *r.CountryCode != "+62" || r.FieldMask == nil || !reflect.DeepEqual(r.FieldMask.Paths, []string{"name", "address", "phone", "country_code"}) || !reflect.DeepEqual(md.Get("idempotency-key"), keyWant) || len(md.Get("version")) != 0 || md.Get("x-request-id")[0] != "target-profile-http" || len(md.Get("x-service-token")) != 0 || len(md.Get("namespace")) != 0 {
					t.Error(r, md)
				}
				result := &userv1.UpdateUserResponse{User: shared.UserDetailResponse(target).User, Changed: mode == "changed"}
				result.User.Version = version
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
				case "bad status":
					result.User.Status = "unknown"
				case "missing timestamp":
					result.User.CreatedAt = nil
				case "nil role":
					result.User.Roles[0] = nil
				case "no growth":
					result.User.Version = 1
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
			md := metadata.Pairs("version", "inherited", "idempotency-key", "inherited", "x-request-id", "target-profile-http")
			parent := metadata.NewOutgoingContext(context.Background(), md)
			app := newTargetProfileApp(t, user, sdk, nil, func(c fiber.Ctx) error { c.SetContext(parent); return c.Next() })
			req := targetProfileRequest()
			if noOp {
				setTargetProfileBody(req, "{}")
				expected = []string{"Me"}
			}
			switch mode {
			case "self":
				req.URL.Path = "/api/v1/users/17/profile"
			case "unkeyed", "absent key no-op":
				req.Header.Del("Idempotency-Key")
			case "lowercase headers":
				req.Header.Del("Version")
				req.Header.Del("Idempotency-Key")
				req.Header["version"] = []string{"9007199254740993"}
				req.Header["idempotency-key"] = []string{"opaque, key"}
			case "prefix preserved":
				setTargetProfileBody(req, strings.Replace(targetProfileJSON, "0812345678901", "62628123456789", 1))
			case "leading zero version":
				req.Header.Set("Version", "0009007199254740993")
			case "max version":
				req.Header.Set("Version", "9223372036854775807")
			case "unverified local target":
				req.URL.Path = "/api/v1/users/2147483647/profile"
			case "local null":
				setTargetProfileBody(req, "null")
			case "local null fields":
				setTargetProfileBody(req, `{"name":null,"phone":null,"address":null,"country_code":null}`)
			case "local unknown fields":
				setTargetProfileBody(req, `{"user_id":99,"version":1,"field_mask":["password"],"email":"forged","status":"ACTIVE"}`)
			case "missing bearer", "cookie only":
				req.Header.Del("Authorization")
				want = 401
				expected = nil
			case "Me failure", "invalid Me":
				want = 401
				expected = []string{"Me"}
			case "restricted actor", "restricted no-op", "read-only no-op", "role only", "wildcard", "prefix", "false permission", "read only", "status only", "session only":
				want = 403
				expected = []string{"Me"}
			case "invalid target", "literal me":
				req.URL.Path = "/api/v1/users/2147483648/profile"
				if mode == "literal me" {
					req.URL.Path = "/api/v1/users/me/profile"
				}
				want = 400
				expected = []string{"Me"}
			case "empty body", "malformed", "array", "scalar", "wrong type", "empty name", "all invalid":
				raw := map[string]string{"malformed": "{", "array": "[]", "scalar": "true", "wrong type": `{"name":123}`, "empty name": `{"name":""}`, "all invalid": `{"name":"!","phone":"x","address":"é","country_code":"0"}`}[mode]
				setTargetProfileBody(req, raw)
				want = 400
				expected = []string{"Me"}
			case "missing version", "missing version no-op", "duplicate version", "case duplicate version", "empty key", "duplicate key", "case duplicate key", "invalid key no-op":
				want = 400
				expected = []string{"Me"}
				switch mode {
				case "missing version", "missing version no-op":
					req.Header.Del("Version")
				case "duplicate version":
					req.Header.Add("Version", "9007199254740993")
				case "case duplicate version":
					req.Header["version"] = []string{"1"}
				case "empty key":
					req.Header.Set("Idempotency-Key", "")
				case "duplicate key":
					req.Header.Add("Idempotency-Key", "opaque, key")
				case "case duplicate key":
					req.Header["idempotency-key"] = []string{"opaque, key"}
				case "invalid key no-op":
					req.Header.Set("Idempotency-Key", "é")
				}
			case "nil result", "nil user", "wrong target", "zero version", "overflow version", "bad status", "missing timestamp", "nil role":
				want = 500
			case "preflight", "bad preflight target":
				req.Method = "OPTIONS"
				req.Header.Set("Origin", shared.BrowserOrigin)
				req.Header.Set("Access-Control-Request-Method", "PATCH")
				want = 204
				expected = nil
				if mode == "bad preflight target" {
					req.URL.Path = "/api/v1/users/bad/profile"
				}
			case "denied origin":
				req.Header.Set("Origin", "https://evil.example.test")
				want = 403
				expected = nil
			case "POST alias", "old PATCH alias", "listusers":
				want = 403
				expected = []string{"Me"}
				if mode == "POST alias" {
					req.Method = "POST"
				}
				if mode == "old PATCH alias" {
					req.URL.Path = "/api/v1/users/23"
				}
				if mode == "listusers" {
					req.Method = "GET"
					req.URL.Path = "/api/v1/users"
					req.URL.RawQuery = ""
				}
			case "unversioned alias":
				req.URL.Path = "/users/23/profile"
				want = 404
				expected = nil
			}
			res, err := app.Test(req)
			body := testutils.ResponseBody(t, res, err)
			if res.StatusCode != want || !equalCalls(calls, expected) || len(res.Cookies()) != 0 {
				t.Fatal(res.StatusCode, want, body, calls, expected)
			}
			if mode != "unversioned alias" && mode != "listusers" && res.Header.Get("Cache-Control") != "no-store" {
				t.Fatal("no-store")
			}
			if want == 200 {
				if noOp {
					if body != `{"code":"SS","message":"User profile updated successfully.","data":{"changed":false}}` {
						t.Fatal(body)
					}
				} else {
					var env struct{ Data map[string]json.RawMessage }
					if json.Unmarshal([]byte(body), &env) != nil || len(env.Data) != 2 || string(env.Data["changed"]) != fmt.Sprint(mode == "changed") {
						t.Fatal(body)
					}
					var got map[string]json.RawMessage
					if json.Unmarshal(env.Data["user"], &got) != nil || len(got) != 12 {
						t.Fatal(body)
					}
					wantUser := strings.TrimSuffix(strings.SplitN(shared.UserDetailBody, `"data":`, 2)[1], "}")
					wantUser = strings.Replace(wantUser, `"id":23`, fmt.Sprintf(`"id":%d`, target), 1)
					wantUser = strings.Replace(wantUser, `"version":"9007199254740993"`, fmt.Sprintf(`"version":"%d"`, version), 1)
					switch mode {
					case "no growth":
						wantUser = strings.Replace(wantUser, `"version":"9007199254740993"`, `"version":"1"`, 1)
					case "target restricted":
						wantUser = strings.Replace(wantUser, `"must_change_password":false`, `"must_change_password":true`, 1)
					case "PENDING", "SUSPENDED", "DEACTIVATED":
						wantUser = strings.Replace(wantUser, `"status":"ACTIVE"`, `"status":"`+mode+`"`, 1)
					}
					if mode == "empty arrays" {
						if string(got["roles"]) != "[]" || string(got["application_access"]) != "[]" || string(got["must_change_password"]) != "false" {
							t.Fatal(body)
						}
					} else if string(env.Data["user"]) != wantUser {
						t.Fatal(body, wantUser)
					}
				}
			} else if want != 204 && !strings.Contains(body, `"data":[]`) {
				t.Fatal(body)
			}
			if mode == "all invalid" {
				var env struct{ Errors map[string]json.RawMessage }
				_ = json.Unmarshal([]byte(body), &env)
				if len(env.Errors) != 4 {
					t.Fatal(body)
				}
			}
			for _, secret := range []string{"browser-secret", "browser-refresh", "private-password", "private-dob"} {
				if strings.Contains(body, secret) {
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
func targetProfileHTTPFailures(t *testing.T) {
	encoder := grpcerror.NewEncoder(mapping.UserCodeMappings()...)
	cases := []struct {
		name string
		err  error
		code string
		http int
	}{
		{"version", encoder.Encode(usererrors.ErrVersionConflict, locale.Indonesian), "VC", 409},
		{"idempotency", encoder.Encode(usererrors.ErrIdempotencyConflict, locale.Indonesian), "IC", 409},
		{"busy", encoder.Encode(usererrors.ErrResourceBusy, locale.Indonesian), "RB", 409},
		{"unauthorized", common.ErrUnauthorized, "", 401}, {"forbidden", status.Error(codes.PermissionDenied, "private-sentinel"), "", 403},
		{"not found", status.Error(codes.NotFound, "private-sentinel"), "", 404}, {"rate", testutils.ServiceFieldError(t, 429), "EMAIL_USED", 429}, {"unavailable", testutils.ServiceFieldError(t, 503), "EMAIL_USED", 503},
		{"timeout", context.DeadlineExceeded, "", 504}, {"cancel", context.Canceled, "", 499}, {"unknown", status.Error(codes.Internal, "private-sentinel"), "", 500},
	}
	for _, stage := range []string{"Me", "UpdateUser"} {
		for _, tc := range cases {
			t.Run(stage+"/"+tc.name, func(t *testing.T) {
				calls := []string{}
				user := &mocks.Service{MeFunc: func(context.Context, *emptypb.Empty, ...grpc.CallOption) (*userv1.UserData, error) {
					calls = append(calls, "Me")
					if stage == "Me" {
						return nil, tc.err
					}
					u := shared.MeUserData()
					u.Permissions = map[string]bool{domain.UserUpdatePermission: true}
					return u, nil
				}}
				sdk := &clientmocks.UserProfileService{Update: func(context.Context, *userv1.UpdateUserRequest) (*userv1.UpdateUserResponse, error) {
					calls = append(calls, "UpdateUser")
					return &userv1.UpdateUserResponse{User: shared.UserDetailResponse(23).User}, tc.err
				}}
				app := newTargetProfileApp(t, user, sdk, nil)
				res, err := app.Test(targetProfileRequest())
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

func targetProfileHTTPFlow(t *testing.T) {
	for _, mode := range []string{"success", "stale conflict", "read only", "update only", "replay", "revoked authority", "key conflict", "unkeyed repeat", "local then command"} {
		t.Run(mode, func(t *testing.T) {
			calls := []string{}
			commands := 0
			user := &mocks.Service{MeFunc: func(context.Context, *emptypb.Empty, ...grpc.CallOption) (*userv1.UserData, error) {
				calls = append(calls, "Me")
				u := shared.MeUserData()
				u.Version = 1
				u.Permissions = map[string]bool{domain.UserReadPermission: mode != "update only", domain.UserUpdatePermission: mode != "read only"}
				return u, nil
			}}
			detail := &clientmocks.UserDetailService{Get: func(_ context.Context, r *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
				calls = append(calls, "GetUser")
				return shared.UserDetailResponse(r.Id), nil
			}}
			sdk := &clientmocks.UserProfileService{Update: func(c context.Context, r *userv1.UpdateUserRequest) (*userv1.UpdateUserResponse, error) {
				commands++
				calls = append(calls, "UpdateUser")
				md, _ := metadata.FromOutgoingContext(c)
				if r.UserId != 23 || r.ExpectedUserVersion == nil || *r.ExpectedUserVersion != 9007199254740993 || r.Name == nil || *r.Name != "NASA O'Neil-Jean" {
					t.Fatal(r)
				}
				if mode == "unkeyed repeat" {
					if len(md.Get("idempotency-key")) != 0 {
						t.Fatal("inherited key")
					}
				} else if !reflect.DeepEqual(md.Get("idempotency-key"), []string{"opaque, key"}) {
					t.Fatal(md)
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
				u := shared.UserDetailResponse(23).User
				u.Version = 1
				if mode == "unkeyed repeat" {
					u.Version = uint64(commands)
				}
				return &userv1.UpdateUserResponse{User: u, Changed: false}, nil
			}}
			app := newTargetProfileApp(t, user, sdk, detail)
			res, err := app.Test(detailRequest())
			body := testutils.ResponseBody(t, res, err)
			if mode == "update only" {
				if res.StatusCode != 403 || !equalCalls(calls, []string{"Me"}) {
					t.Fatal(body, calls)
				}
			} else {
				var env struct{ Data struct{ Version string } }
				if res.StatusCode != 200 || json.Unmarshal([]byte(body), &env) != nil || env.Data.Version != "9007199254740993" {
					t.Fatal(body)
				}
			}
			if mode == "local then command" {
				req := targetProfileRequest()
				setTargetProfileBody(req, "{}")
				res, err = app.Test(req)
				body = testutils.ResponseBody(t, res, err)
				if res.StatusCode != 200 || commands != 0 || !strings.Contains(body, `"data":{"changed":false}`) {
					t.Fatal(body, calls)
				}
			}
			req := targetProfileRequest()
			if mode == "unkeyed repeat" {
				req.Header.Del("Idempotency-Key")
			}
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
			expected := []string{"Me", "GetUser", "Me", "UpdateUser"}
			if mode == "read only" {
				expected = expected[:3]
			}
			if mode == "update only" {
				expected = []string{"Me", "Me", "UpdateUser"}
			}
			if mode == "local then command" {
				expected = []string{"Me", "GetUser", "Me", "Me", "UpdateUser"}
			}
			if !equalCalls(calls, expected) {
				t.Fatal(calls, expected)
			}
			if mode == "replay" || mode == "revoked authority" || mode == "key conflict" || mode == "unkeyed repeat" {
				req = targetProfileRequest()
				if mode == "unkeyed repeat" {
					req.Header.Del("Idempotency-Key")
				}
				res, err = app.Test(req)
				body = testutils.ResponseBody(t, res, err)
				want = 200
				if mode == "revoked authority" {
					want = 403
				}
				if mode == "key conflict" {
					want = 409
				}
				expected = append(expected, "Me", "UpdateUser")
				if res.StatusCode != want || commands != 2 || !equalCalls(calls, expected) || len(res.Cookies()) != 0 {
					t.Fatal(res.StatusCode, body, calls)
				}
				if mode == "unkeyed repeat" && !strings.Contains(body, `"version":"2"`) {
					t.Fatal(body)
				}
			}
		})
	}
}
