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
	shared "github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func accessWireRequest(op string) *http.Request {
	method, path, body := "GET", "/api/v1/users/00023/applications", "{"
	if op != "list" {
		method = "POST"
		path += "/clm/" + op
		body = `{"role_ids":["11","00010"],"reason":"  Approved\r\nreason  ","version":1,"user_id":99}`
	}
	if op == "list" {
		path += "?page=001&limit=020&keyword=legal%09reviewer&application_code=clm&statuses=REVOKED,ACTIVE,ACTIVE&last_id=opaque%2B%2F%3D%25&namespace=forged"
	}
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", userBearer)
	req.Header.Set("Origin", shared.BrowserOrigin)
	req.Header.Set(locale.HTTPHeader, "id-ID")
	req.Header.Set("X-Service-Token", browserToken)
	if op == "list" {
		req.Header["Version"] = []string{"bad", "bad"}
		req.Header["Idempotency-Key"] = []string{"bad", "bad"}
	} else {
		req.Header.Set("Version", "9007199254740993")
		req.Header.Set("Idempotency-Key", "opaque, key")
	}
	return req
}
func accessWirePermission(op string) string {
	if op == "grant" {
		return "iam.user.application.grant"
	}
	if op == "revoke" {
		return "iam.user.application.revoke"
	}
	return "iam.user.read"
}
func accessExpectedMethod(op string) string {
	switch op {
	case "grant":
		return applicationv1.ApplicationService_GrantApplicationAccess_FullMethodName
	case "revoke":
		return applicationv1.ApplicationService_RevokeApplicationAccess_FullMethodName
	}
	return userv1.UserService_ListUserApplicationAccess_FullMethodName
}
func accessWireOutcomes(t *testing.T) {
	for _, op := range []string{"list", "grant", "revoke"} {
		for _, mode := range []string{"success", "changed", "access only", "large revoke", "absent cleanup", "missing role grant", "current denied", "method denied", "IAM denied", "Me failure", "restricted", "missing grant", "invalid input", "bad result", "max precision", "not found", "409", "429", "503", "conflicting service token", "missing bearer"} {
			t.Run(op+"/"+mode, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					cfg := secureConfig(t)
					want, n := 200, 2
					switch mode {
					case "current denied", "method denied", "IAM denied":
						want = 403
					case "Me failure":
						want, n = 401, 1
					case "missing role grant":
						if op != "list" {
							want, n = 403, 1
						}
					case "restricted", "missing grant":
						want, n = 403, 1
					case "invalid input":
						want, n = 400, 1
					case "bad result":
						want = 500
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
							u.Permissions = map[string]bool{accessWirePermission(op): mode != "missing grant", "iam.user.role." + op: true}
							u.MustChangePassword = mode == "restricted"
							if mode == "missing role grant" {
								delete(u.Permissions, "iam.user.role."+op)
							}
							return u, nil
						}
						if info.FullMethod != accessExpectedMethod(op) {
							t.Error("wrong method")
						}
						switch mode {
						case "current denied", "method denied", "IAM denied":
							return nil, status.Error(codes.PermissionDenied, "private-sentinel")
						case "not found":
							return nil, status.Error(codes.NotFound, "private-sentinel")
						case "409", "429", "503":
							return nil, testutils.ServiceFieldError(t, want)
						}
						if mode == "success" {
							return next(c, r)
						}
						if op == "list" {
							out := shared.UserAccessHistory(23)
							if mode == "bad result" {
								out.Meta = nil
							}
							if mode == "max precision" {
								out.Version = 9223372036854775807
								out.Meta.Total = 9223372036854775807
							}
							return out, nil
						}
						v := uint64(9007199254740993)
						if mode == "max precision" {
							v = 9223372036854775807
						}
						if mode == "bad result" {
							v = 0
						}
						ids := []uint64{}
						if mode == "changed" {
							ids = []uint64{10}
						}
						if mode == "large revoke" && op == "revoke" {
							for k := 0; k < 25; k++ {
								ids = append(ids, uint64(k+1))
							}
						}
						if op == "grant" {
							return &applicationv1.GrantApplicationAccessResponse{UserId: 23, ApplicationId: 5, AccessStatus: accessWireStatus("ACTIVE"), Version: v, Changed: len(ids) > 0 || mode == "access only", ChangedRoleIds: ids}, nil
						}
						state := accessWireStatus("REVOKED")
						if mode == "absent cleanup" {
							state = nil
							ids = []uint64{10}
						}
						return &applicationv1.RevokeApplicationAccessResponse{UserId: 23, ApplicationId: 5, AccessStatus: state, Version: v, Changed: len(ids) > 0 || mode == "access only", ChangedRoleIds: ids}, nil
					})
					defer h.close(t)
					md := metadata.Pairs("x-request-id", "assignment-wire", "authorization", "Bearer parent", "version", "old", "idempotency-key", "old")
					if mode == "conflicting service token" {
						md.Set("x-service-token", "conflict")
					}
					parent, cancel := context.WithTimeout(metadata.NewOutgoingContext(context.Background(), md), time.Second)
					defer cancel()
					app := shared.NewApp(t, h.client, func(c fiber.Ctx) error { c.SetContext(parent); return c.Next() })
					req := accessWireRequest(op)
					if mode == "missing bearer" {
						req.Header.Del("Authorization")
					}
					if mode == "invalid input" {
						req.URL.Path = strings.Replace(req.URL.Path, "00023", "0", 1)
					}
					res, e := app.Test(req, fiber.TestConfig{})
					body := testutils.ResponseBody(t, res, e)
					if res.StatusCode != want || res.Header.Get("Cache-Control") != "no-store" || len(res.Cookies()) != 0 {
						t.Fatal(res.StatusCode, want, body)
					}
					calls, dials := h.snapshot()
					if len(calls) != n || n > 0 && dials != 1 {
						t.Fatal("calls/dials", len(calls), dials)
					}
					deadline, _ := parent.Deadline()
					for i, c := range calls {
						method := userv1.UserService_Me_FullMethodName
						if i == 1 {
							method = accessExpectedMethod(op)
						}
						if c.method != method || c.tls != tls.VersionTLS13 || c.deadline.IsZero() || c.deadline.After(deadline) {
							t.Fatal("method TLS deadline")
						}
						assertMetadata(t, c.metadata, "x-service-token", cfg.ServiceToken)
						assertMetadata(t, c.metadata, "authorization", userBearer)
						assertMetadata(t, c.metadata, "x-request-id", "assignment-wire")
						assertMetadata(t, c.metadata, locale.MetadataKey, "id-ID")
						if i == 1 {
							if len(c.metadata.Get("version")) != 0 {
								t.Fatal("version metadata")
							}
							v := uint64(9007199254740993)
							switch op {
							case "list":
								if len(c.metadata.Get("idempotency-key")) != 0 {
									t.Fatal("read key")
								}
								expected := &userv1.ListUserApplicationAccessRequest{UserId: 23, Query: &paginatorv1.PageQuery{Page: 1, Limit: 20, LastId: "opaque+/=%", Sort: []*paginatorv1.Sort{{Field: "application_id"}}, Search: &paginatorv1.Search{Fields: []string{"application_code", "name"}, Keyword: "legal reviewer", Mode: paginatorv1.SearchMode_SEARCH_MODE_CONTAINS}}, Filter: &userv1.UserApplicationAccessFilter{ApplicationCode: "clm", Statuses: []string{"ACTIVE", "REVOKED"}}}
								if !proto.Equal(c.request.(*userv1.ListUserApplicationAccessRequest), expected) {
									t.Fatal(c.request)
								}
							case "grant":
								assertMetadata(t, c.metadata, "idempotency-key", "opaque, key")
								expected := &applicationv1.GrantApplicationAccessRequest{UserId: 23, ApplicationCode: "clm", RoleIds: []uint64{10, 11}, Reason: "Approved\r\nreason", ExpectedUserVersion: &v}
								if !proto.Equal(c.request.(*applicationv1.GrantApplicationAccessRequest), expected) {
									t.Fatal(c.request)
								}
							case "revoke":
								assertMetadata(t, c.metadata, "idempotency-key", "opaque, key")
								expected := &applicationv1.RevokeApplicationAccessRequest{UserId: 23, ApplicationCode: "clm", Reason: "Approved\r\nreason", ExpectedUserVersion: &v}
								if !proto.Equal(c.request.(*applicationv1.RevokeApplicationAccessRequest), expected) {
									t.Fatal(c.request)
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
						if op == "list" {
							if !strings.Contains(body, `"items":[`+shared.UserAccessHistoryJSON+`]`) || !strings.Contains(body, `"user_id":23,"version":"9007199254740993"`) {
								t.Fatal(body)
							}
						} else if !strings.Contains(body, `"changed_role_ids":[],"version":"9007199254740993","changed":false`) {
							t.Fatal(body)
						}
					}
					if mode == "max precision" && !strings.Contains(body, `"version":"9223372036854775807"`) {
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
						t.Fatal("parent mutated")
					}
				})
			})
		}
	}
}

func accessWireContext(t *testing.T) {
	for _, view := range []string{"list", "grant", "revoke"} {
		for _, mode := range []string{"client timeout", "parent deadline", "parent cancellation", "already canceled", "already expired"} {
			t.Run(view+"/"+mode, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					entered, finished := make(chan struct{}), make(chan struct{})
					cfg := secureConfig(t)
					h := newTransport(t, cfg, defaultServerTLS(t), func(c context.Context, r any, info *grpc.UnaryServerInfo, _ grpc.UnaryHandler) (any, error) {
						if info.FullMethod == userv1.UserService_Me_FullMethodName {
							u := shared.MeUserData()
							u.Permissions = map[string]bool{accessWirePermission(view): true, "iam.user.role." + view: true}
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
					res, err := app.Test(accessWireRequest(view), fiber.TestConfig{})
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
					if len(calls) != n || n == 2 && (calls[0].method != userv1.UserService_Me_FullMethodName || calls[1].method != accessExpectedMethod(view) || dials != 1) {
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

func accessWireStatus(s string) *string { return &s }
