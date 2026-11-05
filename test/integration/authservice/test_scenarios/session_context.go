package test_scenarios

import (
	"context"
	"crypto/tls"
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	authv1 "github.com/siti-nabila/api-contracts/pb/auth/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	authhandler "github.com/siti-nabila/rest-orc/internal/features/auth/handler"
	sharedfixtures "github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

func sessionWireContext(t *testing.T) {
	for _, op := range []string{"refresh", "bearer logout", "cookie logout"} {
		for _, mode := range []string{"client timeout", "parent deadline", "parent cancellation", "already canceled", "already expired"} {
			t.Run(op+"/"+mode, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					entered, finished := make(chan struct{}), make(chan struct{})
					cfg := secureConfig(t)
					h := newTransport(t, cfg, defaultServerTLS(t), func(ctx context.Context, _ any, _ *grpc.UnaryServerInfo, _ grpc.UnaryHandler) (any, error) {
						close(entered)
						<-ctx.Done()
						close(finished)
						return nil, status.FromContextError(ctx.Err()).Err()
					})
					defer h.close(t)
					parent, cancel := context.WithCancel(context.Background())
					defer cancel()
					wantStatus, wantDuration := 504, 5*time.Second
					switch mode {
					case "parent deadline":
						var stop context.CancelFunc
						parent, stop = context.WithTimeout(parent, time.Second)
						defer stop()
						wantDuration = time.Second
					case "parent cancellation":
						go func() { <-entered; cancel() }()
						wantStatus = 499
					case "already canceled":
						cancel()
						wantStatus = 499
					case "already expired":
						var stop context.CancelFunc
						parent, stop = context.WithDeadline(parent, time.Now().Add(-time.Second))
						defer stop()
					}
					app := sharedfixtures.NewApp(t, h.client, func(c fiber.Ctx) error { c.SetContext(parent); return c.Next() })
					csrf := wireBootstrap(t, app, "opaque")
					path, bearer, wantMethod := "logout", "", authv1.AuthService_Logout_FullMethodName
					if op == "refresh" {
						path = "refresh"
						wantMethod = authv1.AuthService_RefreshSession_FullMethodName
					} else if op == "bearer logout" {
						bearer = userBearer
					}
					start := time.Now()
					res, err := app.Test(wireSessionRequest(path, "opaque", csrf, bearer), fiber.TestConfig{})
					body := testutils.ResponseBody(t, res, err)
					if res.StatusCode != wantStatus || res.Header.Get("Cache-Control") != "no-store" || strings.Contains(body, `"code":"SS"`) {
						t.Fatalf("context status %d want %d", res.StatusCode, wantStatus)
					}
					testutils.AssertRefreshCookie(t, res, false)
					wantCalls := 1
					if strings.HasPrefix(mode, "already ") {
						wantCalls = 0
					} else {
						<-finished
					}
					calls, dials := h.snapshot()
					if len(calls) != wantCalls || wantCalls == 1 && (dials != 1 || calls[0].method != wantMethod) {
						t.Fatal("context failure dispatched Me/retry/wrong method")
					}
					if mode == "client timeout" || mode == "parent deadline" {
						if time.Since(start) != wantDuration {
							t.Fatalf("elapsed=%s want %s", time.Since(start), wantDuration)
						}
					}
					if mode == "parent deadline" {
						deadline, _ := parent.Deadline()
						if calls[0].deadline.After(deadline) {
							t.Fatal("parent deadline extended")
						}
					}
					assertNoSecrets(t, body, cfg.ServiceToken)
				})
			})
		}
	}
}

