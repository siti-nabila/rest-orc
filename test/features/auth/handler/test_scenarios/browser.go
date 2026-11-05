package test_scenarios

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	authhandler "github.com/siti-nabila/rest-orc/internal/features/auth/handler"
	"github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func BrowserScenarios() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "signed CSRF binds cookies with no session lookup and works across instances", Run: browserCSRF},
		{Name: "browser session cookie lifetime and errors are atomic", Run: browserCookies},
		{Name: "browser Origin media type and logout selection are explicit", Run: browserRequestPolicy},
	}
}

func browserPolicy(t *testing.T) *authhandler.BrowserPolicy {
	t.Helper()
	p, err := authhandler.NewBrowserPolicy(fixtures.BrowserConfig(), "/api/v1/auth", fixtures.SessionNow, strings.NewReader(strings.Repeat("n", 32768)))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func bootstrapProof(t *testing.T, p *authhandler.BrowserPolicy, refresh string) (string, *http.Response) {
	t.Helper()
	app, _ := testutils.NewHTTPApp(fiber.Config{DisableDefaultDate: true})
	app.Get("/csrf", func(c fiber.Ctx) error {
		result, err := p.Bootstrap(c)
		if err != nil {
			return err
		}
		return c.JSON(result)
	})
	r := httptest.NewRequest("GET", "/csrf", nil)
	r.Header.Set("Origin", fixtures.BrowserOrigin)
	r.AddCookie(&http.Cookie{Name: authhandler.RefreshCookie, Value: refresh})
	res, err := app.Test(r)
	body := testutils.ResponseBody(t, res, err)
	if res.StatusCode != 200 {
		t.Fatalf("bootstrap %d: %s", res.StatusCode, body)
	}
	var data domain.CSRFResponse
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		t.Fatal(err)
	}
	return data.CSRFToken, res
}

func browserCSRF(t *testing.T) {
	p := browserPolicy(t)
	token, res := bootstrapProof(t, p, "opaque-refresh")
	if cookies := res.Cookies(); len(cookies) != 1 || cookies[0].Name != authhandler.CSRFCookie || cookies[0].HttpOnly || cookies[0].MaxAge != 0 || !cookies[0].Expires.IsZero() {
		t.Fatal("bootstrap changed refresh or invented expiry")
	}
	digest := sha256.Sum256([]byte("opaque-refresh"))
	if strings.Contains(token, "opaque-refresh") || strings.Contains(token, base64.RawURLEncoding.EncodeToString(digest[:])) {
		t.Fatal("token exposed credential binding")
	}
	other := browserPolicy(t)
	cfg := fixtures.BrowserConfig()
	cfg.CSRFSecret = base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat("x", 32)))
	wrong, _ := authhandler.NewBrowserPolicy(cfg, "/api/v1/auth", fixtures.SessionNow, strings.NewReader(strings.Repeat("n", 32)))
	for _, tc := range []struct {
		name, refresh, cookie, header    string
		policy                           *authhandler.BrowserPolicy
		status                           int
		duplicateCookie, duplicateHeader bool
	}{
		{name: "valid across instance", refresh: "opaque-refresh", cookie: token, header: token, policy: other, status: 200},
		{name: "rotated refresh", refresh: "rotated", cookie: token, header: token, policy: p, status: 403},
		{name: "wrong secret", refresh: "opaque-refresh", cookie: token, header: token, policy: wrong, status: 403},
		{name: "header mismatch", refresh: "opaque-refresh", cookie: token, header: token + "x", policy: p, status: 403},
		{name: "signed component tamper", refresh: "opaque-refresh", cookie: token[:len(token)-2] + "AA", header: token[:len(token)-2] + "AA", policy: p, status: 403},
		{name: "malformed", refresh: "opaque-refresh", cookie: "invalid", header: "invalid", policy: p, status: 403},
		{name: "missing cookie", refresh: "opaque-refresh", header: token, policy: p, status: 403},
		{name: "missing header", refresh: "opaque-refresh", cookie: token, policy: p, status: 403},
		{name: "duplicate cookie", refresh: "opaque-refresh", cookie: token, header: token, policy: p, status: 403, duplicateCookie: true},
		{name: "duplicate header", refresh: "opaque-refresh", cookie: token, header: token, policy: p, status: 403, duplicateHeader: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app, _ := testutils.NewHTTPApp(fiber.Config{DisableDefaultDate: true})
			calls := 0
			app.Post("/refresh", func(c fiber.Ctx) error {
				_, err := tc.policy.RefreshCredential(c)
				if err != nil {
					return err
				}
				calls++
				return c.SendStatus(200)
			})
			r := httptest.NewRequest("POST", "/refresh", nil)
			r.Header.Set("Origin", fixtures.BrowserOrigin)
			r.AddCookie(&http.Cookie{Name: authhandler.RefreshCookie, Value: tc.refresh})
			if tc.cookie != "" {
				r.AddCookie(&http.Cookie{Name: authhandler.CSRFCookie, Value: tc.cookie})
			}
			if tc.duplicateCookie {
				r.AddCookie(&http.Cookie{Name: authhandler.CSRFCookie, Value: tc.cookie})
			}
			if tc.header != "" {
				r.Header.Set(authhandler.CSRFHeader, tc.header)
			}
			if tc.duplicateHeader {
				r.Header.Add(authhandler.CSRFHeader, tc.header)
			}
			res, err := app.Test(r)
			testutils.ResponseBody(t, res, err)
			if res.StatusCode != tc.status || tc.status != 200 && calls != 0 || len(res.Cookies()) != 0 {
				t.Fatalf("status=%d calls=%d", res.StatusCode, calls)
			}
		})
	}
}

