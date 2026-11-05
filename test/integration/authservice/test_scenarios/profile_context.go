package test_scenarios

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"github.com/gofiber/fiber/v3"
	profilev1 "github.com/siti-nabila/api-contracts/pb/profile/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	usersfixtures "github.com/siti-nabila/rest-orc/test/integration/users/fixtures"
	sharedfixtures "github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

func profileWireContext(t *testing.T) {
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
				request := httptest.NewRequest(http.MethodPatch, "/api/v1/me/profile", strings.NewReader(`{"name":"NASA"}`))
				request.Header.Set(fiber.HeaderAuthorization, userBearer)
				request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
				start := time.Now()
				response, err := app.Test(request, fiber.TestConfig{})
				body := testutils.ResponseBody(t, response, err)
				if response.StatusCode != wantStatus || !strings.Contains(body, `"data":[]`) || strings.Contains(body, `"code":"SS"`) {
					t.Fatalf("context response=%d %s want %d", response.StatusCode, body, wantStatus)
				}
				wantCalls := 2
				if strings.HasPrefix(mode, "already ") {
					wantCalls = 0
				} else {
					<-finished
				}
				calls, dials := h.snapshot()
				if len(calls) != wantCalls {
					t.Fatalf("calls=%d want %d", len(calls), wantCalls)
				}
				if wantCalls != 0 {
					if dials != 1 || calls[0].method != userv1.UserService_Me_FullMethodName || calls[1].method != profilev1.ProfileService_UpdateProfile_FullMethodName {
						t.Fatalf("unexpected RPC order/dials: %v %d", calls, dials)
					}
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
							t.Fatal("extended parent deadline")
						}
					}
				}
				assertNoSecrets(t, body, cfg.ServiceToken)
			})
		})
	}
}
func profileWireConcurrent(t *testing.T) {
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
				return user, nil
			}
			payload := request.(*profilev1.ProfileRequest)
			entered <- struct{}{}
			<-release
			result := usersfixtures.ProfileResponse()
			result.Profile.UserId = uint64(100 + index)
			result.Profile.Name = payload.Name
			result.Profile.Phone = payload.Phone
			result.Version += uint64(index)
			return result, nil
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
				name := "member " + string(rune('a'+i))
				phone := "081234567890" + id
				payload, _ := json.Marshal(map[string]string{"name": name, "phone": phone, "country_code": "62"})
				request := httptest.NewRequest(http.MethodPatch, "/api/v1/me/profile", strings.NewReader(string(payload)))
				request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
				request.Header.Set(fiber.HeaderAuthorization, "Bearer user-"+id)
				request.Header.Set("X-Request-ID", id)
				request.Header.Set(locale.HTTPHeader, []string{"en", "id"}[i%2])
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
			var result struct{ Data domain.UpdateProfileResponse }
			if err := json.Unmarshal([]byte(body), &result); err != nil {
				t.Fatal(err)
			}
			id := strconv.Itoa(got.index)
			if got.response.StatusCode != 200 || result.Data.Profile == nil || result.Data.Version == nil || result.Data.Profile.UserID != uint64(100+got.index) || result.Data.Profile.Name != "Member "+string(rune('A'+got.index)) || result.Data.Profile.Phone != "81234567890"+id || *result.Data.Version != strconv.FormatUint(9007199254740993+uint64(got.index), 10) {
				t.Fatalf("concurrent response=%d %s", got.response.StatusCode, body)
			}
			assertNoSecrets(t, body, cfg.ServiceToken)
		}
		calls, dials := h.snapshot()
		if len(calls) != count*2 || dials != 1 {
			t.Fatalf("RPCs/dials=%d/%d", len(calls), dials)
		}
		methods := map[string][]string{}
		for _, call := range calls {
			id := call.metadata.Get("x-request-id")[0]
			index, _ := strconv.Atoi(id)
			methods[id] = append(methods[id], call.method)
			assertMetadata(t, call.metadata, "authorization", "Bearer user-"+id)
			assertMetadata(t, call.metadata, locale.MetadataKey, []string{"en", "id"}[index%2])
			assertMetadata(t, call.metadata, "x-service-token", cfg.ServiceToken)
			if call.tls != tls.VersionTLS13 {
				t.Fatal("TLS downgraded")
			}
			if call.method == profilev1.ProfileService_UpdateProfile_FullMethodName {
				payload := call.request.(*profilev1.ProfileRequest)
				if payload.Name != "Member "+string(rune('A'+index)) || payload.Phone != "81234567890"+id || payload.GetCountryCode() != "+62" || !reflect.DeepEqual(payload.FieldMask.Paths, []string{"name", "phone", "country_code"}) {
					t.Fatalf("cross-request payload=%v", payload)
				}
			}
		}
		if len(methods) != count {
			t.Fatal("lost request identity")
		}
		for _, actual := range methods {
			if !reflect.DeepEqual(actual, []string{userv1.UserService_Me_FullMethodName, profilev1.ProfileService_UpdateProfile_FullMethodName}) {
				t.Fatalf("per-request order=%v", actual)
			}
		}
	})
}
