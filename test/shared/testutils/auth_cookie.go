package testutils

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func AssertRefreshCookie(t *testing.T, response *http.Response, cleared bool) {
	t.Helper()
	headers := response.Header.Values("Set-Cookie")
	if !cleared {
		if len(headers) != 0 {
			t.Fatal("response unexpectedly changed browser cookies")
		}
		return
	}
	cookies := response.Cookies()
	if len(headers) != 2 || len(cookies) != 2 {
		t.Fatal("expected refresh and CSRF cookie deletions")
	}
	seen := map[string]bool{}
	for _, cookie := range cookies {
		if (cookie.Name != "__Secure-refresh" && cookie.Name != "__Secure-csrf") || seen[cookie.Name] || cookie.Value != "" || cookie.Path != "/api/v1/auth" || cookie.Domain != "" || !cookie.Secure || cookie.HttpOnly != (cookie.Name == "__Secure-refresh") || cookie.SameSite != http.SameSiteLaxMode {
			t.Fatal("refresh-cookie deletion attributes differ from the browser contract")
		}
		seen[cookie.Name] = true
		// Max-Age=0 removes immediately and takes precedence over Expires. Fiber
		// emits only Max-Age in that case; either deletion form satisfies the contract.
		pastExpiry := !cookie.Expires.IsZero() && cookie.Expires.Before(time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC))
		if cookie.MaxAge >= 0 && !pastExpiry {
			t.Fatalf("refresh-cookie deletion expiry=%s max-age=%d", cookie.Expires, cookie.MaxAge)
		}
	}
	for _, header := range headers {
		if strings.Contains(strings.ToLower(header), "domain=") {
			t.Fatal("refresh cookie must remain host-only")
		}
	}
}
