package test_scenarios

import (
	"context"
	"crypto/tls"
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	authv1 "github.com/siti-nabila/api-contracts/pb/auth/v1"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	authhandler "github.com/siti-nabila/rest-orc/internal/features/auth/handler"
	sharedfixtures "github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

func wireSessionRequest(op, refresh, csrf, bearer string) *http.Request {
	method := "POST"
	if op == "csrf" {
		method = "GET"
	}
	r := httptest.NewRequest(method, "/api/v1/auth/"+op+"?refresh_token=forged&namespace=clm", nil)
	r.Header.Set("Origin", sharedfixtures.BrowserOrigin)
	r.Header.Set(locale.HTTPHeader, "id-ID")
	r.Header.Set("X-Service-Token", browserToken)
	if refresh != "" {
		r.AddCookie(&http.Cookie{Name: authhandler.RefreshCookie, Value: refresh})
	}
	if csrf != "" {
		r.AddCookie(&http.Cookie{Name: authhandler.CSRFCookie, Value: csrf})
		r.Header.Set(authhandler.CSRFHeader, csrf)
	}
	if bearer != "" {
		r.Header.Set("Authorization", bearer)
	}
	return r
}
func wireBootstrap(t *testing.T, app *fiber.App, refresh string) string {
	t.Helper()
	res, err := app.Test(wireSessionRequest("csrf", refresh, "", ""), fiber.TestConfig{})
	body := testutils.ResponseBody(t, res, err)
	var envelope struct {
		Data struct {
			Token string `json:"csrf_token"`
		}
	}
	if res.StatusCode != 200 || json.Unmarshal([]byte(body), &envelope) != nil || envelope.Data.Token == "" || len(res.Cookies()) != 1 {
		t.Fatal("CSRF bootstrap failed")
	}
	return envelope.Data.Token
}
func sessionWireOutcomes(t *testing.T) {
	for _, mode := range []string{"refresh", "bearer logout", "cookie logout"} {
		for _, outcome := range []string{"success", "invalid proof", "restricted", "method denied", "iam namespace denied", "409", "429", "503"} {
			t.Run(mode+"/"+outcome, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					cfg := secureConfig(t)
					want := 200
					var failure error
					switch outcome {
					case "invalid proof":
						want = 401
						failure = status.Error(codes.Unauthenticated, "private-session-backend")
					case "restricted", "method denied", "iam namespace denied":
						want = 403
						failure = status.Error(codes.PermissionDenied, "private-session-backend")
					case "409":
						want = 409
						failure = testutils.ServiceFieldError(t, 409)
					case "429":
						want = 429
						failure = testutils.ServiceFieldError(t, 429)
					case "503":
						want = 503
						failure = testutils.ServiceFieldError(t, 503)
					}
					h := newTransport(t, cfg, defaultServerTLS(t), func(ctx context.Context, r any, info *grpc.UnaryServerInfo, next grpc.UnaryHandler) (any, error) {
						if failure != nil {
							return nil, failure
						}
						return next(ctx, r)
					})
					defer h.close(t)
					parent, cancel := context.WithTimeout(context.Background(), time.Second)
					defer cancel()
					original := metadata.Pairs("authorization", "Bearer parent-not-selected", "x-request-id", "session-wire")
					parent = metadata.NewOutgoingContext(parent, original)
					app := sharedfixtures.NewApp(t, h.client, func(c fiber.Ctx) error { c.SetContext(parent); return c.Next() })
					csrf := wireBootstrap(t, app, "opaque-refresh")
					calls, dials := h.snapshot()
					if len(calls) != 0 || dials != 0 {
						t.Fatal("bootstrap dispatched")
					}
					op, bearer, wantMethod := "logout", "", authv1.AuthService_Logout_FullMethodName
					if mode == "refresh" {
						op = "refresh"
						bearer = "Bearer expired"
						wantMethod = authv1.AuthService_RefreshSession_FullMethodName
					} else if mode == "bearer logout" {
						bearer = userBearer
					}
					res, err := app.Test(wireSessionRequest(op, "opaque-refresh", csrf, bearer), fiber.TestConfig{})
					body := testutils.ResponseBody(t, res, err)
					if res.StatusCode != want || res.Header.Get("Cache-Control") != "no-store" {
						t.Fatalf("status=%d want=%d", res.StatusCode, want)
					}
					calls, dials = h.snapshot()
					if len(calls) != 1 || dials != 1 || calls[0].method != wantMethod || calls[0].tls != tls.VersionTLS13 {
						t.Fatal("unexpected method/Me/retry/connection/TLS")
					}
					call := calls[0]
					assertMetadata(t, call.metadata, "x-service-token", cfg.ServiceToken)
					assertMetadata(t, call.metadata, "x-request-id", "session-wire")
					assertMetadata(t, call.metadata, locale.MetadataKey, "id-ID")
					deadline, _ := parent.Deadline()
					if call.deadline.IsZero() || call.deadline.After(deadline) {
						t.Fatal("earlier parent deadline extended")
					}
					for _, key := range []string{"namespace", "cookie", "refresh_token", "user_id"} {
						if len(call.metadata.Get(key)) != 0 {
							t.Fatal("browser authority leaked")
						}
					}
					if mode == "bearer logout" {
						assertMetadata(t, call.metadata, "authorization", userBearer)
						if call.request.(*authv1.LogoutRequest).RefreshToken != nil {
							t.Fatal("bearer included refresh")
						}
					} else {
						if len(call.metadata.Get("authorization")) != 0 {
							t.Fatal("cookie operation included bearer")
						}
						if mode == "refresh" {
							if call.request.(*authv1.RefreshSessionRequest).RefreshToken != "opaque-refresh" {
								t.Fatal("refresh payload changed")
							}
						} else {
							proof := call.request.(*authv1.LogoutRequest).RefreshToken
							if proof == nil || *proof != "opaque-refresh" {
								t.Fatal("optional cookie proof lost")
							}
						}
					}
					md, _ := metadata.FromOutgoingContext(parent)
					if !reflect.DeepEqual(md, original) {
						t.Fatal("parent metadata mutated")
					}
					if want == 200 {
						if op == "logout" {
							testutils.AssertRefreshCookie(t, res, true)
							if body != `{"code":"SS","message":"Logout successfully.","data":{"changed":false}}` {
								t.Fatal("false logout result lost")
							}
						} else if len(res.Cookies()) != 2 || !strings.Contains(body, `"token":"refreshed-token"`) {
							t.Fatal("refresh success invalid")
						}
					} else {
						testutils.AssertRefreshCookie(t, res, false)
					}
					if outcome == "409" || outcome == "429" || outcome == "503" {
						testutils.AssertServiceErrorResponse(t, res, body, want)
					}
					assertNoSecrets(t, body, cfg.ServiceToken)
					if strings.Contains(body, "opaque-refresh") || strings.Contains(body, "rotated-refresh") || strings.Contains(body, "private-session-backend") {
						t.Fatal("credential/internal message leaked")
					}
				})
			})
		}
	}
}
