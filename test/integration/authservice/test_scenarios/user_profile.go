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
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func targetProfileWireRequest() *http.Request {
	req := httptest.NewRequest("PATCH", "/api/v1/users/00023/profile?user_id=99&version=1&namespace=forged", strings.NewReader(`{"name":"  NASA  o'neil-jean  ","address":" X\r\nY ","country_code":"62","phone":"0812345678901","user_id":99,"version":1,"field_mask":["password"]}`))
	req.Header.Set("Authorization", userBearer)
	req.Header.Set("Origin", shared.BrowserOrigin)
	req.Header.Set(locale.HTTPHeader, "id-ID")
	req.Header.Set("X-Service-Token", browserToken)
	req.Header.Set("Idempotency-Key", "opaque, key")
	req.Header.Set("Version", "9007199254740993")
	return req
}
func targetProfileWireOutcomes(t *testing.T) {
	for _, mode := range []string{"success", "unkeyed", "local no-op", "null no-op", "unkeyed no-op", "invalid no-op key", "missing no-op version", "changed", "max version", "no growth replay", "wrong target", "invalid result", "restricted actor", "restricted no-op", "Me failure", "current authority denied", "method denied", "IAM denied", "not found", "409", "429", "503", "conflicting service token"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				cfg := secureConfig(t)
				want := 200
				noOp := strings.Contains(mode, "no-op")
				switch mode {
				case "wrong target", "invalid result":
					want = 500
				case "restricted actor", "restricted no-op", "current authority denied", "method denied", "IAM denied":
					want = 403
				case "Me failure", "conflicting service token":
					want = 401
				case "invalid no-op key", "missing no-op version":
					want = 400
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
						u.Permissions = map[string]bool{"iam.user.update": true}
						u.MustChangePassword = mode == "restricted actor" || mode == "restricted no-op"
						return u, nil
					}
					if info.FullMethod != userv1.UserService_UpdateUser_FullMethodName {
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
					result := &userv1.UpdateUserResponse{User: shared.UserDetailResponse(23).User, Changed: mode == "changed"}
					switch mode {
					case "max version":
						result.User.Version = 9223372036854775807
					case "no growth replay":
						result.User.Version = 1
					case "wrong target":
						result.User.Id = 99
					case "invalid result":
						result.User = nil
					}
					if mode == "success" || mode == "unkeyed" {
						return next(c, r)
					}
					return result, nil
				})
				defer h.close(t)
				if _, err := h.client.UpdateUser(nil, &userv1.UpdateUserRequest{}); err == nil {
					t.Fatal("nil context")
				}
				if _, err := h.client.UpdateUser(context.Background(), nil); err == nil {
					t.Fatal("nil request")
				}
				before, _ := h.snapshot()
				if len(before) != 0 {
					t.Fatal("guard dispatch")
				}
				md := metadata.Pairs("x-request-id", "target-profile-wire", "authorization", "Bearer parent", "version", "old", "idempotency-key", "inherited")
				if mode == "conflicting service token" {
					md.Set("x-service-token", "conflicting")
				}
				parent, cancel := context.WithTimeout(metadata.NewOutgoingContext(context.Background(), md), time.Second)
				defer cancel()
				app := shared.NewApp(t, h.client, func(c fiber.Ctx) error { c.SetContext(parent); return c.Next() })
				req := targetProfileWireRequest()
				if noOp {
					raw := "{}"
					if mode == "null no-op" {
						raw = "null"
					}
					req = httptest.NewRequest("PATCH", "/api/v1/users/00023/profile", strings.NewReader(raw))
					for k, v := range targetProfileWireRequest().Header {
						req.Header[k] = v
					}
				}
				if mode == "unkeyed" || mode == "unkeyed no-op" {
					req.Header.Del("Idempotency-Key")
				}
				if mode == "invalid no-op key" {
					req.Header.Set("Idempotency-Key", "")
				}
				if mode == "missing no-op version" {
					req.Header.Del("Version")
				}
				res, err := app.Test(req, fiber.TestConfig{})
				body := testutils.ResponseBody(t, res, err)
				if res.StatusCode != want || res.Header.Get("Cache-Control") != "no-store" || len(res.Cookies()) != 0 {
					t.Fatal(res.StatusCode, want, body)
				}
				calls, dials := h.snapshot()
				n := 2
				if noOp || mode == "restricted actor" || mode == "Me failure" {
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
						method = userv1.UserService_UpdateUser_FullMethodName
					}
					if c.method != method || c.tls != tls.VersionTLS13 || c.deadline.IsZero() || c.deadline.After(deadline) {
						t.Fatal("method/TLS/deadline")
					}
					assertMetadata(t, c.metadata, "x-service-token", cfg.ServiceToken)
					assertMetadata(t, c.metadata, "authorization", userBearer)
					assertMetadata(t, c.metadata, "x-request-id", "target-profile-wire")
					assertMetadata(t, c.metadata, locale.MetadataKey, "id-ID")
					if i == 1 {
						name, address, phone, country := "NASA O'Neil-Jean", " X\r\nY ", "812345678901", "+62"
						version := uint64(9007199254740993)
						wantPayload := &userv1.UpdateUserRequest{UserId: 23, ExpectedUserVersion: &version, Name: &name, Address: &address, Phone: &phone, CountryCode: &country, FieldMask: &fieldmaskpb.FieldMask{Paths: []string{"name", "address", "phone", "country_code"}}}
						if !proto.Equal(c.request.(*userv1.UpdateUserRequest), wantPayload) || len(c.metadata.Get("version")) != 0 {
							t.Fatal("payload/presence", c.request, c.metadata)
						}
						if mode == "unkeyed" {
							if len(c.metadata.Get("idempotency-key")) != 0 {
								t.Fatal("inherited key")
							}
						} else {
							assertMetadata(t, c.metadata, "idempotency-key", "opaque, key")
						}
					}
					if len(c.metadata.Get("namespace")) != 0 || len(c.metadata.Get("user_id")) != 0 || len(c.metadata.Get("cookie")) != 0 {
						t.Fatal("browser authority")
					}
				}
				if want == 200 {
					expected := `{"code":"SS","message":"User profile updated successfully.","data":{"changed":false}}`
					if !noOp {
						user := strings.TrimSuffix(strings.SplitN(shared.UserDetailBody, `"data":`, 2)[1], "}")
						if mode == "max version" {
							user = strings.Replace(user, "9007199254740993", "9223372036854775807", 1)
						}
						if mode == "no growth replay" {
							user = strings.Replace(user, "9007199254740993", "1", 1)
						}
						changed := "false"
						if mode == "changed" {
							changed = "true"
						}
						expected = `{"code":"SS","message":"User profile updated successfully.","data":{"changed":` + changed + `,"user":` + user + `}}`
					}
					if body != expected {
						t.Fatal(body, expected)
					}
				} else if !strings.Contains(body, `"data":[]`) {
					t.Fatal(body)
				}
				if want == 409 || want == 429 || want == 503 {
					testutils.AssertServiceErrorResponse(t, res, body, want)
				}
				assertNoSecrets(t, body, cfg.ServiceToken)
				for _, secret := range []string{"private-sentinel", "private-password", "private-dob"} {
					if strings.Contains(body, secret) {
						t.Fatal(body)
					}
				}
				after, _ := metadata.FromOutgoingContext(parent)
				if !reflect.DeepEqual(after, md) {
					t.Fatal("parent mutated")
				}
			})
		})
	}
}
func targetProfileWireContext(t *testing.T) {
	for _, mode := range []string{"client timeout", "parent deadline", "parent cancellation", "already canceled", "already expired"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				entered, finished := make(chan struct{}), make(chan struct{})
				cfg := secureConfig(t)
				h := newTransport(t, cfg, defaultServerTLS(t), func(c context.Context, r any, info *grpc.UnaryServerInfo, _ grpc.UnaryHandler) (any, error) {
					if info.FullMethod == userv1.UserService_Me_FullMethodName {
						u := shared.MeUserData()
						u.Permissions["iam.user.update"] = true
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
				res, err := app.Test(targetProfileWireRequest(), fiber.TestConfig{})
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
				if len(calls) != n || n == 2 && (calls[0].method != userv1.UserService_Me_FullMethodName || calls[1].method != userv1.UserService_UpdateUser_FullMethodName || dials != 1) {
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
