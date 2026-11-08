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
	applicationv1 "github.com/siti-nabila/api-contracts/pb/application/v1"
	paginatorv1 "github.com/siti-nabila/api-contracts/pb/paginator/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	applicationerrors "github.com/siti-nabila/api-contracts/pkg/dictionary/application"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/api-contracts/pkg/grpcerror"
	"github.com/siti-nabila/api-contracts/pkg/grpcerror/mapping"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	"github.com/siti-nabila/rest-orc/internal/client/authservice"
	"github.com/siti-nabila/rest-orc/internal/client/grpcclient"
	"github.com/siti-nabila/rest-orc/internal/features/applications/domain"
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

const bearer = "Bearer application-reader"

func newApp(t *testing.T, user authservice.Service, sdk authservice.ApplicationCatalogService, before ...fiber.Handler) *fiber.App {
	t.Helper()
	tr, e := grpcclient.New(grpcfixtures.ValidConfig())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := tr.Close(); e != nil {
			t.Error(e)
		}
	})
	c, e := authservice.NewWithApplicationCatalogService(tr, user, &clientmocks.ProfileService{}, &clientmocks.PasswordService{}, &clientmocks.SessionService{}, &clientmocks.SelfSessionService{}, &clientmocks.AdminSessionService{}, &clientmocks.UserDetailService{}, &clientmocks.UserStatusService{}, &clientmocks.UserProfileService{}, &clientmocks.UserCreationService{}, &clientmocks.RoleCatalogService{}, &clientmocks.UserRoleManagementService{}, sdk)
	if e != nil {
		t.Fatal(e)
	}
	return shared.NewApp(t, c, before...)
}
func request(view string) *http.Request {
	path := "/api/v1/applications"
	if view == "options" {
		path += "/options"
	}
	path += "?page=001&limit=020&keyword=app%09name&statuses=DISABLED,ACTIVE,ACTIVE&last_id=opaque%2B%2F%3D%25&user_id=99&namespace=forged"
	req := httptest.NewRequest("GET", path, strings.NewReader("{"))
	req.Header.Set("Authorization", bearer)
	req.Header.Set(locale.HTTPHeader, "id-ID")
	req.Header.Set("X-Service-Token", "browser-secret")
	req.Header["Version"] = []string{"bad", "bad"}
	req.Header["Idempotency-Key"] = []string{"bad", "bad"}
	req.AddCookie(&http.Cookie{Name: "__Secure-refresh", Value: "browser-refresh"})
	return req
}
func All() []testutils.Scenario {
	return []testutils.Scenario{{Name: "application HTTP authority exact query shapes browser boundary and malformed successes", Run: outcomes}, {Name: "application HTTP safe encoded field retry fallback and context errors", Run: failures}, {Name: "application list options equivalent queries and cursor reuse", Run: equivalentPages}}
}
func outcomes(t *testing.T) {
	for _, view := range []string{"list", "options"} {
		for _, mode := range []string{"success", "empty", "max total", "system", "missing bearer", "cookie only", "Me failure", "invalid Me", "restricted", "role only", "wildcard", "prefix", "false grant", "user read", "role read", "application write", "no grant invalid query", "invalid query", "duplicate query", "unsupported query", "bad status", "nil result", "nil metadata", "bad item", "nil item", "bad page", "preflight", "denied origin", "unversioned", "detail alias", "mutation alias"} {
			t.Run(view+"/"+mode, func(t *testing.T) {
				calls := []string{}
				md := metadata.Pairs("authorization", "Bearer parent", "version", "old", "idempotency-key", "old", "x-request-id", "application-http")
				parent := metadata.NewOutgoingContext(context.Background(), md)
				user := &mocks.Service{MeFunc: func(context.Context, *emptypb.Empty, ...grpc.CallOption) (*userv1.UserData, error) {
					calls = append(calls, "Me")
					if mode == "Me failure" {
						return nil, status.Error(codes.Unauthenticated, "private-sentinel")
					}
					u := shared.MeUserData()
					u.RoleCodes = []int32{1}
					u.RoleNames = []string{"member"}
					u.Permissions = map[string]bool{domain.ReadPermission: true}
					u.MustChangePassword = mode == "restricted"
					switch mode {
					case "invalid Me":
						u.Id = 0
					case "role only", "no grant invalid query":
						u.Permissions = nil
					case "wildcard":
						u.Permissions = map[string]bool{"*": true}
					case "prefix":
						u.Permissions = map[string]bool{"iam.permission": true}
					case "false grant":
						u.Permissions[domain.ReadPermission] = false
					case "user read":
						u.Permissions = map[string]bool{"iam.user.read": true}
					case "role read":
						u.Permissions = map[string]bool{"iam.role.read": true}
					case "application write":
						u.Permissions = map[string]bool{"iam.user.application.grant": true, "iam.user.application.revoke": true}
					}
					return u, nil
				}}
				sdk := &clientmocks.ApplicationCatalogService{List: func(c context.Context, r *applicationv1.ListApplicationsRequest) (*applicationv1.ListApplicationsResponse, error) {
					calls = append(calls, "ListApplications")
					actual, _ := metadata.FromOutgoingContext(c)
					if !reflect.DeepEqual(actual.Get("authorization"), []string{bearer}) || actual.Get("x-request-id")[0] != "application-http" || len(actual.Get("version")) != 0 || len(actual.Get("idempotency-key")) != 0 || len(actual.Get("x-service-token")) != 0 || len(actual.Get("namespace")) != 0 {
						t.Error(actual)
					}
					expected := &applicationv1.ListApplicationsRequest{Query: &paginatorv1.PageQuery{Page: 1, Limit: 20, LastId: "opaque+/=%", Sort: []*paginatorv1.Sort{{Field: "id", Desc: true}}, Search: &paginatorv1.Search{Fields: []string{"keyword"}, Keyword: "app name", Mode: paginatorv1.SearchMode_SEARCH_MODE_FULL_TEXT_TRIGRAM}}, Filter: &applicationv1.ApplicationFilter{Statuses: []string{"ACTIVE", "DISABLED"}}}
					if !proto.Equal(r, expected) {
						t.Error(r, expected)
					}
					out := shared.ApplicationList()
					switch mode {
					case "nil result":
						return nil, nil
					case "nil metadata":
						out.Meta = nil
					case "bad item":
						out.Items[0].Id = 0
					case "nil item":
						out.Items = append(out.Items, nil)
					case "bad page":
						out.Meta.Total = -1
					case "empty":
						out.Items = nil
						out.Meta.Total = 42
						out.Meta.Page = 99
					case "max total":
						out.Meta.Total = 9223372036854775807
					case "system":
						out.Items[0].IsSystem = true
						out.Items[0].Code = "iam"
						out.Items[0].Status = "ACTIVE"
					}
					return out, nil
				}}
				app := newApp(t, user, sdk, func(c fiber.Ctx) error { c.SetContext(parent); return c.Next() })
				req := request(view)
				want, n := 200, 2
				switch mode {
				case "missing bearer", "cookie only":
					req.Header.Del("Authorization")
					want, n = 401, 0
				case "Me failure", "invalid Me":
					want, n = 401, 1
				case "restricted", "role only", "wildcard", "prefix", "false grant", "user read", "role read", "application write":
					want, n = 403, 1
				case "no grant invalid query":
					req.URL.RawQuery = "page=0"
					want, n = 403, 1
				case "invalid query":
					req.URL.RawQuery = "page=0&limit=101&statuses=REVOKED&keyword=%00private-sentinel"
					want, n = 400, 1
				case "duplicate query":
					req.URL.RawQuery += "&page=1"
					want, n = 400, 1
				case "unsupported query":
					req.URL.RawQuery += "&application_code="
					want, n = 400, 1
				case "bad status":
					req.URL.RawQuery = "statuses=REVOKED"
					want, n = 400, 1
				case "nil result", "nil metadata", "bad item", "nil item", "bad page":
					want = 500
				case "preflight":
					req.Method = "OPTIONS"
					req.Header.Set("Origin", shared.BrowserOrigin)
					req.Header.Set("Access-Control-Request-Method", "GET")
					want, n = 204, 0
				case "denied origin":
					req.Header.Set("Origin", "https://evil.test")
					want, n = 403, 0
				case "unversioned":
					req.URL.Path = strings.TrimPrefix(req.URL.Path, "/api/v1")
					want, n = 404, 0
				case "detail alias":
					req.URL.Path = "/api/v1/applications/5"
					want, n = 404, 0
				case "mutation alias":
					req.Method = "POST"
					// The existing shared mapper keeps unsupported HTTP methods on its safe fallback.
					want, n = 500, 0
				}
				res, e := app.Test(req)
				b := testutils.ResponseBody(t, res, e)
				noStore := mode != "unversioned" && mode != "detail alias"
				if res.StatusCode != want || len(calls) != n || len(res.Cookies()) != 0 || noStore && res.Header.Get("Cache-Control") != "no-store" || strings.Contains(b, "private-sentinel") || strings.Contains(b, "browser-secret") {
					t.Fatal(want, res.StatusCode, b, calls)
				}
				if n == 2 && !reflect.DeepEqual(calls, []string{"Me", "ListApplications"}) {
					t.Fatal(calls)
				}
				if want == 200 {
					var env struct {
						Code, Message string
						Data          map[string]json.RawMessage
					}
					if json.Unmarshal([]byte(b), &env) != nil || env.Code != "SS" || len(env.Data) != 8 {
						t.Fatal(b)
					}
					message := "Applications retrieved successfully."
					if view == "options" {
						message = "Application options retrieved successfully."
					}
					if env.Message != message {
						t.Fatal(b)
					}
					if mode == "success" {
						expected := "[" + shared.ApplicationJSON + "]"
						if view == "options" {
							expected = `[{"value":"clm","label":"clm -  Legacy NAME\té "}]`
						}
						if string(env.Data["items"]) != expected {
							t.Fatal(b)
						}
					}
					if mode == "empty" && (string(env.Data["items"]) != "[]" || string(env.Data["total"]) != "42" || string(env.Data["page"]) != "99") {
						t.Fatal(b)
					}
					if mode == "max total" && string(env.Data["total"]) != "9223372036854775807" {
						t.Fatal(b)
					}
				}
				if mode == "invalid query" {
					var env struct{ Errors map[string]json.RawMessage }
					_ = json.Unmarshal([]byte(b), &env)
					for _, key := range []string{"page", "limit", "statuses", "keyword"} {
						if len(env.Errors[key]) == 0 {
							t.Fatal("missing field", key, b)
						}
					}
				}
				after, _ := metadata.FromOutgoingContext(parent)
				if !reflect.DeepEqual(after, md) {
					t.Fatal("parent mutation")
				}
			})
		}
	}
}
func failures(t *testing.T) {
	encoder := grpcerror.NewEncoder(mapping.ApplicationCodeMappings()...)
	cases := []struct {
		name string
		e    error
		code string
		http int
	}{{"cursor", status.Error(codes.InvalidArgument, "private-sentinel"), "", 400}, {"401", common.ErrUnauthorized, "", 401}, {"403", status.Error(codes.PermissionDenied, "private-sentinel"), "", 403}, {"404", status.Error(codes.NotFound, "private-sentinel"), "", 404}, {"409", testutils.ServiceFieldError(t, 409), "EMAIL_USED", 409}, {"429", testutils.ServiceFieldError(t, 429), "EMAIL_USED", 429}, {"503", testutils.ServiceFieldError(t, 503), "EMAIL_USED", 503}, {"unregistered application", encoder.Encode(applicationerrors.ErrApplicationInactive, locale.Indonesian), "", 409}, {"deadline", context.DeadlineExceeded, "", 504}, {"cancel", context.Canceled, "", 499}, {"unknown", status.Error(codes.Internal, "private-sentinel"), "", 500}}
	for _, view := range []string{"list", "options"} {
		for _, stage := range []string{"Me", "ListApplications"} {
			for _, tc := range cases {
				t.Run(view+"/"+stage+"/"+tc.name, func(t *testing.T) {
					calls := []string{}
					user := &mocks.Service{MeFunc: func(context.Context, *emptypb.Empty, ...grpc.CallOption) (*userv1.UserData, error) {
						calls = append(calls, "Me")
						if stage == "Me" {
							return nil, tc.e
						}
						u := shared.MeUserData()
						u.Permissions = map[string]bool{domain.ReadPermission: true}
						return u, nil
					}}
					sdk := &clientmocks.ApplicationCatalogService{List: func(context.Context, *applicationv1.ListApplicationsRequest) (*applicationv1.ListApplicationsResponse, error) {
						calls = append(calls, "ListApplications")
						return shared.ApplicationList(), tc.e
					}}
					app := newApp(t, user, sdk)
					res, e := app.Test(request(view))
					b := testutils.ResponseBody(t, res, e)
					n := 2
					if stage == "Me" {
						n = 1
					}
					if res.StatusCode != tc.http || len(calls) != n || len(res.Cookies()) != 0 || res.Header.Get("Cache-Control") != "no-store" || !strings.Contains(b, `"data":[]`) || strings.Contains(b, "private-sentinel") {
						t.Fatal(res.StatusCode, b, calls)
					}
					if tc.code != "" {
						testutils.AssertServiceErrorResponse(t, res, b, tc.http)
					}
					if tc.name == "unregistered application" && !strings.Contains(b, common.ErrBadRequest.Message(locale.Indonesian)) {
						t.Fatal("localized fallback", b)
					}
				})
			}
		}
	}
}
func equivalentPages(t *testing.T) {
	calls := []string{}
	var first *applicationv1.ListApplicationsRequest
	user := &mocks.Service{MeFunc: func(context.Context, *emptypb.Empty, ...grpc.CallOption) (*userv1.UserData, error) {
		calls = append(calls, "Me")
		u := shared.MeUserData()
		u.Permissions = map[string]bool{domain.ReadPermission: true}
		return u, nil
	}}
	sdk := &clientmocks.ApplicationCatalogService{List: func(_ context.Context, r *applicationv1.ListApplicationsRequest) (*applicationv1.ListApplicationsResponse, error) {
		calls = append(calls, "ListApplications")
		if first == nil {
			first = proto.Clone(r).(*applicationv1.ListApplicationsRequest)
		} else {
			expected := proto.Clone(first).(*applicationv1.ListApplicationsRequest)
			if len(calls) > 4 {
				expected.Query.LastId = "next+/=%"
			}
			if !proto.Equal(expected, r) {
				t.Error("query differs", r, expected)
			}
		}
		out := shared.ApplicationList()
		out.NextCursor = "next+/=%"
		out.Meta.Total = 1 << 32
		return out, nil
	}}
	app := newApp(t, user, sdk)
	var next string
	for index, view := range []string{"list", "options", "options"} {
		req := request(view)
		if index == 2 {
			query := req.URL.Query()
			query.Set("last_id", next)
			req.URL.RawQuery = query.Encode()
		}
		res, e := app.Test(req)
		b := testutils.ResponseBody(t, res, e)
		var env struct {
			Data struct {
				NextCursor string `json:"next_cursor"`
			}
		}
		if json.Unmarshal([]byte(b), &env) != nil {
			t.Fatal(b)
		}
		next = env.Data.NextCursor
		if res.StatusCode != 200 || !strings.Contains(b, `"next_cursor":"next+/=%"`) {
			t.Fatal(b)
		}
	}
	if !reflect.DeepEqual(calls, []string{"Me", "ListApplications", "Me", "ListApplications", "Me", "ListApplications"}) {
		t.Fatal(calls)
	}
}