func browserCookies(t *testing.T) {
	p := browserPolicy(t)
	for _, mode := range []string{"valid", "subsecond", "expired access", "missing session", "missing refresh", "reversed", "unsafe cookie"} {
		t.Run(mode, func(t *testing.T) {
			result := fixtures.LoginResult()
			result.SessionExpiresAt = result.SessionExpiresAt.Add(999 * time.Millisecond)
			switch mode {
			case "subsecond":
				result.SessionExpiresAt = fixtures.SessionNow().Add(999 * time.Millisecond)
				result.AccessExpiresAt = result.SessionExpiresAt
			case "expired access":
				result.AccessExpiresAt = fixtures.SessionNow()
			case "missing session":
				result.SessionID = ""
			case "missing refresh":
				result.RefreshToken = ""
			case "reversed":
				result.AccessExpiresAt = result.SessionExpiresAt.Add(time.Second)
			case "unsafe cookie":
				result.RefreshToken = "bad;cookie"
			}
			app, _ := testutils.NewHTTPApp(fiber.Config{DisableDefaultDate: true})
			app.Post("/issue", func(c fiber.Ctx) error {
				nonce, err := p.PrepareNonce()
				if err != nil {
					return err
				}
				data, err := p.Issue(c, result, nonce)
				if err != nil {
					return err
				}
				return c.JSON(data)
			})
			res, err := app.Test(httptest.NewRequest("POST", "/issue", nil))
			body := testutils.ResponseBody(t, res, err)
			if mode != "valid" {
				if res.StatusCode != 500 || len(res.Cookies()) != 0 {
					t.Fatal("invalid result produced cookie effects")
				}
				return
			}
			if res.StatusCode != 200 || len(res.Cookies()) != 2 || strings.Contains(body, "private-refresh") {
				t.Fatal("invalid safe success")
			}
			for _, c := range res.Cookies() {
				if c.Path != "/api/v1/auth" || c.Domain != "" || !c.Secure || c.SameSite != http.SameSiteLaxMode || c.HttpOnly != (c.Name == authhandler.RefreshCookie) || c.MaxAge != 86400 {
					t.Fatalf("invalid cookie attributes: %s", c.Name)
				}
				if !c.Expires.IsZero() && c.Expires.After(result.SessionExpiresAt) {
					t.Fatal("cookie extends session")
				}
			}
		})
	}
	broken, err := authhandler.NewBrowserPolicy(fixtures.BrowserConfig(), "/api/v1/auth", fixtures.SessionNow, strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = broken.PrepareNonce(); err == nil {
		t.Fatal("entropy failure accepted")
	}
	app, _ := testutils.NewHTTPApp(fiber.Config{DisableDefaultDate: true})
	app.Post("/clear", func(c fiber.Ctx) error { p.Clear(c); return c.SendStatus(200) })
	res, err := app.Test(httptest.NewRequest("POST", "/clear", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if len(res.Cookies()) != 2 {
		t.Fatal("expected both deletions")
	}
	for _, c := range res.Cookies() {
		if c.Value != "" || c.MaxAge >= 0 || c.Domain != "" || c.Path != "/api/v1/auth" {
			t.Fatal("invalid deletion")
		}
	}
}

func browserRequestPolicy(t *testing.T) {
	p := browserPolicy(t)
	for _, tc := range []struct {
		origin, content string
		status          int
	}{{fixtures.BrowserOrigin, "application/json; charset=utf-8", 200}, {"", "application/json", 403}, {"null", "application/json", 403}, {fixtures.BrowserOrigin + ".evil", "application/json", 403}, {fixtures.BrowserOrigin, "text/plain", 415}} {
		app, _ := testutils.NewHTTPApp(fiber.Config{DisableDefaultDate: true})
		app.Post("/login", func(c fiber.Ctx) error {
			if err := p.ValidateLogin(c); err != nil {
				return err
			}
			return c.SendStatus(200)
		})
		r := httptest.NewRequest("POST", "/login", nil)
		if tc.origin != "" {
			r.Header.Set("Origin", tc.origin)
		}
		r.Header.Set("Content-Type", tc.content)
		res, err := app.Test(r)
		testutils.ResponseBody(t, res, err)
		if res.StatusCode != tc.status {
			t.Fatal("origin/media type policy differs")
		}
	}
	for _, values := range [][]string{{""}, {"invalid"}, {"Bearer access", "Bearer second"}, {"Bearer access"}} {
		app, _ := testutils.NewHTTPApp(fiber.Config{DisableDefaultDate: true})
		app.Post("/logout", func(c fiber.Ctx) error {
			proof, err := p.LogoutCredential(c)
			if err != nil {
				return err
			}
			if proof.RefreshToken != "" {
				t.Fatal("fell back cookie")
			}
			return c.SendStatus(200)
		})
		r := httptest.NewRequest("POST", "/logout", nil)
		r.Header["Authorization"] = values
		r.AddCookie(&http.Cookie{Name: authhandler.RefreshCookie, Value: "refresh"})
		res, err := app.Test(r)
		testutils.ResponseBody(t, res, err)
		want := 401
		if len(values) == 1 && values[0] == "Bearer access" {
			want = 200
		}
		if res.StatusCode != want {
			t.Fatalf("bearer selection status %d want %d", res.StatusCode, want)
		}
	}
}
