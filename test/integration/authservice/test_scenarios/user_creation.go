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

func creationWireRequest() *http.Request {
	req := httptest.NewRequest("POST", "/api/v1/users?user_id=99&version=bad&namespace=forged", strings.NewReader(`{"email":" User.Name@Example.COM ","password":"simple","name":"  NASA  o'neil-jean  ","address":" X\r\nY ","country_code":"62","phone":"00628123456789","user_id":99,"version":1,"field_mask":["password"],"permissions":{"iam.user.create":true}}`))
	req.Header.Set("Authorization", userBearer)
	req.Header.Set("Origin", shared.BrowserOrigin)
	req.Header.Set(locale.HTTPHeader, "id-ID")
	req.Header.Set("X-Service-Token", browserToken)
	req.Header.Set("Idempotency-Key", "opaque, key")
	req.Header["Version"] = []string{"bad", "also bad"}
	return req
}
func creationWireOutcomes(t *testing.T) {
	for _, mode := range []string{"success", "old constructor default", "missing key", "empty object", "null", "max version", "wrong email", "wrong status", "invalid result", "restricted actor", "Me failure", "current authority denied", "method denied", "IAM denied", "default member unavailable", "not found", "409", "429", "503", "conflicting service token"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				cfg := secureConfig(t)
				want := 200
				n := 2
				switch mode {
				case "missing key", "empty object", "null":
					want = 400
					n = 1
				case "wrong email", "wrong status", "invalid result":
					want = 500
				case "restricted actor":
					want = 403
					n = 1
				case "Me failure":
					want = 401
					n = 1
				case "conflicting service token":
					want = 401
					n = 0
				case "current authority denied", "method denied", "IAM denied":
					want = 403
				case "default member unavailable", "503":
					want = 503
				case "not found":
					want = 404
				case "409":
					want = 409
				case "429":
					want = 429
				}
				h := newTransport(t, cfg, defaultServerTLS(t), func(c context.Context, r any, info *grpc.UnaryServerInfo, next grpc.UnaryHandler) (any, error) {
					if info.FullMethod == userv1.UserService_Me_FullMethodName {
						if mode == "Me failure" {
							return nil, status.Error(codes.Unauthenticated, "private-sentinel")
						}
						u := shared.MeUserData()
						u.Permissions = map[string]bool{"iam.user.create": true}
						u.MustChangePassword = mode == "restricted actor"
						return u, nil
					}
					if info.FullMethod != userv1.UserService_CreateUser_FullMethodName {
						t.Error("unexpected method", info.FullMethod)
					}
					switch mode {
					case "current authority denied", "method denied", "IAM denied":
						return nil, status.Error(codes.PermissionDenied, "private-sentinel")
					case "default member unavailable":
						return nil, status.Error(codes.Unavailable, "private-member-sentinel")
					case "not found":
						return nil, status.Error(codes.NotFound, "private-sentinel")
					case "409", "429", "503":
						return nil, testutils.ServiceFieldError(t, want)
					}
					if mode == "old constructor default" || mode == "success" {
						return next(c, r)
					}
					u := shared.UserDetailResponse(23).User
					u.Email = "user.name@example.com"
					u.Status = "PENDING"
					switch mode {
					case "max version":
						u.Version = 9223372036854775807
					case "wrong email":
						u.Email = "other@example.com"
					case "wrong status":
						u.Status = "ACTIVE"
					case "invalid result":
						u = nil
					}
					return &userv1.CreateUserResponse{User: u}, nil
				})
				defer h.close(t)
				if _, err := h.client.CreateUser(nil, &userv1.CreateUserRequest{}); err == nil {
					t.Fatal("nil context")
				}
				if _, err := h.client.CreateUser(context.Background(), nil); err == nil {
					t.Fatal("nil request")
				}
				before, _ := h.snapshot()
				if len(before) != 0 {
					t.Fatal("guard dispatch")
				}
				md := metadata.Pairs("x-request-id", "creation-wire", "authorization", "Bearer parent", "version", "old", "idempotency-key", "inherited")
				if mode == "conflicting service token" {
					md.Set("x-service-token", "conflicting")
				}
				parent, cancel := context.WithTimeout(metadata.NewOutgoingContext(context.Background(), md), time.Second)
				defer cancel()
				app := shared.NewApp(t, h.client, func(c fiber.Ctx) error { c.SetContext(parent); return c.Next() })
				req := creationWireRequest()
				if mode == "missing key" {
					req.Header.Del("Idempotency-Key")
				}
				if mode == "empty object" || mode == "null" {
					raw := "{}"
					if mode == "null" {
						raw = "null"
					}
					req = httptest.NewRequest("POST", "/api/v1/users", strings.NewReader(raw))
					req.Header = creationWireRequest().Header
				}
				res, err := app.Test(req, fiber.TestConfig{})
				body := testutils.ResponseBody(t, res, err)
				if res.StatusCode != want || res.Header.Get("Cache-Control") != "no-store" || len(res.Cookies()) != 0 {
					t.Fatal(mode, res.StatusCode, want, body)
				}
				calls, dials := h.snapshot()
				if len(calls) != n || n > 0 && dials != 1 {
					t.Fatal("calls/dials", len(calls), dials)
				}
				deadline, _ := parent.Deadline()
				for i, c := range calls {
					method := userv1.UserService_Me_FullMethodName
					if i == 1 {
						method = userv1.UserService_CreateUser_FullMethodName
					}
					if c.method != method || c.tls != tls.VersionTLS13 || c.deadline.IsZero() || c.deadline.After(deadline) {
						t.Fatal("method TLS deadline")
					}
					assertMetadata(t, c.metadata, "x-service-token", cfg.ServiceToken)
					assertMetadata(t, c.metadata, "authorization", userBearer)
					assertMetadata(t, c.metadata, "x-request-id", "creation-wire")
					assertMetadata(t, c.metadata, locale.MetadataKey, "id-ID")
					if i == 1 {
						country, initial := "+62", "PENDING"
						payload := &userv1.CreateUserRequest{Email: " User.Name@Example.COM ", Password: "simple", Name: "NASA O'Neil-Jean", Address: " X\r\nY ", Phone: "0628123456789", CountryCode: &country, Status: &initial}
						if !proto.Equal(c.request.(*userv1.CreateUserRequest), payload) || len(c.metadata.Get("version")) != 0 {
							t.Fatal("payload/presence")
						}
						assertMetadata(t, c.metadata, "idempotency-key", "opaque, key")
					}
					for _, key := range []string{"namespace", "user_id", "cookie", "password", "email"} {
						if len(c.metadata.Get(key)) != 0 {
							t.Fatal("private metadata", key)
						}
					}
				}
				if want == 200 {
					user := strings.TrimSuffix(strings.SplitN(shared.UserDetailBody, `"data":`, 2)[1], "}")
					user = strings.Replace(user, "target@example.com", "user.name@example.com", 1)
					user = strings.Replace(user, `"status":"ACTIVE"`, `"status":"PENDING"`, 1)
					if mode == "max version" {
						user = strings.Replace(user, "9007199254740993", "9223372036854775807", 1)
					}
					expected := `{"code":"SS","message":"User created successfully.","data":{"user":` + user + `}}`
					if body != expected {
						t.Fatal(body, expected)
					}
				} else if !strings.Contains(body, `"data":[]`) {
					t.Fatal(body)
				}
				if mode == "409" || mode == "429" || mode == "503" {
					testutils.AssertServiceErrorResponse(t, res, body, want)
				}
				assertNoSecrets(t, body, cfg.ServiceToken)
				for _, secret := range []string{"simple", "private-sentinel", "private-member-sentinel", "private-password", "private-dob"} {
					if strings.Contains(body, secret) {
						t.Fatal("secret leaked")
					}
				}
				after, _ := metadata.FromOutgoingContext(parent)
				if !reflect.DeepEqual(after, md) {
					t.Fatal("parent changed")
				}
			})
		})
	}
}

func creationWireContext(t *testing.T) {
	for _, mode := range []string{"client timeout", "parent deadline", "parent cancellation", "already canceled", "already expired"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				entered, finished := make(chan struct{}), make(chan struct{})
				cfg := secureConfig(t)
				h := newTransport(t, cfg, defaultServerTLS(t), func(c context.Context, r any, info *grpc.UnaryServerInfo, _ grpc.UnaryHandler) (any, error) {
					if info.FullMethod == userv1.UserService_Me_FullMethodName {
						u := shared.MeUserData()
						u.Permissions["iam.user.create"] = true
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
				res, err := app.Test(creationWireRequest(), fiber.TestConfig{})
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
				if len(calls) != n || n == 2 && (calls[0].method != userv1.UserService_Me_FullMethodName || calls[1].method != userv1.UserService_CreateUser_FullMethodName || dials != 1) {
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
