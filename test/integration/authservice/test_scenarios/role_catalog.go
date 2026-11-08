package test_scenarios

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/gofiber/fiber/v3"
	paginatorv1 "github.com/siti-nabila/api-contracts/pb/paginator/v1"
	rolev1 "github.com/siti-nabila/api-contracts/pb/role/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	shared "github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func roleWireRequest(view string) *http.Request {
	path := "/api/v1/roles"
	if view == "options" {
		path += "/options"
	}
	if view == "detail" {
		path += "/00010"
	}
	path += "?page=001&limit=020&keyword=legal%09reviewer&application_code=clm&statuses=DISABLED,ACTIVE,ACTIVE&last_id=opaque%2B%2F%3D%25&namespace=forged&role_id=999"
	req := httptest.NewRequest("GET", path, nil)
	req.Header.Set("Authorization", userBearer)
	req.Header.Set("Origin", shared.BrowserOrigin)
	req.Header.Set(locale.HTTPHeader, "id-ID")
	req.Header.Set("X-Service-Token", browserToken)
	req.Header["Version"] = []string{"bad", "bad"}
	req.Header["Idempotency-Key"] = []string{"bad", "bad"}
	return req
}
func roleWireOutcomes(t *testing.T) {
	for _, view := range []string{"list", "options", "detail"} {
		for _, mode := range []string{"success", "current denied", "method denied", "IAM denied", "Me failure", "restricted", "missing grant", "invalid input", "nil nested result", "bad role", "max precision", "dependency unavailable", "not found", "409", "429", "503", "conflicting service token", "missing bearer"} {
			t.Run(view+"/"+mode, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					cfg := secureConfig(t)
					want, n := 200, 2
					switch mode {
					case "current denied", "method denied", "IAM denied":
						want = 403
					case "Me failure", "restricted", "missing grant", "invalid input":
						n = 1
						want = 403
						if mode == "Me failure" {
							want = 401
						}
						if mode == "invalid input" {
							want = 400
						}
					case "nil nested result", "bad role":
						want = 500
					case "dependency unavailable", "503":
						want = 503
					case "not found":
						want = 404
					case "409":
						want = 409
					case "429":
						want = 429
					case "conflicting service token", "missing bearer":
						want = 401
						n = 0
					}
					h := newTransport(t, cfg, defaultServerTLS(t), func(c context.Context, r any, info *grpc.UnaryServerInfo, next grpc.UnaryHandler) (any, error) {
						if info.FullMethod == userv1.UserService_Me_FullMethodName {
							if mode == "Me failure" {
								return nil, status.Error(codes.Unauthenticated, "private-sentinel")
							}
							u := shared.MeUserData()
							u.Permissions = map[string]bool{"iam.role.read": mode != "missing grant"}
							u.MustChangePassword = mode == "restricted"
							return u, nil
						}
						method := rolev1.RoleService_ListRoles_FullMethodName
						if view == "detail" {
							method = rolev1.RoleService_GetRole_FullMethodName
						}
						if info.FullMethod != method {
							t.Error("unexpected method")
						}
						switch mode {
						case "current denied", "method denied", "IAM denied":
							return nil, status.Error(codes.PermissionDenied, "private-sentinel")
						case "dependency unavailable":
							return nil, status.Error(codes.Unavailable, "private-sentinel")
						case "not found":
							return nil, status.Error(codes.NotFound, "private-sentinel")
						case "409", "429", "503":
							return nil, testutils.ServiceFieldError(t, want)
						}
						if mode == "success" {
							return next(c, r)
						}
						if view == "detail" {
							out := shared.RoleDetail()
							if mode == "nil nested result" {
								out.Role = nil
							}
							if mode == "bad role" {
								out.Role.Id = 99
							}
							if mode == "max precision" {
								out.Role.Version = 9223372036854775807
							}
							return out, nil
						}
						out := shared.RoleList()
						if mode == "nil nested result" {
							out.Meta = nil
						}
						if mode == "bad role" {
							out.Items[0].Version = 0
						}
						if mode == "max precision" {
							out.Meta.Total = 9223372036854775807
							out.Items[0].Version = 9223372036854775807
						}
						return out, nil
					})
					defer h.close(t)
					if _, err := h.client.ListRoles(nil, &rolev1.ListRolesRequest{}); err == nil {
						t.Fatal("nil context")
					}
					if _, err := h.client.GetRole(nil, &rolev1.GetRoleRequest{}); err == nil {
						t.Fatal("nil context")
					}
					if _, err := h.client.ListRoles(context.Background(), nil); err == nil {
						t.Fatal("nil request")
					}
					if _, err := h.client.GetRole(context.Background(), nil); err == nil {
						t.Fatal("nil request")
					}
					md := metadata.Pairs("x-request-id", "role-wire", "authorization", "Bearer parent", "version", "old", "idempotency-key", "old")
					if mode == "conflicting service token" {
						md.Set("x-service-token", "conflict")
					}
					parent, cancel := context.WithTimeout(metadata.NewOutgoingContext(context.Background(), md), time.Second)
					defer cancel()
					app := shared.NewApp(t, h.client, func(c fiber.Ctx) error { c.SetContext(parent); return c.Next() })
					req := roleWireRequest(view)
					if mode == "missing bearer" {
						req.Header.Del("Authorization")
					}
					if mode == "invalid input" {
						if view == "detail" {
							req.URL.Path = "/api/v1/roles/0"
						} else {
							req.URL.RawQuery = "page=0"
						}
					}
					res, err := app.Test(req, fiber.TestConfig{})
					body := testutils.ResponseBody(t, res, err)
					if res.StatusCode != want || res.Header.Get("Cache-Control") != "no-store" || len(res.Cookies()) != 0 {
						t.Fatal(view, mode, res.StatusCode, want, body)
					}
					calls, dials := h.snapshot()
					if len(calls) != n || n > 0 && dials != 1 {
						t.Fatal("calls/dials", len(calls), dials)
					}
					deadline, _ := parent.Deadline()
					for i, c := range calls {
						method := userv1.UserService_Me_FullMethodName
						if i == 1 {
							method = rolev1.RoleService_ListRoles_FullMethodName
							if view == "detail" {
								method = rolev1.RoleService_GetRole_FullMethodName
							}
						}
						if c.method != method || c.tls != tls.VersionTLS13 || c.deadline.IsZero() || c.deadline.After(deadline) {
							t.Fatal("method TLS deadline")
						}
						assertMetadata(t, c.metadata, "x-service-token", cfg.ServiceToken)
						assertMetadata(t, c.metadata, "authorization", userBearer)
						assertMetadata(t, c.metadata, "x-request-id", "role-wire")
						assertMetadata(t, c.metadata, locale.MetadataKey, "id-ID")
						if i == 1 {
							if len(c.metadata.Get("version")) != 0 || len(c.metadata.Get("idempotency-key")) != 0 {
								t.Fatal("mutation metadata")
							}
							if view == "detail" {
								if !proto.Equal(c.request.(*rolev1.GetRoleRequest), &rolev1.GetRoleRequest{RoleId: 10}) {
									t.Fatal("get payload")
								}
							} else {
								payload := &rolev1.ListRolesRequest{Query: &paginatorv1.PageQuery{Page: 1, Limit: 20, LastId: "opaque+/=%", Sort: []*paginatorv1.Sort{{Field: "created_at", Desc: true}, {Field: "updated_at", Desc: true}, {Field: "id", Desc: true}}, Search: &paginatorv1.Search{Fields: []string{"keyword"}, Keyword: "legal reviewer", Mode: paginatorv1.SearchMode_SEARCH_MODE_FULL_TEXT_TRIGRAM}}, Filter: &rolev1.RoleFilter{ApplicationCode: "clm", Statuses: []string{"ACTIVE", "DISABLED"}}}
								if !proto.Equal(c.request.(*rolev1.ListRolesRequest), payload) {
									t.Fatal("list payload")
								}
							}
						}
						for _, key := range []string{"namespace", "role_id", "user_id", "cookie"} {
							if len(c.metadata.Get(key)) != 0 {
								t.Fatal("browser authority")
							}
						}
					}
					if mode == "success" {
						expected := `{"code":"SS","message":"Roles retrieved successfully.","data":{"items":[` + shared.RoleJSON + `],"total":1,"page":1,"limit":20,"total_pages":1,"has_next":false,"has_prev":false,"next_cursor":""}}`
						if view == "detail" {
							expected = `{"code":"SS","message":"Role retrieved successfully.","data":{"role":` + shared.RoleJSON + `,"permissions":` + shared.RolePermissionsJSON + `}}`
						}
						if view == "options" {
							expected = `{"code":"SS","message":"Role options retrieved successfully.","data":{"items":[{"value":"10","label":"clm.legal_reviewer - Reviewer legal kontrak","code":"clm.legal_reviewer","role_code":"3"}],"total":1,"page":1,"limit":20,"total_pages":1,"has_next":false,"has_prev":false,"next_cursor":""}}`
						}
						if body != expected {
							t.Fatal(body, expected)
						}
					}
					if mode == "max precision" {
						if view != "detail" && !strings.Contains(body, `"total":9223372036854775807`) {
							t.Fatal("total narrowed")
						}
						if view != "options" && !strings.Contains(body, `"version":"9223372036854775807"`) {
							t.Fatal("version narrowed")
						}
					}
					if mode == "409" || mode == "429" || mode == "503" {
						testutils.AssertServiceErrorResponse(t, res, body, want)
					}
					if want != 200 && !strings.Contains(body, `"data":[]`) {
						t.Fatal(body)
					}
					assertNoSecrets(t, body, cfg.ServiceToken)
					if strings.Contains(body, "private-sentinel") {
						t.Fatal("raw internals")
					}
					after, _ := metadata.FromOutgoingContext(parent)
					if !reflect.DeepEqual(after, md) {
						t.Fatal("parent mutation")
					}
				})
			})
		}
	}
}

