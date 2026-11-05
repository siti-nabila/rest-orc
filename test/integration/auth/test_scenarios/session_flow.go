package test_scenarios

import (
	"context"
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	authv1 "github.com/siti-nabila/api-contracts/pb/auth/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/rest-orc/internal/client/authservice"
	"github.com/siti-nabila/rest-orc/internal/client/grpcclient"
	authhandler "github.com/siti-nabila/rest-orc/internal/features/auth/handler"
	clientmocks "github.com/siti-nabila/rest-orc/test/client/authservice/mocks"
	grpcfixtures "github.com/siti-nabila/rest-orc/test/client/grpcclient/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func newSessionApp(t *testing.T, user *mocks.Service, sdk *clientmocks.SessionService, before ...fiber.Handler) *fiber.App {
	t.Helper()
	transport, err := grpcclient.New(grpcfixtures.ValidConfig())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := transport.Close(); err != nil {
			t.Error(err)
		}
	})
	client, err := authservice.NewWithSessionService(transport, user, &clientmocks.ProfileService{}, &clientmocks.PasswordService{}, sdk)
	if err != nil {
		t.Fatal(err)
	}
	return fixtures.NewApp(t, client, before...)
}
func sessionHTTP(t *testing.T, app *fiber.App, method, path, payload string, cookies []*http.Cookie, headers map[string]string) (*http.Response, string) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(payload))
	req.Header.Set("Origin", fixtures.BrowserOrigin)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "id-ID")
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	for k, v := range headers {
		if k == "Origin" && v == "" {
			req.Header.Del(k)
		} else {
			req.Header.Set(k, v)
		}
	}
	res, err := app.Test(req)
	body := testutils.ResponseBody(t, res, err)
	if res.Header.Get("Cache-Control") != "no-store" {
		t.Fatal("lifecycle response missing no-store")
	}
	return res, body
}
func csrfFromResponse(t *testing.T, body string) string {
	t.Helper()
	var envelope struct {
		Data struct {
			Token string `json:"csrf_token"`
		}
	}
	if err := json.Unmarshal([]byte(body), &envelope); err != nil || envelope.Data.Token == "" {
		t.Fatal("bootstrap data invalid")
	}
	return envelope.Data.Token
}
func proofCookies(refresh, csrf string) []*http.Cookie {
	return []*http.Cookie{{Name: authhandler.RefreshCookie, Value: refresh}, {Name: authhandler.CSRFCookie, Value: csrf}}
}

