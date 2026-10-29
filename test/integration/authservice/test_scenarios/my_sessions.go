package test_scenarios

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"github.com/gofiber/fiber/v3"
	sessionv1 "github.com/siti-nabila/api-contracts/pb/session/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	shared "github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

func mySessionsWireRequest(op string) *http.Request {
	method, path := "GET", "/api/v1/me/sessions?last_id=%252F%2B%20"
	if op == "one" {
		method, path = "DELETE", "/api/v1/me/sessions/"+shared.MySessionID
	}
	if op == "all" {
		method, path = "POST", "/api/v1/me/sessions/revoke-all"
	}
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	req := httptest.NewRequest(method, path+sep+"user_id=999&namespace=forged", strings.NewReader(`{"session_id":"forged","version":99,"user_id":999}`))
	req.Header.Set("Authorization", userBearer)
	req.Header.Set("Origin", shared.BrowserOrigin)
	req.Header.Set(locale.HTTPHeader, "id-ID")
	req.Header.Set("X-Service-Token", browserToken)
	req.Header.Set("Version", "99")
	req.Header.Set("Idempotency-Key", "forged")
	return req
}
func mySessionsWireMethod(op string) string {
	switch op {
	case "list":
		return sessionv1.SessionService_ListMySessions_FullMethodName
	case "one":
		return sessionv1.SessionService_RevokeMySession_FullMethodName
	default:
		return sessionv1.SessionService_RevokeAllMySessions_FullMethodName
	}
}
func mySessionsWireOutcomes(t *testing.T) {
	for _, op := range []string{"list", "one", "all"} {
		for _, mode := range []string{"false", "changed", "login", "both", "restricted", "me failure", "method denied", "iam namespace denied", "not found", "409", "429", "503"} {
			t.Run(op+"/"+mode, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					want := 200
					switch mode {
					case "restricted", "method denied", "iam namespace denied":
						want = 403
					case "me failure":
						want = 401
					case "not found":
						want = 404
					case "409":
						want = 409
					case "429":
						want = 429
					case "503":
						want = 503
					}
					cfg := secureConfig(t)
					h := newTransport(t, cfg, defaultServerTLS(t), func(ctx context.Context, r any, info *grpc.UnaryServerInfo, next grpc.UnaryHandler) (any, error) {
						if info.FullMethod == userv1.UserService_Me_FullMethodName {
							if mode == "me failure" {
								return nil, status.Error(codes.Unauthenticated, "private")
							}
							u := shared.MeUserData()
							u.MustChangePassword = mode == "restricted"
							return u, nil
						}
						switch mode {
						case "method denied", "iam namespace denied":
							return nil, status.Error(codes.PermissionDenied, "private")
						case "not found":
							return nil, status.Error(codes.NotFound, "private")
						case "409", "429", "503":
							return nil, testutils.ServiceFieldError(t, want)
						}
						switch op {
						case "list":
							return next(ctx, r)
						case "one":
							return &sessionv1.RevokeSessionResponse{Changed: mode == "changed" || mode == "both"}, nil
						default:
							return &sessionv1.RevokeAllMySessionsResponse{Changed: mode == "changed" || mode == "both", LoginRequired: mode == "login" || mode == "both"}, nil
						}
					})
					defer h.close(t)
					md := metadata.Pairs("x-request-id", "self-wire", "authorization", "Bearer parent")
					parent, stop := context.WithTimeout(metadata.NewOutgoingContext(context.Background(), md), time.Second)
					defer stop()
					app := shared.NewApp(t, h.client, func(c fiber.Ctx) error { c.SetContext(parent); return c.Next() })
					res, e := app.Test(mySessionsWireRequest(op), fiber.TestConfig{})
					body := testutils.ResponseBody(t, res, e)
					if res.StatusCode != want || res.Header.Get("Cache-Control") != "no-store" {
						t.Fatalf("status %d want %d %s", res.StatusCode, want, body)
					}
					calls, dials := h.snapshot()
					n := 2
					if mode == "restricted" || mode == "me failure" {
						n = 1
					}
					if len(calls) != n || dials != 1 {
						t.Fatal("calls/dials", len(calls), dials)
					}
					deadline, _ := parent.Deadline()
					for i, c := range calls {
						method := userv1.UserService_Me_FullMethodName
						if i == 1 {
							method = mySessionsWireMethod(op)
						}
						if c.method != method || c.tls != tls.VersionTLS13 || c.deadline.IsZero() || c.deadline.After(deadline) {
							t.Fatal("ordering TLS deadline")
						}
						assertMetadata(t, c.metadata, "x-service-token", cfg.ServiceToken)
						assertMetadata(t, c.metadata, "authorization", userBearer)
						assertMetadata(t, c.metadata, "x-request-id", "self-wire")
						assertMetadata(t, c.metadata, locale.MetadataKey, "id-ID")
						for _, key := range []string{"namespace", "user_id", "cookie", "version", "idempotency-key"} {
							if len(c.metadata.Get(key)) != 0 {
								t.Fatal("browser authority leaked")
							}
						}
					}
					if n == 2 {
						switch op {
						case "list":
							q := calls[1].request.(*sessionv1.ListMySessionsRequest).Query
							if q.Page != 1 || q.Limit != 20 || q.LastId != "%2F+ " || q.Fields != nil || q.Search != nil || len(q.Sort) != 2 || q.Sort[0].Field != "created_at" || !q.Sort[0].Desc || q.Sort[1].Field != "id" || !q.Sort[1].Desc {
								t.Fatal("query payload")
							}
						case "one":
							if calls[1].request.(*sessionv1.RevokeMySessionRequest).SessionId != shared.MySessionID {
								t.Fatal("target payload")
							}
						case "all":
							if !proto.Equal(calls[1].request.(*emptypb.Empty), &emptypb.Empty{}) {
								t.Fatal("all payload")
							}
						}
					}
					testutils.AssertRefreshCookie(t, res, want == 200 && op == "all" && (mode == "login" || mode == "both"))
					if want == 200 {
						var env struct{ Data map[string]json.RawMessage }
						if json.Unmarshal([]byte(body), &env) != nil {
							t.Fatal(body)
						}
						if op == "list" {
							if string(env.Data["total"]) != "2147483648" || len(env.Data) != 8 {
								t.Fatal(body)
							}
						} else {
							var changed bool
							if json.Unmarshal(env.Data["changed"], &changed) != nil || changed != (mode == "changed" || mode == "both") {
								t.Fatal(body)
							}
							if op == "one" && len(env.Data) != 1 {
								t.Fatal("invented current flag")
							}
							if op == "all" {
								var login bool
								if json.Unmarshal(env.Data["login_required"], &login) != nil || login != (mode == "login" || mode == "both") || len(env.Data) != 2 {
									t.Fatal(body)
								}
							}
						}
					}
					if mode == "409" || mode == "429" || mode == "503" {
						testutils.AssertServiceErrorResponse(t, res, body, want)
					}
					assertNoSecrets(t, body, cfg.ServiceToken)
					if strings.Contains(body, "private") {
						t.Fatal("private error leak")
					}
					original, _ := metadata.FromOutgoingContext(parent)
					if !reflect.DeepEqual(original, md) {
						t.Fatal("parent mutated")
					}
				})
			})
		}
	}
}
func mySessionsWireContext(t *testing.T) {
	for _, op := range []string{"list", "one", "all"} {
		for _, mode := range []string{"client timeout", "parent deadline", "parent cancellation", "already canceled", "already expired"} {
			t.Run(op+"/"+mode, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					entered, finished := make(chan struct{}), make(chan struct{})
					cfg := secureConfig(t)
					h := newTransport(t, cfg, defaultServerTLS(t), func(ctx context.Context, r any, info *grpc.UnaryServerInfo, next grpc.UnaryHandler) (any, error) {
						if info.FullMethod == userv1.UserService_Me_FullMethodName {
							return next(ctx, r)
						}
						close(entered)
						<-ctx.Done()
						close(finished)
						return nil, status.FromContextError(ctx.Err()).Err()
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
					res, e := app.Test(mySessionsWireRequest(op), fiber.TestConfig{})
					body := testutils.ResponseBody(t, res, e)
					if res.StatusCode != want || res.Header.Get("Cache-Control") != "no-store" || strings.Contains(body, `"code":"SS"`) {
						t.Fatalf("context status %d want %d", res.StatusCode, want)
					}
					testutils.AssertRefreshCookie(t, res, false)
					n := 2
					if strings.HasPrefix(mode, "already ") {
						n = 0
					} else {
						<-finished
					}
					calls, dials := h.snapshot()
					if len(calls) != n || (n == 2 && (calls[0].method != userv1.UserService_Me_FullMethodName || calls[1].method != mySessionsWireMethod(op) || dials != 1)) {
						t.Fatal("context call order/retry")
					}
					if (mode == "client timeout" || mode == "parent deadline") && time.Since(start) != duration {
						t.Fatal("deadline extended")
					}
					if mode == "parent deadline" {
						deadline, _ := parent.Deadline()
						if calls[1].deadline.After(deadline) {
							t.Fatal("earlier parent extended")
						}
					}
				})
			})
		}
	}
}
