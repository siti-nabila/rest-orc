package test_scenarios

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	apperrors "github.com/siti-nabila/api-contracts/pkg/dictionary/application"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	roleerrors "github.com/siti-nabila/api-contracts/pkg/dictionary/role"
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

func UserRoleScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "assignment HTTP exact contracts permission boundaries inputs and browser safety", Run: assignmentHTTPOutcomes}, {Name: "assignment HTTP encoded errors shared fallback fields retry and contexts", Run: assignmentHTTPFailures}, {Name: "assignment separate read command replay and current authority", Run: assignmentHTTPFlow}}
}
func assignmentPermission(op string) string {
	if op == "grant" {
		return domain.UserRoleGrantPermission
	}
	if op == "revoke" {
		return domain.UserRoleRevokePermission
	}
	return domain.UserRoleReadPermission
}
func assignmentRequest(op string) *http.Request {
	method, path := "GET", "/api/v1/users/00023/roles"
	raw := "{"
	if op != "list" {
		method = "POST"
		path += "/" + op
		raw = `{"role_ids":["11","00010"],"reason":"  Approved\r\nreason  ","user_id":99,"version":1}`
	}
	path += "?user_id=99&namespace=forged&version=1"
	if op == "list" {
		path += "&page=001&limit=020&keyword=legal%09reviewer&application_code=clm&statuses=REVOKED,ACTIVE,ACTIVE&last_id=opaque%2B%2F%3D%25"
	}
	req := httptest.NewRequest(method, path, strings.NewReader(raw))
	req.Header.Set("Authorization", authorizationValue)
	req.Header.Set("Version", "9007199254740993")
	req.Header.Set("Idempotency-Key", "opaque, key")
	req.Header.Set(locale.HTTPHeader, "id-ID")
	req.Header.Set("X-Service-Token", "browser-secret")
	req.AddCookie(&http.Cookie{Name: "__Secure-refresh", Value: "browser-refresh"})
	return req
}
func newAssignmentApp(t *testing.T, user authservice.Service, sdk authservice.UserRoleManagementService, before ...fiber.Handler) *fiber.App {
	t.Helper()
	c, e := authservice.NewWithUserRoleManagementService(newTransport(t), user, &clientmocks.ProfileService{}, &clientmocks.PasswordService{}, &clientmocks.SessionService{}, &clientmocks.SelfSessionService{}, &clientmocks.AdminSessionService{}, &clientmocks.UserDetailService{}, &clientmocks.UserStatusService{}, &clientmocks.UserProfileService{}, &clientmocks.UserCreationService{}, &clientmocks.RoleCatalogService{}, sdk)
	if e != nil {
		t.Fatal(e)
	}
	return shared.NewApp(t, c, before...)
}
func assignmentHTTPOutcomes(t *testing.T) {
	for _, op := range []string{"list", "grant", "revoke"} {
		for _, mode := range []string{"success", "no-op", "replay", "self", "max version", "empty list", "bad result", "missing bearer", "cookie only", "Me failure", "invalid Me", "restricted", "role only", "wildcard", "prefix", "false permission", "other permission", "invalid target", "literal me", "invalid business plus no grant", "duplicate query", "bad status", "server controls", "empty body", "null body", "array body", "scalar body", "wrong type", "missing fields", "null fields", "duplicate IDs", "numeric duplicate IDs", "null element", "numeric element", "bad reason", "missing key", "duplicate key", "case duplicate key", "missing version", "duplicate version", "case duplicate version", "preflight", "denied origin", "unsupported method", "unversioned"} {
			t.Run(op+"/"+mode, func(t *testing.T) {
				calls := []string{}
				target := uint64(23)
				version := uint64(9007199254740993)
				if mode == "max version" {
					version = 9223372036854775807
				}
				if mode == "self" {
					target = 17
				}
				user := &mocks.Service{MeFunc: func(context.Context, *emptypb.Empty, ...grpc.CallOption) (*userv1.UserData, error) {
					calls = append(calls, "Me")
					if mode == "Me failure" {
						return nil, status.Error(codes.Unauthenticated, "private-sentinel")
					}
					u := shared.MeUserData()
					u.Permissions = map[string]bool{assignmentPermission(op): true}
					u.MustChangePassword = mode == "restricted"
					switch mode {
					case "invalid Me":
						u.Id = 0
					case "role only", "invalid business plus no grant":
						u.Permissions = nil
					case "wildcard":
						u.Permissions = map[string]bool{"*": true}
					case "prefix":
						u.Permissions = map[string]bool{"iam.user.role": true}
					case "false permission":
						u.Permissions[assignmentPermission(op)] = false
					case "other permission":
						u.Permissions = map[string]bool{"iam.role.read": true}
					}
					return u, nil
				}}
				check := func(c context.Context) {
					assertAuthorizationMetadata(t, c)
					md, _ := metadata.FromOutgoingContext(c)
					if len(md.Get("version")) != 0 || len(md.Get("namespace")) != 0 || len(md.Get("x-service-token")) != 0 || md.Get("x-request-id")[0] != "assignment-http" {
						t.Error(md)
					}
					keys := md.Get("idempotency-key")
					if op == "list" {
						if len(keys) != 0 {
							t.Error(keys)
						}
					} else if !reflect.DeepEqual(keys, []string{"opaque, key"}) {
						t.Error(keys)
					}
				}
				sdk := &clientmocks.UserRoleManagementService{ListUserRolesFunc: func(c context.Context, r *userv1.ListUserRolesRequest) (*userv1.ListUserRolesResponse, error) {
					calls = append(calls, "ListUserRoles")
					check(c)
					if op != "list" || r.UserId != target || r.Query.Page != 1 || r.Query.Limit != 20 || r.Query.LastId != "opaque+/=%" || r.Query.Search.Keyword != "legal reviewer" || !reflect.DeepEqual(r.Filter.Statuses, []string{"ACTIVE", "REVOKED"}) {
						t.Error(r)
					}
					out := shared.UserRoleList(target)
					out.Version = version
					if mode == "empty list" {
						out.Items = nil
						out.Meta.Total = 1 << 32
					}
					if mode == "bad result" {
						out.Items = append(out.Items, nil)
					}
					return out, nil
				}}
				changed := mode != "no-op" && mode != "replay"
				changedIDs := []uint64{}
				if changed {
					changedIDs = []uint64{10}
				}
				inspectCommand := func(c context.Context, id uint64, ids []uint64, reason string, v *uint64) {
					check(c)
					if id != target || !reflect.DeepEqual(ids, []uint64{10, 11}) || reason != "Approved\r\nreason" || v == nil || *v != version {
						t.Error(id, ids, reason, v)
					}
				}
				sdk.GrantUserRolesFunc = func(c context.Context, r *userv1.GrantUserRolesRequest) (*userv1.GrantUserRolesResponse, error) {
					calls = append(calls, "GrantUserRoles")
					if op != "grant" {
						t.Error("wrong method")
					}
					inspectCommand(c, r.UserId, r.RoleIds, r.Reason, r.ExpectedUserVersion)
					if mode == "bad result" {
						return nil, nil
					}
					return &userv1.GrantUserRolesResponse{UserId: target, Version: version, Changed: changed, ChangedRoleIds: changedIDs}, nil
				}
				sdk.RevokeUserRolesFunc = func(c context.Context, r *userv1.RevokeUserRolesRequest) (*userv1.RevokeUserRolesResponse, error) {
					calls = append(calls, "RevokeUserRoles")
					if op != "revoke" {
						t.Error("wrong method")
					}
					inspectCommand(c, r.UserId, r.RoleIds, r.Reason, r.ExpectedUserVersion)
					if mode == "bad result" {
						return &userv1.RevokeUserRolesResponse{UserId: 99, Version: 1}, nil
					}
					return &userv1.RevokeUserRolesResponse{UserId: target, Version: version, Changed: changed, ChangedRoleIds: changedIDs}, nil
				}
				md := metadata.Pairs("version", "parent", "idempotency-key", "parent", "x-request-id", "assignment-http")
				parent := metadata.NewOutgoingContext(context.Background(), md)
				app := newAssignmentApp(t, user, sdk, func(c fiber.Ctx) error { c.SetContext(parent); return c.Next() })
				req := assignmentRequest(op)
				setBody := func(s string) { req.Body = io.NopCloser(strings.NewReader(s)); req.ContentLength = int64(len(s)) }
				want, n := 200, 2
				switch mode {
				case "missing bearer", "cookie only":
					req.Header.Del("Authorization")
					want, n = 401, 0
				case "Me failure", "invalid Me":
					want, n = 401, 1
				case "restricted", "role only", "wildcard", "prefix", "false permission", "other permission":
					want, n = 403, 1
				case "invalid business plus no grant":
					req.URL.Path = strings.Replace(req.URL.Path, "00023", "bad", 1)
					want, n = 403, 1
				case "invalid target", "literal me":
					v := "bad"
					if mode == "literal me" {
						v = "me"
					}
					req.URL.Path = strings.Replace(req.URL.Path, "00023", v, 1)
					want, n = 400, 1
				case "self":
					req.URL.Path = strings.Replace(req.URL.Path, "00023", "17", 1)
				case "max version":
					req.Header.Set("Version", "9223372036854775807")
				case "bad result":
					want = 500
				case "preflight":
					req.Header.Set("Origin", shared.BrowserOrigin)
					req.Header.Set("Access-Control-Request-Method", req.Method)
					req.Method = "OPTIONS"
					want, n = 204, 0
				case "denied origin":
					req.Header.Set("Origin", "https://evil.test")
					want, n = 403, 0
				case "unsupported method":
					req.Method = "PUT"
					// The existing broader users admin group rejects this non-admin actor.
					want, n = 403, 1
				case "unversioned":
					req.URL.Path = strings.TrimPrefix(req.URL.Path, "/api/v1")
					want, n = 404, 0
				}
				if op == "list" {
					switch mode {
					case "duplicate query":
						req.URL.RawQuery += "&page=1"
						want, n = 400, 1
					case "bad status":
						req.URL.RawQuery += "&statuses=DISABLED"
						want, n = 400, 1
					case "server controls":
						req.URL.RawQuery += "&sort=id"
						want, n = 400, 1
					}
					req.Header["Version"] = []string{"bad", "bad"}
					req.Header["Idempotency-Key"] = []string{"bad", "bad"}
				} else {
					switch mode {
					case "empty body":
						setBody("")
						want, n = 400, 1
					case "null body":
						setBody("null")
						want, n = 400, 1
					case "array body":
						setBody("[]")
						want, n = 400, 1
					case "scalar body":
						setBody(`"text"`)
						want, n = 400, 1
					case "wrong type":
						setBody(`{"role_ids":"10"}`)
						want, n = 400, 1
					case "missing fields":
						setBody("{}")
						want, n = 400, 1
					case "null fields":
						setBody(`{"role_ids":null,"reason":null}`)
						want, n = 400, 1
					case "duplicate IDs":
						setBody(`{"role_ids":["10","10"],"reason":"valid reason"}`)
						want, n = 400, 1
					case "numeric duplicate IDs":
						setBody(`{"role_ids":["10","00010"],"reason":"valid reason"}`)
						want, n = 400, 1
					case "null element":
						setBody(`{"role_ids":[null],"reason":"valid reason"}`)
						want, n = 400, 1
					case "numeric element":
						setBody(`{"role_ids":[10],"reason":"valid reason"}`)
						want, n = 400, 1
					case "bad reason":
						setBody(`{"role_ids":["10"],"reason":"\tinvalid"}`)
						want, n = 400, 1
					case "missing key":
						req.Header.Del("Idempotency-Key")
						want, n = 400, 1
					case "duplicate key":
						req.Header.Add("Idempotency-Key", "opaque, key")
						want, n = 400, 1
					case "case duplicate key":
						req.Header["iDeMpOtEnCy-KeY"] = []string{"opaque, key"}
						want, n = 400, 1
					case "missing version":
						req.Header.Del("Version")
						want, n = 400, 1
					case "duplicate version":
						req.Header.Add("Version", "9007199254740993")
						want, n = 400, 1
					case "case duplicate version":
						req.Header["vErSiOn"] = []string{"9007199254740993"}
						want, n = 400, 1
					}
				}
				res, e := app.Test(req)
				body := testutils.ResponseBody(t, res, e)
				if res.StatusCode != want || len(calls) != n || len(res.Cookies()) != 0 || mode != "unversioned" && res.Header.Get("Cache-Control") != "no-store" || strings.Contains(body, "browser-secret") || strings.Contains(body, "private-sentinel") {
					t.Fatal(want, res.StatusCode, body, calls)
				}
				if n == 2 {
					expected := map[string]string{"list": "ListUserRoles", "grant": "GrantUserRoles", "revoke": "RevokeUserRoles"}[op]
					if !reflect.DeepEqual(calls, []string{"Me", expected}) {
						t.Fatal(calls)
					}
				}
				if want == 200 {
					var env struct {
						Code, Message string
						Data          map[string]json.RawMessage
					}
					if json.Unmarshal([]byte(body), &env) != nil || env.Code != "SS" {
						t.Fatal(body)
					}
					count := 4
					message := "User roles granted successfully."
					if op == "list" {
						count = 10
						message = "User roles retrieved successfully."
					}
					if op == "revoke" {
						message = "User roles revoked successfully."
					}
					if len(env.Data) != count || env.Message != message {
						t.Fatal(body)
					}
					if op == "list" && mode != "empty list" {
						if string(env.Data["items"]) != "["+shared.UserRoleAssignmentJSON+"]" {
							t.Fatal(body)
						}
					}
					if op == "list" && mode == "empty list" && string(env.Data["items"]) != "[]" {
						t.Fatal(body)
					}
				}
				if md.Get("idempotency-key")[0] != "parent" || md.Get("x-request-id")[0] != "assignment-http" {
					t.Fatal("parent mutation")
				}
			})
		}
	}
}
func assignmentHTTPFailures(t *testing.T) {
	encoder := grpcerror.NewEncoder(append(append(mapping.UserCodeMappings(), mapping.RoleCodeMappings()...), mapping.ApplicationCodeMappings()...)...)
	cases := []struct {
		name string
		e    error
		code string
		http int
	}{{"version", encoder.Encode(usererrors.ErrVersionConflict, locale.Indonesian), "VC", 409}, {"idempotency", encoder.Encode(usererrors.ErrIdempotencyConflict, locale.Indonesian), "IC", 409}, {"busy", encoder.Encode(usererrors.ErrResourceBusy, locale.Indonesian), "RB", 409}, {"last admin", encoder.Encode(usererrors.ErrLastAdminRequired, locale.Indonesian), "LAR", 409}, {"401", common.ErrUnauthorized, "", 401}, {"403", status.Error(codes.PermissionDenied, "private-sentinel"), "", 403}, {"404", status.Error(codes.NotFound, "private-sentinel"), "", 404}, {"429", testutils.ServiceFieldError(t, 429), "EMAIL_USED", 429}, {"503", testutils.ServiceFieldError(t, 503), "EMAIL_USED", 503}, {"504", context.DeadlineExceeded, "", 504}, {"499", context.Canceled, "", 499}, {"unknown", status.Error(codes.Internal, "private-sentinel"), "", 500}}
	for _, e := range []error{usererrors.ErrGrantTargetInactive, usererrors.ErrSelfAdminRoleRevokeForbidden, roleerrors.ErrRoleDisabled, roleerrors.ErrRoleNotFound, apperrors.ErrApplicationInactive, apperrors.ErrApplicationAccessRequired} {
		encoded := encoder.Encode(e, locale.Indonesian)
		// Obtain the public code from the encoded contract rather than hardcoding another registry.
		st := status.Convert(encoded)
		httpStatus := 409
		if st.Code() == codes.PermissionDenied {
			httpStatus = 403
		}
		if st.Code() == codes.NotFound {
			httpStatus = 404
		}
		cases = append(cases, struct {
			name string
			e    error
			code string
			http int
		}{"eligibility/" + strconv.Itoa(len(cases)), encoded, "", httpStatus})
	}
	for _, op := range []string{"list", "grant", "revoke"} {
		for _, stage := range []string{"Me", "business"} {
			for _, tc := range cases {
				t.Run(op+"/"+stage+"/"+tc.name, func(t *testing.T) {
					calls := 0
					user := &mocks.Service{MeFunc: func(context.Context, *emptypb.Empty, ...grpc.CallOption) (*userv1.UserData, error) {
						calls++
						if stage == "Me" {
							return nil, tc.e
						}
						u := shared.MeUserData()
						u.Permissions = map[string]bool{assignmentPermission(op): true}
						return u, nil
					}}
					sdk := &clientmocks.UserRoleManagementService{ListUserRolesFunc: func(context.Context, *userv1.ListUserRolesRequest) (*userv1.ListUserRolesResponse, error) {
						calls++
						return nil, tc.e
					}, GrantUserRolesFunc: func(context.Context, *userv1.GrantUserRolesRequest) (*userv1.GrantUserRolesResponse, error) {
						calls++
						return nil, tc.e
					}, RevokeUserRolesFunc: func(context.Context, *userv1.RevokeUserRolesRequest) (*userv1.RevokeUserRolesResponse, error) {
						calls++
						return nil, tc.e
					}}
					app := newAssignmentApp(t, user, sdk)
					res, e := app.Test(assignmentRequest(op))
					body := testutils.ResponseBody(t, res, e)
					wantCalls := 2
					if stage == "Me" {
						wantCalls = 1
					}
					if res.StatusCode != tc.http || calls != wantCalls || len(res.Cookies()) != 0 || res.Header.Get("Cache-Control") != "no-store" || !strings.Contains(body, `"data":[]`) || strings.Contains(body, "private-sentinel") {
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
							t.Fatal("fallback", body)
						}
					}
				})
			}
		}
	}
}
func assignmentHTTPFlow(t *testing.T) {
	for _, op := range []string{"grant", "revoke"} {
		for _, mode := range []string{"saved replay", "revoked authority", "key conflict", "read only", "command only"} {
			t.Run(op+"/"+mode, func(t *testing.T) {
				calls := []string{}
				commands := 0
				user := &mocks.Service{MeFunc: func(context.Context, *emptypb.Empty, ...grpc.CallOption) (*userv1.UserData, error) {
					calls = append(calls, "Me")
					u := shared.MeUserData()
					u.Version = 1
					u.Permissions = map[string]bool{domain.UserRoleReadPermission: mode != "command only", assignmentPermission(op): mode != "read only"}
					return u, nil
				}}
				execute := func() (uint64, error) {
					commands++
					if commands == 2 {
						if mode == "revoked authority" {
							return 0, status.Error(codes.PermissionDenied, "private-sentinel")
						}
						if mode == "key conflict" {
							return 0, grpcerror.NewEncoder(mapping.UserCodeMappings()...).Encode(usererrors.ErrIdempotencyConflict, locale.Indonesian)
						}
					}
					return 1, nil
				}
				sdk := &clientmocks.UserRoleManagementService{ListUserRolesFunc: func(_ context.Context, r *userv1.ListUserRolesRequest) (*userv1.ListUserRolesResponse, error) {
					calls = append(calls, "ListUserRoles")
					return shared.UserRoleList(r.UserId), nil
				}, GrantUserRolesFunc: func(_ context.Context, r *userv1.GrantUserRolesRequest) (*userv1.GrantUserRolesResponse, error) {
					calls = append(calls, "GrantUserRoles")
					v, e := execute()
					return &userv1.GrantUserRolesResponse{UserId: r.UserId, Version: v}, e
				}, RevokeUserRolesFunc: func(_ context.Context, r *userv1.RevokeUserRolesRequest) (*userv1.RevokeUserRolesResponse, error) {
					calls = append(calls, "RevokeUserRoles")
					v, e := execute()
					return &userv1.RevokeUserRolesResponse{UserId: r.UserId, Version: v}, e
				}}
				app := newAssignmentApp(t, user, sdk)
				res, e := app.Test(assignmentRequest("list"))
				b := testutils.ResponseBody(t, res, e)
				want := 200
				if mode == "command only" {
					want = 403
				}
				if res.StatusCode != want {
					t.Fatal(b, calls)
				}
				for i := 0; i < 2; i++ {
					before := len(calls)
					res, e = app.Test(assignmentRequest(op))
					b = testutils.ResponseBody(t, res, e)
					want = 200
					if mode == "read only" {
						want = 403
					}
					if i == 1 && mode == "revoked authority" {
						want = 403
					}
					if i == 1 && mode == "key conflict" {
						want = 409
					}
					n := 2
					if mode == "read only" {
						n = 1
					}
					if res.StatusCode != want || len(calls) != before+n || calls[before] != "Me" || len(res.Cookies()) != 0 {
						t.Fatal(b, calls)
					}
					if want == 200 && !strings.Contains(b, `"version":"1","changed":false`) {
						t.Fatal("saved result", b)
					}
				}
			})
		}
	}
}