func sessionBrowserFlow(t *testing.T) {
	for _, mode := range []string{"bearer", "cookie"} {
		t.Run(mode, func(t *testing.T) {
			user, unexpected := rejectingAuthService(t)
			sequence := []string{}
			parentMD := metadata.Pairs("authorization", "Bearer unrelated-parent", "x-request-id", "session-request")
			parent := metadata.NewOutgoingContext(context.Background(), parentMD)
			check := func(ctx context.Context, want string) {
				testutils.AssertServiceContext(t, ctx)
				md, _ := metadata.FromOutgoingContext(ctx)
				if !reflect.DeepEqual(md.Get("x-request-id"), []string{"session-request"}) {
					t.Fatal("request metadata lost")
				}
				if want == "" {
					if len(md.Get("authorization")) != 0 {
						t.Fatal("cookie/public RPC leaked authorization")
					}
				} else if !reflect.DeepEqual(md.Get("authorization"), []string{want}) {
					t.Fatal("selected bearer differs")
				}
			}
			user.LoginFunc = func(ctx context.Context, r *userv1.AuthRequest, _ ...grpc.CallOption) (*userv1.UserTokenResponse, error) {
				sequence = append(sequence, "login")
				check(ctx, "")
				return fixtures.LoginTokenResponse(), nil
			}
			sdk := &clientmocks.SessionService{
				RefreshFunc: func(ctx context.Context, r *authv1.RefreshSessionRequest, _ ...grpc.CallOption) (*userv1.UserTokenResponse, error) {
					sequence = append(sequence, "refresh")
					check(ctx, "")
					if r.RefreshToken != "private-refresh" {
						t.Fatal("body/bearer overrode cookie")
					}
					res := fixtures.LoginTokenResponse()
					res.Token = "rotated-access"
					res.RefreshToken = "rotated-refresh"
					return res, nil
				},
				LogoutFunc: func(ctx context.Context, r *authv1.LogoutRequest, _ ...grpc.CallOption) (*authv1.LogoutResponse, error) {
					sequence = append(sequence, "logout")
					want := ""
					if mode == "bearer" {
						want = "Bearer access"
						if r.RefreshToken != nil {
							t.Fatal("bearer RPC included refresh")
						}
					} else if r.RefreshToken == nil || *r.RefreshToken != "rotated-refresh" {
						t.Fatal("cookie optional proof incorrect")
					}
					check(ctx, want)
					return &authv1.LogoutResponse{Changed: false}, nil
				},
			}
			app := newSessionApp(t, user, sdk, func(c fiber.Ctx) error { c.SetContext(parent); return c.Next() })
			res, body := sessionHTTP(t, app, "POST", "/api/v1/users/login", `{"email":"a@b.co","password":"secret"}`, nil, nil)
			if res.StatusCode != 200 || body != fixtures.LoginSuccessBody || len(res.Cookies()) != 2 {
				t.Fatal("login lifecycle failed")
			}
			res, body = sessionHTTP(t, app, "GET", "/api/v1/auth/csrf", "", []*http.Cookie{{Name: authhandler.RefreshCookie, Value: "private-refresh"}}, nil)
			if res.StatusCode != 200 || len(sequence) != 1 || len(res.Cookies()) != 1 || res.Cookies()[0].Name != authhandler.CSRFCookie {
				t.Fatal("bootstrap called RPC or changed refresh")
			}
			csrf := csrfFromResponse(t, body)
			res, body = sessionHTTP(t, app, "POST", "/api/v1/auth/refresh?refresh_token=forged&namespace=clm", `{"refresh_token":"forged","user_id":999}`, proofCookies("private-refresh", csrf), map[string]string{authhandler.CSRFHeader: csrf, "Authorization": "Bearer expired", "X-Service-Token": "browser-forged"})
			if res.StatusCode != 200 || len(res.Cookies()) != 2 || strings.Contains(body, "rotated-refresh") || !strings.Contains(body, `"token":"rotated-access"`) {
				t.Fatal("refresh lifecycle failed")
			}
			var rotatedCSRF string
			for _, cookie := range res.Cookies() {
				if cookie.Name == authhandler.CSRFCookie {
					rotatedCSRF = cookie.Value
				}
				if cookie.Path != "/api/v1/auth" || cookie.Domain != "" || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
					t.Fatal("cookie attrs differ")
				}
			}
			bad, _ := sessionHTTP(t, app, "POST", "/api/v1/auth/refresh", "", proofCookies("rotated-refresh", csrf), map[string]string{authhandler.CSRFHeader: csrf})
			if bad.StatusCode != 403 || len(sequence) != 2 {
				t.Fatal("old binding accepted")
			}
			headers := map[string]string{authhandler.CSRFHeader: rotatedCSRF}
			if mode == "bearer" {
				headers = map[string]string{"Authorization": "Bearer access", "Origin": ""}
			}
			res, body = sessionHTTP(t, app, "POST", "/api/v1/auth/logout", "", proofCookies("rotated-refresh", rotatedCSRF), headers)
			if res.StatusCode != 200 || body != `{"code":"SS","message":"Logout successfully.","data":{"changed":false}}` {
				t.Fatal("logout lifecycle failed")
			}
			testutils.AssertRefreshCookie(t, res, true)
			if *unexpected != 0 || !reflect.DeepEqual(sequence, []string{"login", "refresh", "logout"}) {
				t.Fatal("unexpected RPC/Me/retry")
			}
			md, _ := metadata.FromOutgoingContext(parent)
			if !reflect.DeepEqual(md, parentMD) {
				t.Fatal("parent mutated")
			}
		})
	}
}

