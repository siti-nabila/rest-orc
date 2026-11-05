package test_scenarios

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	autherrors "github.com/siti-nabila/api-contracts/pkg/dictionary/auth"
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

const creationJSON = `{"email":" User.Name@Example.COM ","password":"simple","name":"  NASA  o'neil-jean  ","phone":"62628123456789","country_code":"62","address":" X\r\nY ","id":999,"user_id":999,"roles":["admin"],"permissions":{"iam.user.create":true},"must_change_password":false,"version":1,"field_mask":["password"],"confirm_password":123,"idempotency_key":"forged"}`

func creationRequest() *http.Request {
	r := httptest.NewRequest("POST", "/api/v1/users?email=forged&password=forged&version=bad&idempotency_key=forged", strings.NewReader(creationJSON))
	r.Header.Set("Authorization", authorizationValue)
	r.Header.Set("Idempotency-Key", "opaque, key")
	r.Header["Version"] = []string{"bad", "-1"}
	r.Header.Set(locale.HTTPHeader, "id-ID")
	r.Header.Set("X-Service-Token", "browser-secret")
	r.AddCookie(&http.Cookie{Name: "__Secure-refresh", Value: "cookie-secret"})
	return r
}
func creationApp(t *testing.T, user authservice.Service, sdk authservice.UserCreationService, before ...fiber.Handler) *fiber.App {
	t.Helper()
	c, err := authservice.NewWithUserCreationService(newTransport(t), user, &clientmocks.ProfileService{}, &clientmocks.PasswordService{}, &clientmocks.SessionService{}, &clientmocks.SelfSessionService{}, &clientmocks.AdminSessionService{}, &clientmocks.UserDetailService{}, &clientmocks.UserStatusService{}, &clientmocks.UserProfileService{}, sdk)
	if err != nil {
		t.Fatal(err)
	}
	return shared.NewApp(t, c, before...)
}
func creationSnapshot() *userv1.CreateUserResponse {
	u := shared.UserDetailResponse(23).User
	u.Email = "user.name@example.com"
	u.Status = "PENDING"
	return &userv1.CreateUserResponse{User: u}
}
func UserCreationScenarios() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "creation then explicit independent read profile and activation grants", Run: creationHTTPFollowUps},
		{Name: "creation HTTP ordered permission input and safe snapshot outcomes", Run: creationHTTPOutcomes},
		{Name: "creation encoded failures retain safe mapper and registry", Run: creationHTTPFailures},
		{Name: "creation retry fresh authority password-sensitive conflict and separate follow-up grants", Run: creationHTTPReplay},
	}
}
func creationHTTPOutcomes(t *testing.T) {
	for _, mode := range []string{"success", "null address", "absent address", "null status", "PENDING", "empty arrays", "max version", "restricted result", "missing bearer", "cookie only", "Me failure", "invalid Me", "restricted actor", "role only", "wildcard", "prefix", "false permission", "read only", "update only", "status only", "session only", "empty object", "null required", "unknown only", "empty address", "all invalid", "ACTIVE", "lowercase status", "empty status", "nil result", "nil user", "wrong email", "wrong status", "zero version", "invalid role", "preflight", "denied origin", "listusers"} {
		t.Run(mode, func(t *testing.T) {
			calls := []string{}
			want := 200
			expected := []string{"Me", "CreateUser"}
			user := &mocks.Service{MeFunc: func(c context.Context, _ *emptypb.Empty, _ ...grpc.CallOption) (*userv1.UserData, error) {
				calls = append(calls, "Me")
				assertAuthorizationMetadata(t, c)
				if mode == "Me failure" {
					return nil, common.ErrUnauthorized
				}
				u := shared.MeUserData()
				u.Permissions = map[string]bool{domain.UserCreatePermission: true}
				u.RoleCodes = nil
				u.RoleNames = nil
				switch mode {
				case "invalid Me":
					u.Id = 0
				case "restricted actor":
					u.MustChangePassword = true
				case "role only":
					u.Permissions = nil
					u.RoleCodes = []int32{2}
					u.RoleNames = []string{"admin"}
				case "wildcard":
					u.Permissions = map[string]bool{"iam.user.*": true}
				case "prefix":
					u.Permissions = map[string]bool{"iam.user": true}
				case "false permission":
					u.Permissions[domain.UserCreatePermission] = false
				case "read only":
					u.Permissions = map[string]bool{domain.UserReadPermission: true}
				case "update only":
					u.Permissions = map[string]bool{domain.UserUpdatePermission: true}
				case "status only":
					u.Permissions = map[string]bool{domain.UserStatusUpdatePermission: true}
				case "session only":
					u.Permissions = map[string]bool{domain.SessionRevokePermission: true}
				}
				return u, nil
			}}
			sdk := &clientmocks.UserCreationService{Create: func(c context.Context, r *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
				calls = append(calls, "CreateUser")
				assertAuthorizationMetadata(t, c)
				md, _ := metadata.FromOutgoingContext(c)
				address := " X\r\nY "
				if mode == "null address" || mode == "absent address" {
					address = ""
				}
				if r.Email != " User.Name@Example.COM " || r.Password != "simple" || r.Name != "NASA O'Neil-Jean" || r.Phone != "628123456789" || r.Address != address || r.CountryCode == nil || *r.CountryCode != "+62" || r.Status == nil || *r.Status != "PENDING" || len(md.Get("version")) != 0 || !reflect.DeepEqual(md.Get("idempotency-key"), []string{"opaque, key"}) || len(md.Get("x-service-token")) != 0 || md.Get("x-request-id")[0] != "creation-http" {
					t.Error("payload metadata")
				}
				result := creationSnapshot()
				switch mode {
				case "nil result":
					return nil, nil
				case "nil user":
					result.User = nil
				case "wrong email":
					result.User.Email = "other@example.com"
				case "wrong status":
					result.User.Status = "ACTIVE"
				case "zero version":
					result.User.Version = 0
				case "invalid role":
					result.User.Roles[0] = nil
				case "empty arrays":
					result.User.Roles = nil
					result.User.ApplicationAccess = nil
				case "max version":
					result.User.Version = 9223372036854775807
				case "restricted result":
					result.User.MustChangePassword = true
				}
				return result, nil
			}}
			md := metadata.Pairs("version", "old", "idempotency-key", "old", "x-request-id", "creation-http")
			parent := metadata.NewOutgoingContext(context.Background(), md)
			app := creationApp(t, user, sdk, func(c fiber.Ctx) error { c.SetContext(parent); return c.Next() })
			req := creationRequest()
			var obj map[string]any
			_ = json.Unmarshal([]byte(creationJSON), &obj)
			switch mode {
			case "null address":
				obj["address"] = nil
			case "absent address":
				delete(obj, "address")
			case "empty address":
				obj["address"] = ""
			case "null status":
				obj["status"] = nil
			case "PENDING", "ACTIVE":
				obj["status"] = mode
			case "lowercase status":
				obj["status"] = "pending"
			case "empty status":
				obj["status"] = ""
			case "empty object":
				obj = map[string]any{}
			case "unknown only":
				obj = map[string]any{"roles": []string{"admin"}}
			case "null required":
				for _, f := range []string{"email", "password", "name", "phone", "country_code"} {
					obj[f] = nil
				}
			case "all invalid":
				for _, f := range []string{"email", "password", "name", "phone", "country_code", "address", "status"} {
					obj[f] = "\tsecret-invalid"
				}
			}
			raw, _ := json.Marshal(obj)
			req.Body = io.NopCloser(strings.NewReader(string(raw)))
			req.ContentLength = int64(len(raw))
			switch mode {
			case "missing bearer", "cookie only":
				req.Header.Del("Authorization")
				want = 401
				expected = nil
			case "Me failure", "invalid Me":
				want = 401
				expected = []string{"Me"}
			case "restricted actor", "role only", "wildcard", "prefix", "false permission", "read only", "update only", "status only", "session only":
				want = 403
				expected = []string{"Me"}
			case "empty object", "null required", "unknown only", "empty address", "all invalid", "ACTIVE", "lowercase status", "empty status":
				want = 400
				expected = []string{"Me"}
			case "nil result", "nil user", "wrong email", "wrong status", "zero version", "invalid role":
				want = 500
			case "preflight":
				req.Method = "OPTIONS"
				req.Header.Set("Origin", shared.BrowserOrigin)
				req.Header.Set("Access-Control-Request-Method", "POST")
				want = 204
				expected = nil
			case "denied origin":
				req.Header.Set("Origin", "https://evil.example.test")
				want = 403
				expected = nil
			case "listusers":
				req.Method = "GET"
				req.URL.RawQuery = ""
				want = 403
				expected = []string{"Me"}
			}
			res, err := app.Test(req)
			body := testutils.ResponseBody(t, res, err)
			if res.StatusCode != want || !equalCalls(calls, expected) || len(res.Cookies()) != 0 || (mode != "listusers" && res.Header.Get("Cache-Control") != "no-store") {
				t.Fatal(mode, res.StatusCode, want, body, calls, expected)
			}
			if want == 200 {
				var env struct {
					Code    string
					Message string
					Data    map[string]json.RawMessage
				}
				if json.Unmarshal([]byte(body), &env) != nil || env.Code != "SS" || env.Message != "User created successfully." || len(env.Data) != 1 {
					t.Fatal(body)
				}
				var got map[string]json.RawMessage
				_ = json.Unmarshal(env.Data["user"], &got)
				if len(got) != 12 || string(got["email"]) != `"user.name@example.com"` || string(got["status"]) != `"PENDING"` || string(got["id"]) != "23" {
					t.Fatal(body)
				}
				if mode == "empty arrays" && (string(got["roles"]) != "[]" || string(got["application_access"]) != "[]") {
					t.Fatal(body)
				}
			}
			if strings.Contains(body, "simple") || strings.Contains(body, "secret-invalid") || strings.Contains(body, "private-password") || strings.Contains(body, "private-dob") || strings.Contains(body, "cookie-secret") {
				t.Fatal("secret response")
			}
			actual, _ := metadata.FromOutgoingContext(parent)
			if !reflect.DeepEqual(actual, md) {
				t.Fatal("parent mutated")
			}
		})
	}
}
func creationHTTPReplay(t *testing.T) {
	for _, mode := range []string{"replay", "changed password", "changed profile", "backend revoked", "Me revoked"} {
		t.Run(mode, func(t *testing.T) {
			calls := []string{}
			mes, commands := 0, 0
			user := &mocks.Service{MeFunc: func(context.Context, *emptypb.Empty, ...grpc.CallOption) (*userv1.UserData, error) {
				calls = append(calls, "Me")
				mes++
				u := shared.MeUserData()
				u.Permissions = map[string]bool{domain.UserCreatePermission: !(mode == "Me revoked" && mes == 2)}
				return u, nil
			}}
			sdk := &clientmocks.UserCreationService{Create: func(c context.Context, r *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
				calls = append(calls, "CreateUser")
				commands++
				md, _ := metadata.FromOutgoingContext(c)
				if !reflect.DeepEqual(md.Get("idempotency-key"), []string{"opaque, key"}) {
					t.Fatal("key replacement")
				}
				if commands == 2 {
					if mode == "backend revoked" {
						return nil, common.ErrForbidden
					}
					if mode == "changed password" || mode == "changed profile" {
						if mode == "changed password" && r.Password != "changed-secret" || mode == "changed profile" && r.Name != "Different Person" {
							t.Fatal("changed input lost")
						}
						return nil, grpcerror.NewEncoder(mapping.UserCodeMappings()...).Encode(usererrors.ErrIdempotencyConflict, locale.Indonesian)
					}
				}
				return creationSnapshot(), nil
			}}
			app := creationApp(t, user, sdk)
			first := ""
			for i := 0; i < 2; i++ {
				r := creationRequest()
				if i == 1 && (mode == "changed password" || mode == "changed profile") {
					raw := creationJSON
					if mode == "changed password" {
						raw = strings.Replace(raw, "simple", "changed-secret", 1)
					} else {
						raw = strings.Replace(raw, "  NASA  o'neil-jean  ", "different person", 1)
					}
					r.Body = io.NopCloser(strings.NewReader(raw))
					r.ContentLength = int64(len(raw))
				}
				res, err := app.Test(r)
				body := testutils.ResponseBody(t, res, err)
				want := 200
				if i == 1 {
					if strings.HasPrefix(mode, "changed ") {
						want = 409
					}
					if strings.HasSuffix(mode, "revoked") {
						want = 403
					}
				}
				if res.StatusCode != want || len(res.Cookies()) != 0 || res.Header.Get("Cache-Control") != "no-store" {
					t.Fatal(body, calls)
				}
				if i == 0 {
					first = body
				}
				if i == 1 && mode == "replay" && body != first {
					t.Fatal("snapshot replay changed")
				}
			}
			expected := []string{"Me", "CreateUser", "Me", "CreateUser"}
			if mode == "Me revoked" {
				expected = expected[:3]
			}
			if !equalCalls(calls, expected) {
				t.Fatal(calls)
			}
			// Creation grant alone never satisfies later resource grants.
			for _, tc := range []struct{ method, path, raw string }{{"GET", "/api/v1/users/23", ""}, {"PATCH", "/api/v1/users/23/profile", `{"name":"Alice"}`}, {"PATCH", "/api/v1/users/23/status", `{"status":"ACTIVE","reason":"activation"}`}, {"GET", "/api/v1/users/23/sessions", ""}} {
				r := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.raw))
				r.Header.Set("Authorization", authorizationValue)
				r.Header.Set("Version", "1")
				r.Header.Set("Idempotency-Key", "follow-up")
				before := len(calls)
				res, err := app.Test(r)
				body := testutils.ResponseBody(t, res, err)
				if res.StatusCode != 403 || len(calls) != before+1 || calls[len(calls)-1] != "Me" || len(res.Cookies()) != 0 {
					t.Fatal("follow-up grant", body, calls)
				}
			}
		})
	}
}

