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
	applicationv1 "github.com/siti-nabila/api-contracts/pb/application/v1"
	paginatorv1 "github.com/siti-nabila/api-contracts/pb/paginator/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	"github.com/siti-nabila/rest-orc/internal/client/authservice"
	clientmocks "github.com/siti-nabila/rest-orc/test/client/authservice/mocks"
	shared "github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func applicationWireRequest(view string) *http.Request {
	path := "/api/v1/applications"
	if view == "options" {
		path += "/options"
	}
	path += "?page=001&limit=020&keyword=app%09name&statuses=DISABLED,ACTIVE,ACTIVE&last_id=opaque%2B%2F%3D%25&namespace=forged"
	req := httptest.NewRequest("GET", path, strings.NewReader("{"))
	req.Header.Set("Authorization", userBearer)
	req.Header.Set("Origin", shared.BrowserOrigin)
	req.Header.Set(locale.HTTPHeader, "id-ID")
	req.Header.Set("X-Service-Token", browserToken)
	req.Header["Version"] = []string{"bad", "bad"}
	req.Header["Idempotency-Key"] = []string{"bad", "bad"}
	return req
}
func applicationWireOutcomes(t *testing.T) {
	for _, view := range []string{"list", "options"} {
		for _, mode := range []string{"success", "old constructor", "cursor rejected", "current denied", "method denied", "IAM denied", "Me failure", "restricted", "missing grant", "invalid input", "nil metadata", "bad item", "max total", "not found", "409", "429", "503", "conflicting service token", "missing bearer"} {
			t.Run(view+"/"+mode, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					cfg := secureConfig(t)
					want, n := 200, 2
					switch mode {
					case "current denied", "method denied", "IAM denied":
						want = 403
					case "Me failure":
						want, n = 401, 1
					case "restricted", "missing grant":
						want, n = 403, 1
					case "invalid input":
						want, n = 400, 1
					case "nil metadata", "bad item":
						want = 500
					case "cursor rejected":
						want = 400
					case "not found":
						want = 404
					case "409":
						want = 409
					case "429":
						want = 429
					case "503":
						want = 503
					case "conflicting service token", "missing bearer":
						want, n = 401, 0
					}
					h := newTransport(t, cfg, defaultServerTLS(t), func(c context.Context, r any, info *grpc.UnaryServerInfo, next grpc.UnaryHandler) (any, error) {
						if info.FullMethod == userv1.UserService_Me_FullMethodName {
							if mode == "Me failure" {
								return nil, status.Error(codes.Unauthenticated, "private-sentinel")
							}
							u := shared.MeUserData()
							u.Permissions = map[string]bool{"iam.permission.read": mode != "missing grant"}
							u.MustChangePassword = mode == "restricted"
							return u, nil
						}
						if info.FullMethod != applicationv1.ApplicationService_ListApplications_FullMethodName {
							t.Error("wrong method", info.FullMethod)
						}
						switch mode {
						case "current denied", "method denied", "IAM denied":
							return nil, status.Error(codes.PermissionDenied, "private-sentinel")
						case "cursor rejected":
							return nil, status.Error(codes.InvalidArgument, "private-sentinel")
						case "not found":
							return nil, status.Error(codes.NotFound, "private-sentinel")
						case "409", "429", "503":
							return nil, testutils.ServiceFieldError(t, want)
						case "success", "old constructor":
							return next(c, r)
						}
						out := shared.ApplicationList()
						if mode == "nil metadata" {
							out.Meta = nil
						}
						if mode == "bad item" {
							out.Items[0].Id = 0
						}
						if mode == "max total" {
							out.Meta.Total = 9223372036854775807
						}
						return out, nil
					})
					defer h.close(t)
					if mode == "old constructor" {
						var e error
						h.client, e = authservice.NewWithUserRoleManagementService(h.connection, userv1.NewUserServiceClient(h.connection.Connection()), &clientmocks.ProfileService{}, &clientmocks.PasswordService{}, &clientmocks.SessionService{}, &clientmocks.SelfSessionService{}, &clientmocks.AdminSessionService{}, &clientmocks.UserDetailService{}, &clientmocks.UserStatusService{}, &clientmocks.UserProfileService{}, &clientmocks.UserCreationService{}, &clientmocks.RoleCatalogService{}, &clientmocks.UserRoleManagementService{})
						if e != nil {
							t.Fatal(e)
						}
					}
					md := metadata.Pairs("x-request-id", "application-wire", "authorization", "Bearer parent", "version", "old", "idempotency-key", "old")
					if mode == "conflicting service token" {
						md.Set("x-service-token", "conflict")
					}
					parent, cancel := context.WithTimeout(metadata.NewOutgoingContext(context.Background(), md), time.Second)
					defer cancel()
					app := shared.NewApp(t, h.client, func(c fiber.Ctx) error { c.SetContext(parent); return c.Next() })
					req := applicationWireRequest(view)
					if mode == "missing bearer" {
						req.Header.Del("Authorization")
					}
					if mode == "invalid input" {
						req.URL.RawQuery = "page=0"
					}
					res, e := app.Test(req, fiber.TestConfig{})
					body := testutils.ResponseBody(t, res, e)
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
							method = applicationv1.ApplicationService_ListApplications_FullMethodName
						}
						if c.method != method || c.tls != tls.VersionTLS13 || c.deadline.IsZero() || c.deadline.After(deadline) {
							t.Fatal("method TLS deadline")
						}
						assertMetadata(t, c.metadata, "x-service-token", cfg.ServiceToken)
						assertMetadata(t, c.metadata, "authorization", userBearer)
						assertMetadata(t, c.metadata, "x-request-id", "application-wire")
						assertMetadata(t, c.metadata, locale.MetadataKey, "id-ID")
						if i == 1 {
							if len(c.metadata.Get("version")) != 0 || len(c.metadata.Get("idempotency-key")) != 0 {
								t.Fatal("read preconditions")
							}
							expected := &applicationv1.ListApplicationsRequest{Query: &paginatorv1.PageQuery{Page: 1, Limit: 20, LastId: "opaque+/=%", Sort: []*paginatorv1.Sort{{Field: "id", Desc: true}}, Search: &paginatorv1.Search{Fields: []string{"keyword"}, Keyword: "app name", Mode: paginatorv1.SearchMode_SEARCH_MODE_FULL_TEXT_TRIGRAM}}, Filter: &applicationv1.ApplicationFilter{Statuses: []string{"ACTIVE", "DISABLED"}}}
							if !proto.Equal(c.request.(*applicationv1.ListApplicationsRequest), expected) {
								t.Fatal(c.request)
							}
						}
						for _, key := range []string{"namespace", "application_code", "user_id", "cookie"} {
							if len(c.metadata.Get(key)) != 0 {
								t.Fatal("caller claims")
							}
						}
					}
					if mode == "success" || mode == "old constructor" {
						item := shared.ApplicationJSON
						message := "Applications retrieved successfully."
						if view == "options" {
							item = `{"value":"clm","label":"clm -  Legacy NAME\té "}`
							message = "Application options retrieved successfully."
						}
						expected := `{"code":"SS","message":"` + message + `","data":{"items":[` + item + `],"total":1,"page":1,"limit":20,"total_pages":1,"has_next":false,"has_prev":false,"next_cursor":""}}`
						if body != expected {
							t.Fatal(body, expected)
						}
					}
					if mode == "max total" && !strings.Contains(body, `"total":9223372036854775807`) {
						t.Fatal(body)
					}
					if mode == "409" || mode == "429" || mode == "503" {
						testutils.AssertServiceErrorResponse(t, res, body, want)
					}
					if want != 200 && !strings.Contains(body, `"data":[]`) {
						t.Fatal(body)
					}
					assertNoSecrets(t, body, cfg.ServiceToken)
					if strings.Contains(body, "private-sentinel") {
						t.Fatal(body)
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

func applicationWireContext(t *testing.T) {
	for _, view := range []string{"list", "options"} {
		for _, mode := range []string{"client timeout", "parent deadline", "parent cancellation", "already canceled", "already expired"} {
			t.Run(view+"/"+mode, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					entered, finished := make(chan struct{}), make(chan struct{})
					cfg := secureConfig(t)
					h := newTransport(t, cfg, defaultServerTLS(t), func(c context.Context, r any, info *grpc.UnaryServerInfo, _ grpc.UnaryHandler) (any, error) {
						if info.FullMethod == userv1.UserService_Me_FullMethodName {
							u := shared.MeUserData()
							u.Permissions = map[string]bool{"iam.permission.read": true}
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
					res, err := app.Test(applicationWireRequest(view), fiber.TestConfig{})
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
					if len(calls) != n || n == 2 && (calls[0].method != userv1.UserService_Me_FullMethodName || calls[1].method != applicationv1.ApplicationService_ListApplications_FullMethodName || dials != 1) {
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