func sessionLocalDenials(t *testing.T) {
	user, unexpected := rejectingAuthService(t)
	calls := 0
	sdk := &clientmocks.SessionService{RefreshFunc: func(context.Context, *authv1.RefreshSessionRequest, ...grpc.CallOption) (*userv1.UserTokenResponse, error) {
		calls++
		return fixtures.LoginTokenResponse(), nil
	}, LogoutFunc: func(context.Context, *authv1.LogoutRequest, ...grpc.CallOption) (*authv1.LogoutResponse, error) {
		calls++
		return &authv1.LogoutResponse{}, nil
	}}
	app := newSessionApp(t, user, sdk)
	bootstrap, body := sessionHTTP(t, app, "GET", "/api/v1/auth/csrf", "", []*http.Cookie{{Name: authhandler.RefreshCookie, Value: "opaque"}}, nil)
	if bootstrap.StatusCode != 200 {
		t.Fatal("bootstrap failed")
	}
	csrf := csrfFromResponse(t, body)
	for _, tc := range []struct {
		path    string
		cookies []*http.Cookie
		headers map[string]string
		status  int
	}{
		{"/api/v1/auth/refresh", nil, nil, 401},
		{"/api/v1/auth/refresh", proofCookies("opaque", csrf), nil, 403},
		{"/api/v1/auth/refresh", append(proofCookies("opaque", csrf), &http.Cookie{Name: authhandler.RefreshCookie, Value: "second"}), map[string]string{authhandler.CSRFHeader: csrf}, 401},
		{"/api/v1/auth/logout", proofCookies("opaque", csrf), map[string]string{authhandler.CSRFHeader: csrf, "Authorization": ""}, 401},
		{"/api/v1/auth/logout", proofCookies("opaque", csrf), map[string]string{authhandler.CSRFHeader: csrf, "Authorization": "malformed"}, 401},
		{"/api/v1/auth/refresh", proofCookies("opaque", csrf), map[string]string{authhandler.CSRFHeader: csrf, "Origin": "null"}, 403},
		{"/api/v1/auth/refresh", proofCookies("opaque", csrf), map[string]string{authhandler.CSRFHeader: "invalid"}, 403},
	} {
		res, _ := sessionHTTP(t, app, "POST", tc.path, "", tc.cookies, tc.headers)
		if res.StatusCode != tc.status || len(res.Cookies()) != 0 {
			t.Fatalf("local denial %s=%d want %d", tc.path, res.StatusCode, tc.status)
		}
	}
	if calls != 0 || *unexpected != 0 {
		t.Fatal("local denial invoked RPC")
	}
}