func sessionWireConcurrent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const count = 16
		entered, release := make(chan struct{}, count), make(chan struct{})
		cfg := secureConfig(t)
		h := newTransport(t, cfg, defaultServerTLS(t), func(ctx context.Context, r any, info *grpc.UnaryServerInfo, _ grpc.UnaryHandler) (any, error) {
			md, _ := metadata.FromIncomingContext(ctx)
			id := md.Get("x-request-id")[0]
			index, err := strconv.Atoi(id)
			if err != nil {
				return nil, err
			}
			switch index % 4 {
			case 0:
				if info.FullMethod != authv1.AuthService_RefreshSession_FullMethodName || r.(*authv1.RefreshSessionRequest).RefreshToken != "raw-"+id || len(md.Get("authorization")) != 0 {
					t.Error("concurrent refresh proof crossed requests")
				}
			case 1:
				if info.FullMethod != authv1.AuthService_Logout_FullMethodName || r.(*authv1.LogoutRequest).RefreshToken != nil {
					t.Error("concurrent bearer logout proof incorrect")
				}
			case 2:
				proof := r.(*authv1.LogoutRequest).RefreshToken
				if info.FullMethod != authv1.AuthService_Logout_FullMethodName || proof == nil || *proof != "raw-"+id || len(md.Get("authorization")) != 0 {
					t.Error("concurrent cookie logout proof crossed requests")
				}
			case 3:
				if info.FullMethod != userv1.UserService_Me_FullMethodName {
					t.Error("protected Me dispatched wrong RPC")
				}
			}
			entered <- struct{}{}
			<-release
			switch index % 4 {
			case 0:
				result := sharedfixtures.LoginTokenResponse()
				result.Token = "access-" + id
				result.RefreshToken = "rotated-" + id
				return result, nil
			case 1, 2:
				return &authv1.LogoutResponse{Changed: index%2 == 0}, nil
			default:
				user := sharedfixtures.MeUserData()
				user.Id = uint64(index + 100)
				return user, nil
			}
		})
		defer h.close(t)
		app := sharedfixtures.NewApp(t, h.client, func(c fiber.Ctx) error {
			c.SetContext(metadata.NewOutgoingContext(c.Context(), metadata.Pairs("authorization", "Bearer unselected-parent", "x-request-id", c.Get("X-Request-ID"))))
			return c.Next()
		})
		proofs := make([]string, count)
		for i := 0; i < count; i++ {
			proofs[i] = wireBootstrap(t, app, "raw-"+strconv.Itoa(i))
		}
		type outcome struct {
			index    int
			response *http.Response
			err      error
		}
		results := make(chan outcome, count)
		for i := 0; i < count; i++ {
			go func(i int) {
				id := strconv.Itoa(i)
				op, bearer := "logout", ""
				switch i % 4 {
				case 0:
					op = "refresh"
					bearer = "Bearer expired-" + id
				case 1:
					bearer = "Bearer user-" + id
				}
				req := wireSessionRequest(op, "raw-"+id, proofs[i], bearer)
				if i%4 == 3 {
					req = httptest.NewRequest("GET", "/api/v1/me", nil)
					req.Header.Set("Authorization", "Bearer user-"+id)
				}
				req.Header.Set("X-Request-ID", id)
				req.Header.Set(locale.HTTPHeader, []string{"en", "id"}[i%2])
				res, err := app.Test(req, fiber.TestConfig{})
				results <- outcome{i, res, err}
			}(i)
		}
		for i := 0; i < count; i++ {
			<-entered
		}
		close(release)
		csrfSeen := map[string]bool{}
		for i := 0; i < count; i++ {
			got := <-results
			body := testutils.ResponseBody(t, got.response, got.err)
			if got.response.StatusCode != 200 {
				t.Fatal("concurrent request failed")
			}
			id := strconv.Itoa(got.index)
			var envelope struct {
				Code string
				Data map[string]json.RawMessage
			}
			if json.Unmarshal([]byte(body), &envelope) != nil || envelope.Code != "SS" {
				t.Fatal("invalid concurrent envelope")
			}
			switch got.index % 4 {
			case 0:
				if string(envelope.Data["token"]) != `"access-`+id+`"` || len(got.response.Cookies()) != 2 {
					t.Fatal("concurrent session result crossed requests")
				}
				for _, c := range got.response.Cookies() {
					if c.Name == authhandler.RefreshCookie && c.Value != "rotated-"+id {
						t.Fatal("refresh cookie crossed requests")
					}
					if c.Name == authhandler.CSRFCookie {
						if csrfSeen[c.Value] {
							t.Fatal("CSRF binding shared between different refresh credentials")
						}
						csrfSeen[c.Value] = true
					}
				}
			case 1, 2:
				testutils.AssertRefreshCookie(t, got.response, true)
				if string(envelope.Data["changed"]) != strconv.FormatBool(got.index%2 == 0) {
					t.Fatal("logout bool crossed requests")
				}
			case 3:
				if string(envelope.Data["id"]) != strconv.Itoa(got.index+100) || len(got.response.Cookies()) != 0 {
					t.Fatal("Me result crossed requests")
				}
			}
			assertNoSecrets(t, body, cfg.ServiceToken)
			if strings.Contains(body, "raw-") || strings.Contains(body, "rotated-") {
				t.Fatal("raw cookie leaked in JSON")
			}
		}
		calls, dials := h.snapshot()
		if len(calls) != count || dials != 1 {
			t.Fatalf("mixed RPCs/dials=%d/%d", len(calls), dials)
		}
		for _, call := range calls {
			id := call.metadata.Get("x-request-id")[0]
			index, _ := strconv.Atoi(id)
			assertMetadata(t, call.metadata, "x-service-token", cfg.ServiceToken)
			assertMetadata(t, call.metadata, locale.MetadataKey, []string{"en", "id"}[index%2])
			if index%4 == 1 || index%4 == 3 {
				assertMetadata(t, call.metadata, "authorization", "Bearer user-"+id)
			} else if len(call.metadata.Get("authorization")) != 0 {
				t.Fatal("authorization leaked into cookie RPC")
			}
			if call.tls != tls.VersionTLS13 || call.deadline.IsZero() {
				t.Fatal("TLS/deadline lost")
			}
		}
		if len(csrfSeen) != count/4 {
			t.Fatal("missing independent CSRF results")
		}
	})
}
