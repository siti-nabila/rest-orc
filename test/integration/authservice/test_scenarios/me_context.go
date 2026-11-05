package test_scenarios

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/gofiber/fiber/v3"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	commondictionary "github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	sharedfixtures "github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func meHTTPContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cfg := secureConfig(t)
		h := newTransport(t, cfg, defaultServerTLS(t), func(context.Context, any, *grpc.UnaryServerInfo, grpc.UnaryHandler) (any, error) {
			return sharedfixtures.MeUserData(), nil
		})
		defer h.close(t)
		parent, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		parent = metadata.NewOutgoingContext(parent, metadata.Pairs("x-request-id", "me-request"))
		app := sharedfixtures.NewApp(t, h.client, func(ctx fiber.Ctx) error { ctx.SetContext(parent); return ctx.Next() })
		response, body := secureHTTPRequest(t, app, http.MethodGet, "/api/v1/me", "", true)
		readMeData(t, response, body)
		calls, _ := h.snapshot()
		deadline, _ := parent.Deadline()
		if len(calls) != 1 || calls[0].deadline.IsZero() || calls[0].deadline.After(deadline) {
			t.Fatal("Me extended parent deadline")
		}
		assertMetadata(t, calls[0].metadata, "x-request-id", "me-request")
		assertHTTPCalls(t, calls, []string{userv1.UserService_Me_FullMethodName}, cfg.ServiceToken, true)
		md, _ := metadata.FromOutgoingContext(parent)
		if len(md) != 1 || len(md.Get("authorization")) != 0 || len(md.Get(locale.MetadataKey)) != 0 {
			t.Fatal("Me mutated parent metadata")
		}
	})
	for _, mode := range []string{"client timeout", "parent deadline", "parent cancellation", "already canceled", "already expired"} {
		t.Run(mode, func(t *testing.T) {
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
				wantStatus := http.StatusGatewayTimeout
				wantDuration := 5 * time.Second
				switch mode {
				case "parent deadline":
					var stop context.CancelFunc
					parent, stop = context.WithTimeout(parent, time.Second)
					defer stop()
					wantDuration = time.Second
				case "parent cancellation":
					go func() { <-entered; cancel() }()
					wantStatus = commondictionary.ErrRequestCanceled.HTTPStatus()
				case "already canceled":
					cancel()
					wantStatus = commondictionary.ErrRequestCanceled.HTTPStatus()
				case "already expired":
					var stop context.CancelFunc
					parent, stop = context.WithDeadline(parent, time.Now().Add(-time.Second))
					defer stop()
				}
				app := sharedfixtures.NewApp(t, h.client, func(ctx fiber.Ctx) error { ctx.SetContext(parent); return ctx.Next() })
				request := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
				request.Header.Set(fiber.HeaderAuthorization, userBearer)
				start := time.Now()
				response, body := meHTTPRequest(t, app, request)
				if response.StatusCode != wantStatus || !strings.Contains(body, `"data":[]`) || strings.Contains(body, `"code":"SS"`) {
					t.Fatalf("context response=%d/%s, want %d", response.StatusCode, body, wantStatus)
				}
				wantCalls := 1
				if strings.HasPrefix(mode, "already ") {
					wantCalls = 0
				} else {
					<-finished
				}
				calls, _ := h.snapshot()
				if len(calls) != wantCalls {
					t.Fatalf("context RPC count=%d, want %d", len(calls), wantCalls)
				}
				if mode == "client timeout" || mode == "parent deadline" {
					if time.Since(start) != wantDuration {
						t.Errorf("elapsed=%s, want %s", time.Since(start), wantDuration)
					}
				}
				assertNoSecrets(t, body, cfg.ServiceToken)
			})
		})
	}
}

func meHTTPConcurrent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const count = 8
		entered, release := make(chan struct{}, count), make(chan struct{})
		cfg := secureConfig(t)
		h := newTransport(t, cfg, defaultServerTLS(t), func(ctx context.Context, _ any, _ *grpc.UnaryServerInfo, _ grpc.UnaryHandler) (any, error) {
			md, _ := metadata.FromIncomingContext(ctx)
			id := md.Get("x-request-id")[0]
			index, err := strconv.Atoi(id)
			if err != nil {
				return nil, err
			}
			entered <- struct{}{}
			<-release
			user := sharedfixtures.MeUserData()
			user.Id = uint64(100 + index)
			user.Email = id + "@example.com"
			user.Roles[0].Code = "role-" + id
			user.ApplicationAccess[0].Code = "app-" + id
			user.Permissions = map[string]bool{"clm.user-" + id: true}
			return user, nil
		})
		defer h.close(t)
		app := sharedfixtures.NewApp(t, h.client, func(ctx fiber.Ctx) error {
			ctx.SetContext(metadata.NewOutgoingContext(ctx.Context(), metadata.Pairs("x-request-id", ctx.Get("X-Request-ID"))))
			return ctx.Next()
		})
		type result struct {
			index    int
			response *http.Response
			err      error
		}
		results := make(chan result, count)
		for i := range count {
			go func() {
				id := strconv.Itoa(i)
				request := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
				request.Header.Set("X-Request-ID", id)
				request.Header.Set(fiber.HeaderAuthorization, "Bearer user-"+id)
				request.Header.Set(locale.HTTPHeader, []string{"en", "id"}[i%2])
				response, err := app.Test(request, fiber.TestConfig{})
				results <- result{i, response, err}
			}()
		}
		for range count {
			<-entered
		}
		close(release)
		for range count {
			got := <-results
			body := testutils.ResponseBody(t, got.response, got.err)
			data := readMeData(t, got.response, body)
			id := strconv.Itoa(got.index)
			assertMeField(t, data, "id", strconv.Itoa(100+got.index))
			assertMeField(t, data, "email", fmt.Sprintf("%q", id+"@example.com"))
			assertMeField(t, data, "permissions", `{"clm.user-`+id+`":true}`)
			if !strings.Contains(string(data["roles"]), `"code":"role-`+id+`"`) || !strings.Contains(string(data["application_access"]), `"code":"app-`+id+`"`) {
				t.Fatal("concurrent access data leaked")
			}
		}
		calls, dials := h.snapshot()
		if len(calls) != count || dials != 1 {
			t.Fatalf("concurrent RPCs/dials=%d/%d, want %d/1", len(calls), dials, count)
		}
		seen := map[string]bool{}
		for _, call := range calls {
			if call.method != userv1.UserService_Me_FullMethodName {
				t.Error("concurrent request aggregated another RPC")
			}
			id := call.metadata.Get("x-request-id")[0]
			if seen[id] {
				t.Fatal("duplicate request identity")
			}
			seen[id] = true
			i, _ := strconv.Atoi(id)
			assertMetadata(t, call.metadata, "authorization", "Bearer user-"+id)
			assertMetadata(t, call.metadata, locale.MetadataKey, []string{"en", "id"}[i%2])
			assertMetadata(t, call.metadata, "x-service-token", cfg.ServiceToken)
		}
	})
}