func sessionDependencyFailures(t *testing.T) {
	for _, op := range []string{"refresh", "logout"} {
		for _, mode := range []string{"nil", "missing credential", "invalid timestamp", "expired", "reuse", "restricted", "revoked", "wrong bearer", "409", "429", "503"} {
			t.Run(op+"/"+mode, func(t *testing.T) {
				user, unexpected := rejectingAuthService(t)
				calls := 0
				want := 500
				var failure error
				switch mode {
				case "reuse", "revoked", "wrong bearer":
					want = 401
					failure = status.Error(codes.Unauthenticated, "private credential failure")
				case "restricted":
					want = 403
					failure = status.Error(codes.PermissionDenied, "private restricted reason")
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
				if op == "logout" && (mode == "missing credential" || mode == "invalid timestamp" || mode == "expired") {
					return
				}
				sdk := &clientmocks.SessionService{
					RefreshFunc: func(context.Context, *authv1.RefreshSessionRequest, ...grpc.CallOption) (*userv1.UserTokenResponse, error) {
						calls++
						if failure != nil || mode == "nil" {
							return nil, failure
						}
						r := fixtures.LoginTokenResponse()
						switch mode {
						case "missing credential":
							r.RefreshToken = ""
						case "invalid timestamp":
							r.AccessExpiresAt = &timestamppb.Timestamp{Seconds: 1 << 62}
						case "expired":
							r.AccessExpiresAt = timestamppb.New(fixtures.SessionNow())
						}
						return r, nil
					},
					LogoutFunc: func(context.Context, *authv1.LogoutRequest, ...grpc.CallOption) (*authv1.LogoutResponse, error) {
						calls++
						return nil, failure
					},
				}
				app := newSessionApp(t, user, sdk)
				_, body := sessionHTTP(t, app, "GET", "/api/v1/auth/csrf", "", []*http.Cookie{{Name: authhandler.RefreshCookie, Value: "opaque"}}, nil)
				csrf := csrfFromResponse(t, body)
				headers := map[string]string{authhandler.CSRFHeader: csrf}
				if mode == "wrong bearer" && op == "logout" {
					headers["Authorization"] = "Bearer invalid"
				}
				res, body := sessionHTTP(t, app, "POST", "/api/v1/auth/"+op, "", proofCookies("opaque", csrf), headers)
				if res.StatusCode != want || calls != 1 || *unexpected != 0 || len(res.Cookies()) != 0 || strings.Contains(body, "private") {
					t.Fatalf("dependency result status=%d want=%d calls=%d", res.StatusCode, want, calls)
				}
				if mode == "409" || mode == "429" || mode == "503" {
					testutils.AssertServiceErrorResponse(t, res, body, want)
				}
			})
		}
	}
}

func sessionRegistrationAndLoginResults(t *testing.T) {
	for _, op := range []string{"register", "login"} {
		for _, mode := range []string{"nil", "missing", "inconsistent", "expired", "restricted success"} {
			t.Run(op+"/"+mode, func(t *testing.T) {
				if op == "register" && (mode == "expired" || mode == "restricted success") {
					return
				}
				user, unexpected := rejectingAuthService(t)
				calls := 0
				fn := func(context.Context, *userv1.AuthRequest, ...grpc.CallOption) (*userv1.UserTokenResponse, error) {
					calls++
					if mode == "nil" {
						return nil, nil
					}
					if op == "register" {
						r := fixtures.RegistrationResponse()
						if mode == "missing" {
							r.RegistrationId = 0
						} else {
							r.RefreshToken = "private-refresh"
						}
						return r, nil
					}
					r := fixtures.LoginTokenResponse()
					switch mode {
					case "missing":
						r.SessionId = ""
					case "inconsistent":
						r.AccessExpiresAt = timestamppb.New(fixtures.SessionNow().AddDate(1, 0, 0))
					case "expired":
						r.SessionExpiresAt = timestamppb.New(fixtures.SessionNow())
					case "restricted success":
						r.MustChangePassword = true
					}
					return r, nil
				}
				path := "/api/v1/users/login"
				user.LoginFunc = fn
				if op == "register" {
					path = "/api/v1/users/create"
					user.RegisterFunc = fn
				}
				res, body := sessionHTTP(t, newAuthApp(t, user), "POST", path, `{"email":"a@b.co","password":"secret"}`, nil, nil)
				want := 500
				if mode == "restricted success" {
					want = 200
				}
				if res.StatusCode != want || calls != 1 || *unexpected != 0 || strings.Contains(body, "private-refresh") {
					t.Fatal("session result guard failed")
				}
				if want == 500 && len(res.Cookies()) != 0 {
					t.Fatal("invalid success changed cookies")
				}
				if mode == "restricted success" && !strings.Contains(body, `"must_change_password":true`) {
					t.Fatal("restriction flag lost")
				}
			})
		}
	}
}
