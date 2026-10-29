package test_scenarios

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/gofiber/fiber/v3"
	authv1 "github.com/siti-nabila/api-contracts/pb/auth/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	sharedfixtures "github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func passwordWireOutcomes(t *testing.T) {
	valid := `{"current_password":" PaSs é ","new_password":"new-password","user_id":999,"namespace":"iam","roles":["admin"],"version":"ignored","confirm_password":"ignored"}`
	for _, tc := range []struct {
		name, payload                                                    string
		status                                                           int
		restricted, missing, mutation, changed, loginRequired, meFailure bool
		failure                                                          error
	}{
		{name: "non-admin success", payload: valid, status: 200, mutation: true, changed: true, loginRequired: true},
		{name: "restricted success", payload: valid, status: 200, restricted: true, mutation: true, changed: true, loginRequired: true},
		{name: "both flags false", payload: valid, status: 200, mutation: true},
		{name: "only login required", payload: valid, status: 200, mutation: true, loginRequired: true},
		{name: "only changed", payload: valid, status: 200, mutation: true, changed: true},
		{name: "local validation", payload: `{}`, status: 400},
		{name: "equal passwords", payload: `{"current_password":"same-password","new_password":"same-password"}`, status: 400},
		{name: "parsing failure", payload: `{"new_password":false}`, status: 400},
		{name: "missing bearer", payload: valid, status: 401, missing: true},
		{name: "Me denied", payload: valid, status: 401, meFailure: true, failure: status.Error(codes.Unauthenticated, "private-password-backend-sentinel")},
		{name: "method denied", payload: valid, status: 403, mutation: true, failure: status.Error(codes.PermissionDenied, "private-password-backend-sentinel")},
		{name: "iam namespace denied", payload: valid, status: 403, mutation: true, failure: status.Error(codes.PermissionDenied, "private-password-backend-sentinel")},
		{name: "conflict fields", payload: valid, status: 409, mutation: true, failure: testutils.PasswordFieldError(t, 409)},
		{name: "rate-limit fields", payload: valid, status: 429, mutation: true, failure: testutils.PasswordFieldError(t, 429)},
		{name: "unavailable fields", payload: valid, status: 503, mutation: true, failure: testutils.PasswordFieldError(t, 503)},
		{name: "internal dependency", payload: valid, status: 500, mutation: true, failure: status.Error(codes.Internal, "private-password-backend-sentinel")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				cfg := secureConfig(t)
				h := newTransport(t, cfg, defaultServerTLS(t), func(ctx context.Context, request any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
					if info.FullMethod == userv1.UserService_Me_FullMethodName {
						if tc.meFailure {
							return nil, tc.failure
						}
						user := sharedfixtures.MeUserData()
						user.MustChangePassword = tc.restricted
						return user, nil
					}
					if tc.failure != nil {
						return nil, tc.failure
					}
					result, err := handler(ctx, request)
					if err != nil {
						return nil, err
					}
					response := result.(*authv1.ChangePasswordResponse)
					response.Changed, response.LoginRequired = tc.changed, tc.loginRequired
					return response, nil
				})
				defer h.close(t)
				parent, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				original := metadata.Pairs("x-request-id", "password-wire")
				parent = metadata.NewOutgoingContext(parent, original)
				app := sharedfixtures.NewApp(t, h.client, func(ctx fiber.Ctx) error { ctx.SetContext(parent); return ctx.Next() })
				response, body := secureHTTPRequest(t, app, http.MethodPost, "/api/v1/me/password", tc.payload, !tc.missing)
				methods := []string{}
				if !tc.missing {
					methods = append(methods, userv1.UserService_Me_FullMethodName)
				}
				if tc.mutation {
					methods = append(methods, authv1.AuthService_ChangePassword_FullMethodName)
				}
				calls, dials := h.snapshot()
				assertHTTPCalls(t, calls, methods, cfg.ServiceToken, !tc.missing)
				wantDials := 1
				if tc.missing {
					wantDials = 0
				}
				if response.StatusCode != tc.status || dials != wantDials {
					t.Fatalf("status/dials=%d/%d want %d/%d", response.StatusCode, dials, tc.status, wantDials)
				}
				deadline, _ := parent.Deadline()
				for _, call := range calls {
					if call.tls != tls.VersionTLS13 || call.deadline.IsZero() || call.deadline.After(deadline) {
						t.Fatal("secure transport or earlier deadline lost")
					}
					assertMetadata(t, call.metadata, "x-request-id", "password-wire")
					for _, key := range []string{"cookie", "refresh_token", "version", "idempotency-key"} {
						if len(call.metadata.Get(key)) != 0 {
							t.Error("unsupported credential/precondition entered RPC metadata")
						}
					}
				}
				if tc.mutation {
					want := &authv1.ChangePasswordRequest{CurrentPassword: " PaSs é ", NewPassword: "new-password"}
					if !proto.Equal(calls[1].request.(*authv1.ChangePasswordRequest), want) {
						t.Error("wire password payload changed")
					}
				}
				md, _ := metadata.FromOutgoingContext(parent)
				if !reflect.DeepEqual(md, original) {
					t.Error("parent metadata mutated")
				}
				if tc.status == 200 {
					want := fmt.Sprintf(`{"code":"SS","message":"Password changed successfully.","data":{"changed":%t,"login_required":%t}}`, tc.changed, tc.loginRequired)
					if body != want || response.Header.Get(fiber.HeaderCacheControl) != "no-store" {
						t.Fatal("success result/cache contract changed")
					}
				} else if !strings.Contains(body, `"data":[]`) {
					t.Fatal("safe error data changed")
				}
				if tc.status == 409 || tc.status == 429 || tc.status == 503 {
					testutils.AssertPasswordErrorResponse(t, response, body, tc.status)
				}
				testutils.AssertRefreshCookie(t, response, tc.status == 200 && tc.loginRequired)
				assertNoSecrets(t, body, cfg.ServiceToken)
				for _, secret := range []string{" PaSs é ", "new-password", "same-password", "private-password-backend-sentinel"} {
					if strings.Contains(body, secret) {
						t.Error("password HTTP response exposed private data")
					}
				}
			})
		})
	}
}
