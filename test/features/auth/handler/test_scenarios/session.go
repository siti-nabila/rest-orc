package test_scenarios

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	authhandler "github.com/siti-nabila/rest-orc/internal/features/auth/handler"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/test/features/auth/handler/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func SessionScenarios() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "session handlers preserve context results and cookie success boundary", Run: sessionHandlers},
		{Name: "session handler constructors and nonce failures fail before dispatch", Run: sessionHandlerGuards},
		{Name: "concurrent browser proofs and input config remain isolated", Run: browserConcurrent},
	}
}
func sessionHandlers(t *testing.T) {
	for _, op := range []string{"refresh", "bearer logout", "cookie logout"} {
		for _, fail := range []bool{false, true} {
			t.Run(op+map[bool]string{false: " success", true: " error"}[fail], func(t *testing.T) {
				p := browserPolicy(t)
				token, _ := bootstrapProof(t, p, "original")
				calls := 0
				parent := context.WithValue(context.Background(), requestIDKey, "session-request")
				check := func(c context.Context) {
					calls++
					if c != parent {
						t.Fatal("handler changed parent context")
					}
				}
				refresh := &mocks.RefreshUseCase{ExecuteFunc: func(c context.Context, raw string) (domain.SessionResult, error) {
					check(c)
					if raw != "original" {
						t.Fatal("refresh input changed")
					}
					if fail {
						return domain.SessionResult{}, common.ErrServiceUnavailable
					}
					r := fixtures.LoginResult()
					r.RefreshToken = "rotated"
					return r, nil
				}}
				logout := &mocks.LogoutUseCase{ExecuteFunc: func(c context.Context, proof domain.LogoutCredential) (domain.LogoutResponse, error) {
					check(c)
					want := domain.LogoutCredential{RefreshToken: "original"}
					if op == "bearer logout" {
						want = domain.LogoutCredential{Authorization: "Bearer access"}
					}
					if proof != want {
						t.Fatal("wrong logout mode")
					}
					if fail {
						return domain.LogoutResponse{}, common.ErrServiceUnavailable
					}
					return domain.LogoutResponse{Changed: false}, nil
				}}
				app, writer := testutils.NewHTTPApp(fiber.Config{DisableDefaultDate: true})
				app.Use(func(c fiber.Ctx) error { c.SetContext(parent); return c.Next() })
				h, err := authhandler.NewSession(refresh, logout, writer, p)
				if err != nil {
					t.Fatal(err)
				}
				h.RegisterRoutes(app.Group("/auth"))
				path := "/auth/logout"
				if op == "refresh" {
					path = "/auth/refresh"
				}
				req := httptest.NewRequest("POST", path+"?refresh_token=forged", strings.NewReader(`{"refresh_token":"forged","user_id":999}`))
				req.Header.Set("Origin", fixtures.BrowserOrigin)
				req.AddCookie(&http.Cookie{Name: authhandler.RefreshCookie, Value: "original"})
				req.AddCookie(&http.Cookie{Name: authhandler.CSRFCookie, Value: token})
				req.Header.Set(authhandler.CSRFHeader, token)
				if op == "bearer logout" {
					req.Header.Set("Authorization", "Bearer access")
					req.Header.Del(authhandler.CSRFHeader)
				} else if op == "refresh" {
					req.Header.Set("Authorization", "Bearer expired")
				}
				res, err := app.Test(req)
				body := testutils.ResponseBody(t, res, err)
				want := 200
				if fail {
					want = 503
				}
				if res.StatusCode != want || calls != 1 || res.Header.Get("Cache-Control") != "no-store" {
					t.Fatalf("status=%d calls=%d", res.StatusCode, calls)
				}
				if fail {
					if len(res.Cookies()) != 0 {
						t.Fatal("error changed browser state")
					}
				} else if op != "refresh" {
					if body != `{"code":"SS","message":"Logout successfully.","data":{"changed":false}}` {
						t.Fatal("false/logout envelope changed")
					}
					testutils.AssertRefreshCookie(t, res, true)
				} else {
					if len(res.Cookies()) != 2 || strings.Contains(body, "rotated") {
						t.Fatal("refresh output/cookies invalid")
					}
				}
			})
		}
	}
}
func sessionHandlerGuards(t *testing.T) {
	p := browserPolicy(t)
	app, writer := testutils.NewHTTPApp(fiber.Config{DisableDefaultDate: true})
	calls := 0
	r := &mocks.RefreshUseCase{ExecuteFunc: func(context.Context, string) (domain.SessionResult, error) {
		calls++
		return fixtures.LoginResult(), nil
	}}
	l := &mocks.LogoutUseCase{ExecuteFunc: func(context.Context, domain.LogoutCredential) (domain.LogoutResponse, error) {
		calls++
		return domain.LogoutResponse{}, nil
	}}
	for _, tc := range []struct {
		r    authhandler.RefreshUseCase
		l    authhandler.LogoutUseCase
		want error
	}{{nil, l, dictionary.ErrAuthHandlerUseCaseRequired}, {r, nil, dictionary.ErrAuthHandlerUseCaseRequired}} {
		if _, err := authhandler.NewSession(tc.r, tc.l, writer, p); !errors.Is(err, tc.want) {
			t.Fatal("nil usecase accepted")
		}
	}
	if _, err := authhandler.NewSession(r, l, nil, p); !errors.Is(err, dictionary.ErrAuthHandlerResponseWriterRequired) {
		t.Fatal("nil writer accepted")
	}
	if _, err := authhandler.NewSession(r, l, writer, nil); !errors.Is(err, dictionary.ErrBrowserPolicyRequired) {
		t.Fatal("nil policy accepted")
	}
	token, _ := bootstrapProof(t, p, "original")
	broken, _ := authhandler.NewBrowserPolicy(fixtures.BrowserConfig(), "/api/v1/auth", fixtures.SessionNow, strings.NewReader(""))
	h, _ := authhandler.NewSession(r, l, writer, broken)
	h.RegisterRoutes(app.Group("/auth"))
	req := httptest.NewRequest("POST", "/auth/refresh", nil)
	req.Header.Set("Origin", fixtures.BrowserOrigin)
	req.Header.Set(authhandler.CSRFHeader, token)
	req.AddCookie(&http.Cookie{Name: authhandler.RefreshCookie, Value: "original"})
	req.AddCookie(&http.Cookie{Name: authhandler.CSRFCookie, Value: token})
	res, err := app.Test(req)
	testutils.ResponseBody(t, res, err)
	if res.StatusCode != 500 || calls != 0 || len(res.Cookies()) != 0 {
		t.Fatal("nonce failure dispatched or changed cookies")
	}
	login := &mocks.LoginUseCase{ExecuteFunc: func(context.Context, domain.AuthRequest) (domain.SessionResult, error) {
		calls++
		return fixtures.LoginResult(), nil
	}}
	auth, _ := authhandler.New(unexpectedAuthUseCase(t), login, unexpectedPasswordUseCase(t), writer, broken)
	auth.RegisterPublicRoutes(app.Group("/users"))
	req = httptest.NewRequest("POST", "/users/login", strings.NewReader(`{"email":"a@b.co","password":"secret"}`))
	req.Header.Set("Origin", fixtures.BrowserOrigin)
	req.Header.Set("Content-Type", "application/json")
	res, err = app.Test(req)
	testutils.ResponseBody(t, res, err)
	if res.StatusCode != 500 || calls != 0 || len(res.Cookies()) != 0 {
		t.Fatal("Login nonce failure dispatched")
	}
}

