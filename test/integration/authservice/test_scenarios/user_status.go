package test_scenarios

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/gofiber/fiber/v3"
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

func statusWireRequest() *http.Request {
	req := httptest.NewRequest("PATCH", "/api/v1/users/00023/status?user_id=99&namespace=forged", strings.NewReader(`{"status":"ACTIVE","reason":"  Approved\r\nreason  ","version":1,"user_id":99}`))
	req.Header.Set("Authorization", userBearer)
	req.Header.Set("Origin", shared.BrowserOrigin)
	req.Header.Set(locale.HTTPHeader, "id-ID")
	req.Header.Set("X-Service-Token", browserToken)
	req.Header.Set("Idempotency-Key", "opaque, key")
	req.Header.Set("Version", "9007199254740993")
	return req
}
func statusWireOutcomes(t *testing.T) {
	for _, mode := range []string{"success", "max version", "no growth replay", "wrong target", "invalid result", "restricted actor", "Me failure", "current authority denied", "method denied", "IAM denied", "not found", "409", "429", "503", "conflicting service token"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				cfg := secureConfig(t)
				want := 200
				switch mode {
				case "wrong target", "invalid result":
					want = 500
				case "restricted actor", "current authority denied", "method denied", "IAM denied":
					want = 403
				case "Me failure", "conflicting service token":
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
				h := newTransport(t, cfg, defaultServerTLS(t), func(c context.Context, r any, info *grpc.UnaryServerInfo, next grpc.UnaryHandler) (any, error) {
					if info.FullMethod == userv1.UserService_Me_FullMethodName {
						if mode == "Me failure" {
							return nil, status.Error(codes.Unauthenticated, "private-sentinel")
						}
						u := shared.MeUserData()
						u.Version = 1
						u.Permissions = map[string]bool{"iam.user.status.update": true}
						u.MustChangePassword = mode == "restricted actor"
						return u, nil
					}
					if info.FullMethod != userv1.UserService_ChangeUserStatus_FullMethodName {
						t.Error("unexpected method", info.FullMethod)
					}
					switch mode {
					case "current authority denied", "method denied", "IAM denied":
						return nil, status.Error(codes.PermissionDenied, "private-sentinel")
					case "not found":
						return nil, status.Error(codes.NotFound, "private-sentinel")
					case "409", "429", "503":
						return nil, testutils.ServiceFieldError(t, want)
					}
					result := &userv1.ChangeUserStatusResponse{UserId: 23, Status: "ACTIVE", Version: 9007199254740993, Changed: false}
					switch mode {
					case "max version":
						result.Version = 9223372036854775807
					case "no growth replay":
						result.Version = 1
					case "wrong target":
						result.UserId = 99
					case "invalid result":
						result.Status = "unknown"
					}
					if mode == "success" {
						return next(c, r)
					}
					return result, nil
				})
				defer h.close(t)
				if _, err := h.client.ChangeUserStatus(nil, &userv1.ChangeUserStatusRequest{}); err == nil {
					t.Fatal("nil context")
				}
				if _, err := h.client.ChangeUserStatus(context.Background(), nil); err == nil {
					t.Fatal("nil request")
				}
				before, _ := h.snapshot()
				if len(before) != 0 {
					t.Fatal("guard dispatch")
				}
				md := metadata.Pairs("x-request-id", "status-wire", "authorization", "Bearer parent", "version", "88", "idempotency-key", "inherited")
				if mode == "conflicting service token" {
					md.Set("x-service-token", "conflicting")
				}
				parent, cancel := context.WithTimeout(metadata.NewOutgoingContext(context.Background(), md), time.Second)
				defer cancel()
				app := shared.NewApp(t, h.client, func(c fiber.Ctx) error { c.SetContext(parent); return c.Next() })
				res, err := app.Test(statusWireRequest(), fiber.TestConfig{})
				body := testutils.ResponseBody(t, res, err)
				if res.StatusCode != want || res.Header.Get("Cache-Control") != "no-store" || len(res.Cookies()) != 0 {
					t.Fatal(res.StatusCode, want, body)
				}
				calls, dials := h.snapshot()
				n := 2
				if mode == "restricted actor" || mode == "Me failure" {
					n = 1
				}
				if mode == "conflicting service token" {
					n = 0
				}
				if len(calls) != n || n > 0 && dials != 1 {
					t.Fatal("calls/dials", len(calls), dials)
				}
				deadline, _ := parent.Deadline()
				for i, c := range calls {
					method := userv1.UserService_Me_FullMethodName
					if i == 1 {
						method = userv1.UserService_ChangeUserStatus_FullMethodName
					}
					if c.method != method || c.tls != tls.VersionTLS13 || c.deadline.IsZero() || c.deadline.After(deadline) {
						t.Fatal("method/TLS/deadline")
					}
					assertMetadata(t, c.metadata, "x-service-token", cfg.ServiceToken)
					assertMetadata(t, c.metadata, "authorization", userBearer)
					assertMetadata(t, c.metadata, "x-request-id", "status-wire")
					assertMetadata(t, c.metadata, locale.MetadataKey, "id-ID")
					if i == 1 {
						state := "ACTIVE"
						version := uint64(9007199254740993)
						if !proto.Equal(c.request.(*userv1.ChangeUserStatusRequest), &userv1.ChangeUserStatusRequest{UserId: 23, Status: &state, Reason: "Approved\r\nreason", ExpectedUserVersion: &version}) || len(c.metadata.Get("version")) != 0 {
							t.Fatal("payload/preconditions", c.request, c.metadata)
						}
						assertMetadata(t, c.metadata, "idempotency-key", "opaque, key")
					}
					if len(c.metadata.Get("namespace")) != 0 || len(c.metadata.Get("user_id")) != 0 || len(c.metadata.Get("cookie")) != 0 {
						t.Fatal("browser authority")
					}
				}
				if want == 200 {
					v := uint64(9007199254740993)
					if mode == "max version" {
						v = 9223372036854775807
					}
					if mode == "no growth replay" {
						v = 1
					}
					expected := fmt.Sprintf(`{"code":"SS","message":"User status updated successfully.","data":{"user_id":23,"status":"ACTIVE","version":"%d","changed":false}}`, v)
					if body != expected {
						t.Fatal(body)
					}
				} else if !strings.Contains(body, `"data":[]`) {
					t.Fatal(body)
				}
				if want == 409 || want == 429 || want == 503 {
					testutils.AssertServiceErrorResponse(t, res, body, want)
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
func statusWireContext(t *testing.T) {
	for _, mode := range []string{"client timeout", "parent deadline", "parent cancellation", "already canceled", "already expired"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				entered, finished := make(chan struct{}), make(chan struct{})
				cfg := secureConfig(t)
				h := newTransport(t, cfg, defaultServerTLS(t), func(c context.Context, r any, info *grpc.UnaryServerInfo, _ grpc.UnaryHandler) (any, error) {
					if info.FullMethod == userv1.UserService_Me_FullMethodName {
						u := shared.MeUserData()
						u.Permissions["iam.user.status.update"] = true
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
				res, err := app.Test(statusWireRequest(), fiber.TestConfig{})
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
				if len(calls) != n || n == 2 && (calls[0].method != userv1.UserService_Me_FullMethodName || calls[1].method != userv1.UserService_ChangeUserStatus_FullMethodName || dials != 1) {
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