func creationHTTPFailures(t *testing.T) {
	encoder := grpcerror.NewEncoder(mapping.UserCodeMappings()...)
	cases := []struct {
		name string
		err  error
		code string
		http int
	}{
		{"duplicate email", grpcerror.NewEncoder(mapping.AuthCodeMappings()...).Encode(autherrors.ErrDataExists, locale.Indonesian), "EX", 409},
		{"idempotency", encoder.Encode(usererrors.ErrIdempotencyConflict, locale.Indonesian), "IC", 409},
		{"busy", encoder.Encode(usererrors.ErrResourceBusy, locale.Indonesian), "RB", 409},
		{"unauthorized", common.ErrUnauthorized, "", 401}, {"forbidden", status.Error(codes.PermissionDenied, "private-sentinel"), "", 403},
		{"not found", status.Error(codes.NotFound, "private-sentinel"), "", 404}, {"rate", testutils.ServiceFieldError(t, 429), "EMAIL_USED", 429}, {"unavailable", testutils.ServiceFieldError(t, 503), "EMAIL_USED", 503},
		{"timeout", context.DeadlineExceeded, "", 504}, {"cancel", context.Canceled, "", 499}, {"unknown", status.Error(codes.Internal, "private-sentinel"), "", 500},
	}
	for _, stage := range []string{"Me", "CreateUser"} {
		for _, tc := range cases {
			t.Run(stage+"/"+tc.name, func(t *testing.T) {
				calls := []string{}
				user := &mocks.Service{MeFunc: func(context.Context, *emptypb.Empty, ...grpc.CallOption) (*userv1.UserData, error) {
					calls = append(calls, "Me")
					if stage == "Me" {
						return nil, tc.err
					}
					u := shared.MeUserData()
					u.Permissions = map[string]bool{domain.UserCreatePermission: true}
					return u, nil
				}}
				sdk := &clientmocks.UserCreationService{Create: func(context.Context, *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
					calls = append(calls, "CreateUser")
					return &userv1.CreateUserResponse{User: creationSnapshot().User}, tc.err
				}}
				app := creationApp(t, user, sdk)
				res, err := app.Test(creationRequest())
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

// Follow-up commands are explicit requests with independently enabled grants.
func creationHTTPFollowUps(t *testing.T) {
	calls := []string{}
	permissions := map[string]bool{domain.UserCreatePermission: true}
	user := &mocks.Service{MeFunc: func(context.Context, *emptypb.Empty, ...grpc.CallOption) (*userv1.UserData, error) {
		calls = append(calls, "Me")
		u := shared.MeUserData()
		u.Permissions = permissions
		return u, nil
	}}
	create := &clientmocks.UserCreationService{Create: func(context.Context, *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
		calls = append(calls, "CreateUser")
		return creationSnapshot(), nil
	}}
	detail := &clientmocks.UserDetailService{Get: func(_ context.Context, r *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
		calls = append(calls, "GetUser")
		if r.Id != 23 {
			t.Fatal("created ID lost")
		}
		return &userv1.GetUserResponse{User: creationSnapshot().User}, nil
	}}
	profile := &clientmocks.UserProfileService{Update: func(_ context.Context, r *userv1.UpdateUserRequest) (*userv1.UpdateUserResponse, error) {
		calls = append(calls, "UpdateUser")
		if r.UserId != 23 || r.GetExpectedUserVersion() != 9007199254740993 || r.Name == nil || *r.Name != "New Name" {
			t.Fatal("follow-up profile")
		}
		u := creationSnapshot().User
		u.Version++
		return &userv1.UpdateUserResponse{User: u, Changed: true}, nil
	}}
	statusSDK := &clientmocks.UserStatusService{Change: func(_ context.Context, r *userv1.ChangeUserStatusRequest) (*userv1.ChangeUserStatusResponse, error) {
		calls = append(calls, "ChangeUserStatus")
		if r.UserId != 23 || r.GetExpectedUserVersion() != 9007199254740994 || r.GetStatus() != "ACTIVE" {
			t.Fatal("follow-up activation")
		}
		return &userv1.ChangeUserStatusResponse{UserId: 23, Status: "ACTIVE", Version: 9007199254740995, Changed: true}, nil
	}}
	client, err := authservice.NewWithUserCreationService(newTransport(t), user, &clientmocks.ProfileService{}, &clientmocks.PasswordService{}, &clientmocks.SessionService{}, &clientmocks.SelfSessionService{}, &clientmocks.AdminSessionService{}, detail, statusSDK, profile, create)
	if err != nil {
		t.Fatal(err)
	}
	app := shared.NewApp(t, client)
	res, err := app.Test(creationRequest())
	body := testutils.ResponseBody(t, res, err)
	if res.StatusCode != 200 || !equalCalls(calls, []string{"Me", "CreateUser"}) {
		t.Fatal(body, calls)
	}
	for _, step := range []struct{ grant, method, path, raw, version, command string }{
		{domain.UserReadPermission, "GET", "/api/v1/users/23", "", "", "GetUser"},
		{domain.UserUpdatePermission, "PATCH", "/api/v1/users/23/profile", `{"name":" new  name "}`, "9007199254740993", "UpdateUser"},
		{domain.UserStatusUpdatePermission, "PATCH", "/api/v1/users/23/status", `{"status":"ACTIVE","reason":"approved"}`, "9007199254740994", "ChangeUserStatus"},
	} {
		permissions = map[string]bool{step.grant: true}
		r := httptest.NewRequest(step.method, step.path, strings.NewReader(step.raw))
		r.Header.Set("Authorization", authorizationValue)
		r.Header.Set("Version", step.version)
		r.Header.Set("Idempotency-Key", "follow-up")
		before := len(calls)
		res, err = app.Test(r)
		body = testutils.ResponseBody(t, res, err)
		if res.StatusCode != 200 || len(res.Cookies()) != 0 || !equalCalls(calls[before:], []string{"Me", step.command}) {
			t.Fatal(body, calls)
		}
	}
}
