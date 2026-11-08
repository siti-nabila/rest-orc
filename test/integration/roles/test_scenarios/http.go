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
	rolev1 "github.com/siti-nabila/api-contracts/pb/role/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	roleerrors "github.com/siti-nabila/api-contracts/pkg/dictionary/role"
	"github.com/siti-nabila/api-contracts/pkg/grpcerror"
	"github.com/siti-nabila/api-contracts/pkg/grpcerror/mapping"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	"github.com/siti-nabila/rest-orc/internal/client/authservice"
	"github.com/siti-nabila/rest-orc/internal/client/grpcclient"
	"github.com/siti-nabila/rest-orc/internal/features/roles/domain"
	clientmocks "github.com/siti-nabila/rest-orc/test/client/authservice/mocks"
	grpcfixtures "github.com/siti-nabila/rest-orc/test/client/grpcclient/fixtures"
	shared "github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

const bearer = "Bearer role-reader"

func newApp(t *testing.T, user authservice.Service, roles authservice.RoleCatalogService, before ...fiber.Handler) *fiber.App {
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
	client, err := authservice.NewWithRoleCatalogService(transport, user, &clientmocks.ProfileService{}, &clientmocks.PasswordService{}, &clientmocks.SessionService{}, &clientmocks.SelfSessionService{}, &clientmocks.AdminSessionService{}, &clientmocks.UserDetailService{}, &clientmocks.UserStatusService{}, &clientmocks.UserProfileService{}, &clientmocks.UserCreationService{}, roles)
	if err != nil {
		t.Fatal(err)
	}
	return shared.NewApp(t, client, before...)
}
func request(view string) *http.Request {
	path := "/api/v1/roles"
	if view == "detail" {
		path += "/00010"
	}
	if view == "options" {
		path += "/options"
	}
	path += "?page=0002&limit=020&keyword=legal%09reviewer&last_id=opaque%2B%2F%3D%25&application_code=clm&statuses=DISABLED,ACTIVE,ACTIVE&role_id=999&user_id=999&namespace=forged"
	r := httptest.NewRequest("GET", path, strings.NewReader(`{"role_id":999,"permissions":{"iam.role.read":true}}`))
	r.Header.Set("Authorization", bearer)
	r.Header.Set(locale.HTTPHeader, "id-ID")
	r.Header["Version"] = []string{"bad", "-1"}
	r.Header["Idempotency-Key"] = []string{"bad", "bad"}
	r.Header.Set("X-Service-Token", "browser-secret")
	r.AddCookie(&http.Cookie{Name: "__Secure-refresh", Value: "cookie-secret"})
	return r
}
func assertContext(t *testing.T, c context.Context, business bool) {
	t.Helper()
	md, _ := metadata.FromOutgoingContext(c)
	if !reflect.DeepEqual(md.Get("authorization"), []string{bearer}) || !reflect.DeepEqual(md.Get(locale.MetadataKey), []string{"id-ID"}) || md.Get("x-request-id")[0] != "role-http" || len(md.Get("x-service-token")) != 0 || len(md.Get("namespace")) != 0 {
		t.Fatal("metadata")
	}
	if business && (len(md.Get("version")) != 0 || len(md.Get("idempotency-key")) != 0) {
		t.Fatal("mutation metadata")
	}
	if _, ok := c.Deadline(); !ok {
		t.Fatal("missing deadline")
	}
}
func All() []testutils.Scenario {
	return []testutils.Scenario{{Name: "roles HTTP ordered reads query parity metadata and authority outcomes", Run: outcomes}, {Name: "roles HTTP encoded errors fields locale and retry guidance", Run: failures}, {Name: "role reads are fresh and list options share query and cursor", Run: fresh}}
}
func outcomes(t *testing.T) {
	for _, view := range []string{"list", "detail", "options"} {
		for _, mode := range []string{"success", "system role", "null metadata", "empty description", "whitespace description", "empty collections", "large total", "max version", "missing bearer", "cookie only", "Me failure", "invalid Me", "restricted", "role only", "wildcard", "prefix", "false permission", "user read", "role write", "invalid input", "duplicate query", "unsupported query", "nil response", "nil role", "wrong target", "malformed permission", "malformed meta", "preflight", "denied origin"} {
			t.Run(view+"/"+mode, func(t *testing.T) {
				calls := []string{}
				want := 200
				n := 2
				method := "ListRoles"
				if view == "detail" {
					method = "GetRole"
				}
				user := &mocks.Service{MeFunc: func(c context.Context, _ *emptypb.Empty, _ ...grpc.CallOption) (*userv1.UserData, error) {
					calls = append(calls, "Me")
					assertContext(t, c, false)
					if mode == "Me failure" {
						return nil, common.ErrUnauthorized
					}
					u := shared.MeUserData()
					u.Permissions = map[string]bool{domain.ReadPermission: true}
					u.RoleCodes = nil
					u.RoleNames = nil
					switch mode {
					case "invalid Me":
						u.Id = 0
					case "restricted":
						u.MustChangePassword = true
					case "role only":
						u.Permissions = nil
						u.RoleCodes = []int32{2}
						u.RoleNames = []string{"admin"}
					case "wildcard":
						u.Permissions = map[string]bool{"iam.role.*": true}
					case "prefix":
						u.Permissions = map[string]bool{"iam.role": true}
					case "false permission":
						u.Permissions[domain.ReadPermission] = false
					case "user read":
						u.Permissions = map[string]bool{"iam.user.read": true}
					case "role write":
						u.Permissions = map[string]bool{"iam.role.create": true, "iam.role.update": true}
					}
					return u, nil
				}}
				role := shared.RoleSummary()
				switch mode {
				case "system role":
					role.IsSystem = true
					role.Code = "iam.reader"
					role.ApplicationCode = "iam"
				case "null metadata":
					role.Description = nil
					role.CreatedAt = nil
					role.UpdatedAt = nil
				case "empty description":
					empty := ""
					role.Description = &empty
				case "whitespace description":
					space := " \r\n "
					role.Description = &space
				case "max version":
					role.Version = 9223372036854775807
				case "nil role":
					role = nil
				}
				sdk := &clientmocks.RoleCatalogService{List: func(c context.Context, r *rolev1.ListRolesRequest) (*rolev1.ListRolesResponse, error) {
					calls = append(calls, "ListRoles")
					if view == "detail" {
						t.Fatal("unexpected list")
					}
					assertContext(t, c, true)
					if r.Query.Page != 2 || r.Query.Limit != 20 || r.Query.LastId != "opaque+/=%" || r.Query.Search == nil || r.Query.Search.Keyword != "legal reviewer" || int32(r.Query.Search.Mode) != 5 || !reflect.DeepEqual(r.Query.Search.Fields, []string{"keyword"}) || r.Filter.ApplicationCode != "clm" || !reflect.DeepEqual(r.Filter.Statuses, []string{"ACTIVE", "DISABLED"}) || len(r.Query.Fields) != 0 || len(r.Query.Sort) != 3 {
						t.Error("query")
					}
					out := shared.RoleList()
					out.Items = []*rolev1.RoleSummary{role}
					if mode == "nil response" {
						return nil, nil
					}
					if mode == "empty collections" {
						out.Items = nil
						out.Meta.Total = 7
						out.Meta.TotalPages = 1
						out.Meta.Page = 2
						out.NextCursor = "next+/=%"
					}
					if mode == "large total" {
						out.Meta.Total = 9223372036854775807
					}
					if mode == "malformed meta" {
						out.Meta = nil
					}
					return out, nil
				}, Get: func(c context.Context, r *rolev1.GetRoleRequest) (*rolev1.GetRoleResponse, error) {
					calls = append(calls, "GetRole")
					assertContext(t, c, true)
					if view != "detail" || r.RoleId != 10 {
						t.Fatal("target source")
					}
					out := shared.RoleDetail()
					out.Role = role
					if mode == "system role" {
						out.Permissions[0].ApplicationCode = "iam"
					}
					if mode == "nil response" {
						return nil, nil
					}
					if mode == "empty collections" {
						out.Permissions = nil
					}
					if mode == "wrong target" {
						out.Role.Id = 11
					}
					if mode == "malformed permission" {
						out.Permissions[0] = nil
					}
					return out, nil
				}}
				md := metadata.Pairs("version", "old", "idempotency-key", "old", "x-request-id", "role-http")
				parent := metadata.NewOutgoingContext(context.Background(), md)
				app := newApp(t, user, sdk, func(c fiber.Ctx) error { c.SetContext(parent); return c.Next() })
				req := request(view)
				switch mode {
				case "missing bearer", "cookie only":
					req.Header.Del("Authorization")
					want = 401
					n = 0
				case "Me failure", "invalid Me":
					want = 401
					n = 1
				case "restricted", "role only", "wildcard", "prefix", "false permission", "user read", "role write":
					want = 403
					n = 1
				case "invalid input":
					if view == "detail" {
						req.URL.Path = "/api/v1/roles/2147483648"
					} else {
						req.URL.RawQuery = "page=0&limit=101&statuses=bad&application_code=Bad&keyword=%00secret"
					}
					want = 400
					n = 1
				case "duplicate query", "unsupported query":
					if view != "detail" {
						if mode == "duplicate query" {
							req.URL.RawQuery += "&page=2"
						} else {
							req.URL.RawQuery += "&created_from="
						}
						want = 400
						n = 1
					}
				case "nil response", "nil role":
					want = 500
				case "wrong target", "malformed permission":
					if view == "detail" {
						want = 500
					}
				case "malformed meta":
					if view != "detail" {
						want = 500
					}
				case "preflight":
					req.Method = "OPTIONS"
					req.Header.Set("Origin", shared.BrowserOrigin)
					req.Header.Set("Access-Control-Request-Method", "GET")
					want = 204
					n = 0
				case "denied origin":
					req.Header.Set("Origin", "https://evil.example.test")
					want = 403
					n = 0
				}
				res, err := app.Test(req)
				body := testutils.ResponseBody(t, res, err)
				expected := []string{"Me", method}[:n]
				if res.StatusCode != want || !reflect.DeepEqual(calls, expected) || res.Header.Get("Cache-Control") != "no-store" || len(res.Cookies()) != 0 {
					t.Fatal(view, mode, res.StatusCode, want, body, calls, expected)
				}
				if want == 200 {
					var env struct {
						Code, Message string
						Data          map[string]json.RawMessage
					}
					if json.Unmarshal([]byte(body), &env) != nil || env.Code != "SS" {
						t.Fatal(body)
					}
					if view == "detail" {
						if len(env.Data) != 2 || env.Message != "Role retrieved successfully." {
							t.Fatal(body)
						}
					} else {
						if len(env.Data) != 8 {
							t.Fatal(body)
						}
						if mode == "large total" && string(env.Data["total"]) != "9223372036854775807" {
							t.Fatal("narrow total")
						}
						if mode == "empty collections" && string(env.Data["items"]) != "[]" {
							t.Fatal(body)
						}
					}
					if mode == "success" {
						expected := ""
						switch view {
						case "list":
							expected = `{"code":"SS","message":"Roles retrieved successfully.","data":{"items":[` + shared.RoleJSON + `],"total":1,"page":1,"limit":20,"total_pages":1,"has_next":false,"has_prev":false,"next_cursor":""}}`
						case "detail":
							expected = `{"code":"SS","message":"Role retrieved successfully.","data":{"role":` + shared.RoleJSON + `,"permissions":` + shared.RolePermissionsJSON + `}}`
						case "options":
							expected = `{"code":"SS","message":"Role options retrieved successfully.","data":{"items":[{"value":"10","label":"clm.legal_reviewer - Reviewer legal kontrak","code":"clm.legal_reviewer","role_code":"3"}],"total":1,"page":1,"limit":20,"total_pages":1,"has_next":false,"has_prev":false,"next_cursor":""}}`
						}
						if body != expected {
							t.Fatal(body, expected)
						}
					}
				} else if want != 204 && !strings.Contains(body, `"data":[]`) {
					t.Fatal(body)
				}
				for _, secret := range []string{"browser-secret", "cookie-secret", "Bearer role-reader", "secret"} {
					if strings.Contains(body, secret) {
						t.Fatal("secret response")
					}
				}
				actual, _ := metadata.FromOutgoingContext(parent)
				if !reflect.DeepEqual(actual, md) {
					t.Fatal("parent mutation")
				}
			})
		}
	}
}
func failures(t *testing.T) {
	encoder := grpcerror.NewEncoder(mapping.RoleCodeMappings()...)
	cases := []struct {
		name string
		err  error
		http int
		code string
	}{
		{"role not found", encoder.Encode(roleerrors.ErrRoleNotFound, locale.Indonesian), 404, "RNF"},
		{"busy", encoder.Encode(roleerrors.ErrResourceBusy, locale.Indonesian), 409, "RB"},
		{"version", encoder.Encode(roleerrors.ErrVersionConflict, locale.Indonesian), 409, "VC"},
		{"unauthorized", common.ErrUnauthorized, 401, ""}, {"forbidden", status.Error(codes.PermissionDenied, "private-sentinel"), 403, ""}, {"cursor invalid", status.Error(codes.InvalidArgument, "private-sentinel"), 400, ""},
		{"fields conflict", testutils.ServiceFieldError(t, 409), 409, "EMAIL_USED"}, {"rate", testutils.ServiceFieldError(t, 429), 429, "EMAIL_USED"}, {"unavailable", testutils.ServiceFieldError(t, 503), 503, "EMAIL_USED"},
		{"timeout", context.DeadlineExceeded, 504, ""}, {"cancel", context.Canceled, 499, ""}, {"unknown", status.Error(codes.Internal, "private-sentinel"), 500, ""},
	}
	for _, view := range []string{"list", "detail", "options"} {
		for _, stage := range []string{"Me", "business"} {
			for _, tc := range cases {
				t.Run(view+"/"+stage+"/"+tc.name, func(t *testing.T) {
					calls := []string{}
					user := &mocks.Service{MeFunc: func(context.Context, *emptypb.Empty, ...grpc.CallOption) (*userv1.UserData, error) {
						calls = append(calls, "Me")
						if stage == "Me" {
							return nil, tc.err
						}
						u := shared.MeUserData()
						u.Permissions = map[string]bool{domain.ReadPermission: true}
						return u, nil
					}}
					sdk := &clientmocks.RoleCatalogService{List: func(context.Context, *rolev1.ListRolesRequest) (*rolev1.ListRolesResponse, error) {
						calls = append(calls, "ListRoles")
						return shared.RoleList(), tc.err
					}, Get: func(context.Context, *rolev1.GetRoleRequest) (*rolev1.GetRoleResponse, error) {
						calls = append(calls, "GetRole")
						return shared.RoleDetail(), tc.err
					}}
					app := newApp(t, user, sdk)
					res, err := app.Test(request(view))
					body := testutils.ResponseBody(t, res, err)
					n := 1
					if stage == "business" {
						n = 2
					}
					if res.StatusCode != tc.http || len(calls) != n || len(res.Cookies()) != 0 || res.Header.Get("Cache-Control") != "no-store" || !strings.Contains(body, `"data":[]`) || strings.Contains(body, "private-sentinel") {
						t.Fatal(body, calls)
					}
					if tc.code != "" && !strings.Contains(body, `"code":"`+tc.code+`"`) {
						t.Fatal(body)
					}
					if tc.code == "EMAIL_USED" {
						testutils.AssertServiceErrorResponse(t, res, body, tc.http)
					}
					if tc.code == "RNF" {
						if !strings.Contains(body, common.ErrNotFound.Message(locale.Indonesian)) {
							t.Fatal("not-found fallback changed", body)
						}
					}
					if tc.code == "RB" || tc.code == "VC" {
						if !strings.Contains(body, common.ErrInternalServerError.Message(locale.Indonesian)) {
							t.Fatal("registry fallback changed", body)
						}
					}
				})
			}
		}
	}
	for _, language := range []string{"en", "id", "zh"} {
		for _, view := range []string{"list", "options"} {
			t.Run("local fields/"+language+"/"+view, func(t *testing.T) {
				user := &mocks.Service{MeFunc: func(context.Context, *emptypb.Empty, ...grpc.CallOption) (*userv1.UserData, error) {
					u := shared.MeUserData()
					u.Permissions = map[string]bool{domain.ReadPermission: true}
					return u, nil
				}}
				app := newApp(t, user, &clientmocks.RoleCatalogService{})
				r := request(view)
				r.Header.Set(locale.HTTPHeader, language)
				r.URL.RawQuery = "page=bad-secret&limit=bad-secret&application_code=bad-secret&statuses=bad-secret&keyword=bad-secret%00"
				res, err := app.Test(r)
				body := testutils.ResponseBody(t, res, err)
				var env struct{ Errors map[string][]string }
				if res.StatusCode != 400 || json.Unmarshal([]byte(body), &env) != nil || len(env.Errors) != 5 || strings.Contains(body, "bad-secret") {
					t.Fatal(body)
				}
			})
		}
	}
}
func fresh(t *testing.T) {
	for _, denied := range []bool{false, true} {
		t.Run(map[bool]string{false: "fresh pages", true: "current denial"}[denied], func(t *testing.T) {
			calls := []string{}
			commands := 0
			var first *rolev1.ListRolesRequest
			user := &mocks.Service{MeFunc: func(context.Context, *emptypb.Empty, ...grpc.CallOption) (*userv1.UserData, error) {
				calls = append(calls, "Me")
				u := shared.MeUserData()
				u.Permissions = map[string]bool{domain.ReadPermission: true}
				return u, nil
			}}
			sdk := &clientmocks.RoleCatalogService{List: func(_ context.Context, r *rolev1.ListRolesRequest) (*rolev1.ListRolesResponse, error) {
				calls = append(calls, "ListRoles")
				commands++
				if first == nil {
					first = proto.Clone(r).(*rolev1.ListRolesRequest)
				} else if !proto.Equal(first, r) {
					t.Fatal("different list/options query")
				}
				if denied && commands == 2 {
					return nil, common.ErrForbidden
				}
				out := shared.RoleList()
				out.Items[0].Version = uint64(9007199254740993 + commands)
				return out, nil
			}}
			app := newApp(t, user, sdk)
			for i, view := range []string{"list", "options", "list"} {
				res, err := app.Test(request(view))
				body := testutils.ResponseBody(t, res, err)
				want := 200
				if denied && i == 1 {
					want = 403
				}
				if res.StatusCode != want || len(res.Cookies()) != 0 {
					t.Fatal(body)
				}
				if i == 2 && !strings.Contains(body, `"version":"9007199254740996"`) {
					t.Fatal("cached read", body)
				}
			}
			if commands != 3 || !reflect.DeepEqual(calls, []string{"Me", "ListRoles", "Me", "ListRoles", "Me", "ListRoles"}) {
				t.Fatal(calls)
			}
		})
	}
}
