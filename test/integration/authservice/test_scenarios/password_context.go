package test_scenarios

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/gofiber/fiber/v3"
	authv1 "github.com/siti-nabila/api-contracts/pb/auth/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	sharedfixtures "github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func passwordWireContext(t *testing.T) {
	for _, mode := range []string{"client timeout", "parent deadline", "parent cancellation", "already canceled", "already expired"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				entered, finished := make(chan struct{}), make(chan struct{})
				cfg := secureConfig(t)
				h := newTransport(t, cfg, defaultServerTLS(t), func(ctx context.Context, _ any, info *grpc.UnaryServerInfo, _ grpc.UnaryHandler) (any, error) {
					if info.FullMethod == userv1.UserService_Me_FullMethodName {
						return sharedfixtures.MeUserData(), nil
					}
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
				app := sharedfixtures.NewApp(t, h.client, func(ctx fiber.Ctx) error { ctx.SetContext(parent); return ctx.Next() })
				request := httptest.NewRequest(http.MethodPost, "/api/v1/me/password", strings.NewReader(`{"current_password":"legacy-password","new_password":"new-password"}`))
				request.Header.Set(fiber.HeaderAuthorization, userBearer)
				request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
				start := time.Now()
				response, err := app.Test(request, fiber.TestConfig{})
				body := testutils.ResponseBody(t, response, err)
				if response.StatusCode != wantStatus || !strings.Contains(body, `"data":[]`) || strings.Contains(body, `"code":"SS"`) {
					t.Fatalf("context response status=%d want %d", response.StatusCode, wantStatus)
				}
				testutils.AssertRefreshCookie(t, response, false)
				wantCalls := 2
				if strings.HasPrefix(mode, "already ") {
					wantCalls = 0
				} else {
					<-finished
				}
				calls, dials := h.snapshot()
				if len(calls) != wantCalls {
					t.Fatalf("context RPCs=%d want %d", len(calls), wantCalls)
				}
				if wantCalls != 0 && (dials != 1 || calls[0].method != userv1.UserService_Me_FullMethodName || calls[1].method != authv1.AuthService_ChangePassword_FullMethodName) {
					t.Fatal("context failure changed RPC order or reused connection")
				}
				if mode == "client timeout" || mode == "parent deadline" {
					if time.Since(start) != wantDuration {
						t.Fatalf("elapsed=%v want %v", time.Since(start), wantDuration)
					}
				}
				if mode == "parent deadline" {
					deadline, _ := parent.Deadline()
					for _, call := range calls {
						if call.deadline.After(deadline) {
							t.Error("parent deadline extended")
						}
					}
				}
				assertNoSecrets(t, body, cfg.ServiceToken)
			})
		})
	}
}

func passwordWireConcurrent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const count = 8
		entered, release := make(chan struct{}, count), make(chan struct{})
		cfg := secureConfig(t)
		h := newTransport(t, cfg, defaultServerTLS(t), func(ctx context.Context, request any, info *grpc.UnaryServerInfo, _ grpc.UnaryHandler) (any, error) {
			md, _ := metadata.FromIncomingContext(ctx)
			id := md.Get("x-request-id")[0]
			index, err := strconv.Atoi(id)
			if err != nil {
				return nil, err
			}
			if info.FullMethod == userv1.UserService_Me_FullMethodName {
				user := sharedfixtures.MeUserData()
				user.Id = uint64(100 + index)
				user.MustChangePassword = index%2 != 0
				return user, nil
			}
			payload := request.(*authv1.ChangePasswordRequest)
			if payload.CurrentPassword != " old-"+id+" " || payload.NewPassword != "new-password-"+id {
				t.Error("concurrent credentials crossed requests")
			}
			entered <- struct{}{}
			<-release
			return &authv1.ChangePasswordResponse{Changed: index%2 == 0, LoginRequired: index%3 == 0}, nil
		})
		defer h.close(t)
		app := sharedfixtures.NewApp(t, h.client, func(ctx fiber.Ctx) error {
			ctx.SetContext(metadata.NewOutgoingContext(ctx.Context(), metadata.Pairs("x-request-id", ctx.Get("X-Request-ID"))))
			return ctx.Next()
		})
		type outcome struct {
			index    int
			response *http.Response
			err      error
		}
		results := make(chan outcome, count)
		for i := range count {
			go func() {
				id := strconv.Itoa(i)
				payload, _ := json.Marshal(map[string]string{"current_password": " old-" + id + " ", "new_password": "new-password-" + id})
				request := httptest.NewRequest(http.MethodPost, "/api/v1/me/password", strings.NewReader(string(payload)))
				request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
				request.Header.Set(fiber.HeaderAuthorization, "Bearer user-"+id)
				request.Header.Set("X-Request-ID", id)
				request.Header.Set(locale.HTTPHeader, []string{"en", "id"}[i%2])
				request.AddCookie(&http.Cookie{Name: "__Secure-refresh", Value: "private-refresh-" + id})
				response, err := app.Test(request, fiber.TestConfig{})
				results <- outcome{i, response, err}
			}()
		}
		for range count {
			<-entered
		}
		close(release)
		for range count {
			got := <-results
			body := testutils.ResponseBody(t, got.response, got.err)
			var envelope struct {
				Code string
				Data domain.ChangePasswordResponse
			}
			if err := json.Unmarshal([]byte(body), &envelope); err != nil {
				t.Fatal(err)
			}
			if got.response.StatusCode != 200 || envelope.Code != "SS" || envelope.Data != (domain.ChangePasswordResponse{Changed: got.index%2 == 0, LoginRequired: got.index%3 == 0}) || got.response.Header.Get(fiber.HeaderCacheControl) != "no-store" {
				t.Fatal("concurrent flags, result, or cache policy crossed requests")
			}
			testutils.AssertRefreshCookie(t, got.response, got.index%3 == 0)
			assertNoSecrets(t, body, cfg.ServiceToken)
			for _, secret := range []string{"old-", "new-password-", "private-refresh-"} {
				if strings.Contains(body, secret) {
					t.Error("concurrent credential leaked")
				}
			}
		}
		calls, dials := h.snapshot()
		if len(calls) != count*2 || dials != 1 {
			t.Fatalf("concurrent RPCs/dials=%d/%d", len(calls), dials)
		}
		methods := map[string][]string{}
		for _, call := range calls {
			id := call.metadata.Get("x-request-id")[0]
			index, _ := strconv.Atoi(id)
			methods[id] = append(methods[id], call.method)
			assertMetadata(t, call.metadata, "authorization", "Bearer user-"+id)
			assertMetadata(t, call.metadata, locale.MetadataKey, []string{"en", "id"}[index%2])
			assertMetadata(t, call.metadata, "x-service-token", cfg.ServiceToken)
			if call.tls != tls.VersionTLS13 || call.deadline.IsZero() {
				t.Error("concurrent TLS or deadline lost")
			}
			for _, key := range []string{"cookie", "refresh_token", "namespace", "user_id"} {
				if len(call.metadata.Get(key)) != 0 {
					t.Error("cookie or browser authority entered concurrent metadata")
				}
			}
		}
		if len(methods) != count {
			t.Fatal("request correlation lost")
		}
		for _, actual := range methods {
			if !reflect.DeepEqual(actual, []string{userv1.UserService_Me_FullMethodName, authv1.AuthService_ChangePassword_FullMethodName}) {
				t.Fatal("per-request RPC ordering differs")
			}
		}
	})
}