func browserConcurrent(t *testing.T) {
	cfg := fixtures.BrowserConfig()
	p, err := authhandler.NewBrowserPolicy(cfg, "/api/v1/auth", fixtures.SessionNow, strings.NewReader(strings.Repeat("n", 32768)))
	if err != nil {
		t.Fatal(err)
	}
	cfg.AllowedOrigins[0] = "https://evil.test"
	cfg.CSRFSecret = "changed"
	app, _ := testutils.NewHTTPApp(fiber.Config{DisableDefaultDate: true})
	app.Get("/csrf", func(c fiber.Ctx) error {
		r, e := p.Bootstrap(c)
		if e != nil {
			return e
		}
		return c.JSON(r)
	})
	app.Post("/refresh", func(c fiber.Ctx) error {
		raw, e := p.RefreshCredential(c)
		if e != nil {
			return e
		}
		return c.SendString(raw)
	})
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			raw := strings.Repeat("x", i+1)
			token, _ := bootstrapProof(t, p, raw)
			req := httptest.NewRequest("POST", "/refresh", nil)
			req.Header.Set("Origin", fixtures.BrowserOrigin)
			req.Header.Set(authhandler.CSRFHeader, token)
			req.AddCookie(&http.Cookie{Name: authhandler.RefreshCookie, Value: raw})
			req.AddCookie(&http.Cookie{Name: authhandler.CSRFCookie, Value: token})
			res, err := app.Test(req)
			body := testutils.ResponseBody(t, res, err)
			if res.StatusCode != 200 || body != raw {
				t.Error("concurrent proof or config leaked")
			}
		}(i)
	}
	wg.Wait()
}