func roleWireContext(t *testing.T) {
	for _, view := range []string{"list", "options", "detail"} {
		for _, mode := range []string{"client timeout", "parent deadline", "parent cancellation", "already canceled", "already expired"} {
			t.Run(view+"/"+mode, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					entered, finished := make(chan struct{}), make(chan struct{})
					cfg := secureConfig(t)
					h := newTransport(t, cfg, defaultServerTLS(t), func(c context.Context, r any, info *grpc.UnaryServerInfo, _ grpc.UnaryHandler) (any, error) {
						if info.FullMethod == userv1.UserService_Me_FullMethodName {
							u := shared.MeUserData()
							u.Permissions["iam.role.read"] = true
							return u, nil
						}
						close(entered)
						<-c.Done()
						close(finished)
						return nil, status.FromContextError(c.Err()).Err()
					})
					defer h.close(t)
					parent, cancel := context.WithCancel(context.Background())
					defer cancel()
					want, duration := 504, 5*time.Second
					switch mode {
					case "parent deadline":
						var stop context.CancelFunc
						parent, stop = context.WithTimeout(parent, time.Second)
						defer stop()
						duration = time.Second
					case "parent cancellation":
						go func() { <-entered; cancel() }()
						want = 499
					case "already canceled":
						cancel()
						want = 499
					case "already expired":
						var stop context.CancelFunc
						parent, stop = context.WithDeadline(parent, time.Now().Add(-time.Second))
						defer stop()
					}
					app := shared.NewApp(t, h.client, func(c fiber.Ctx) error { c.SetContext(parent); return c.Next() })
					start := time.Now()
					res, err := app.Test(roleWireRequest(view), fiber.TestConfig{})
					body := testutils.ResponseBody(t, res, err)
					if res.StatusCode != want || res.Header.Get("Cache-Control") != "no-store" || len(res.Cookies()) != 0 || strings.Contains(body, `"code":"SS"`) {
						t.Fatal(res.StatusCode, body)
					}
					n := 2
					if strings.HasPrefix(mode, "already ") {
						n = 0
					} else {
						<-finished
					}
					calls, dials := h.snapshot()
					if len(calls) != n || n == 2 && (calls[0].method != userv1.UserService_Me_FullMethodName || calls[1].method != roleExpectedMethod(view) || dials != 1) {
						t.Fatal("ordering/retry")
					}
					if (mode == "parent deadline" || mode == "client timeout") && time.Since(start) != duration {
						t.Fatal("extended deadline")
					}
					if mode == "parent deadline" {
						deadline, _ := parent.Deadline()
						if calls[1].deadline.After(deadline) {
							t.Fatal("parent extended")
						}
					}
					if mode == "already canceled" && !errors.Is(parent.Err(), context.Canceled) {
						t.Fatal("parent state")
					}
				})
			})
		}
	}

}
func roleExpectedMethod(view string) string {
	if view == "detail" {
		return rolev1.RoleService_GetRole_FullMethodName
	}
	return rolev1.RoleService_ListRoles_FullMethodName
}
