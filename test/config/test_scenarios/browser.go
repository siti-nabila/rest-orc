package test_scenarios

import (
	"strings"
	"testing"

	"github.com/siti-nabila/rest-orc/internal/app"
	"github.com/siti-nabila/rest-orc/internal/config"
	appfixtures "github.com/siti-nabila/rest-orc/test/app/fixtures"
	"github.com/siti-nabila/rest-orc/test/config/fixtures"
)

func browserConfiguration(t *testing.T) {
	for _, tc := range []struct{ key, value string }{
		{"AUTH_BROWSER_ALLOWED_ORIGINS", ""}, {"AUTH_BROWSER_ALLOWED_ORIGINS", "*"},
		{"AUTH_BROWSER_ALLOWED_ORIGINS", "null"}, {"AUTH_BROWSER_ALLOWED_ORIGINS", "http://app.test"},
		{"AUTH_BROWSER_ALLOWED_ORIGINS", "https://app.test/"}, {"AUTH_BROWSER_ALLOWED_ORIGINS", "https://u@app.test"},
		{"AUTH_BROWSER_ALLOWED_ORIGINS", "https://app.test?q=x"}, {"AUTH_BROWSER_ALLOWED_ORIGINS", "https://app.test#x"},
		{"AUTH_BROWSER_ALLOWED_ORIGINS", "https://*.test"}, {"AUTH_BROWSER_ALLOWED_ORIGINS", "https://app.test:99999"},
		{"AUTH_BROWSER_CSRF_SECRET", ""}, {"AUTH_BROWSER_CSRF_SECRET", "private-csrf-secret-sentinel"},
		{"AUTH_BROWSER_CSRF_SECRET", strings.Repeat("A", 43) + "="},
	} {
		t.Run(tc.key+tc.value, func(t *testing.T) {
			fixtures.SetValidEnvironment(t)
			t.Setenv(tc.key, tc.value)
			_, err := config.Load()
			if err == nil {
				t.Fatal("invalid browser configuration accepted")
			}
			if tc.key == "AUTH_BROWSER_CSRF_SECRET" && tc.value != "" && strings.Contains(err.Error(), tc.value) {
				t.Fatal("secret exposed")
			}
		})
	}
	cfg := appfixtures.ValidConfig()
	cfg.Browser.CSRFSecret = cfg.Clients.AuthGRPC.ServiceToken
	if app, err := app.New(cfg); err == nil || app != nil {
		t.Fatal("direct config bypassed browser validation")
	}
	fixtures.SetValidEnvironment(t)
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	original := cfg.Browser.CSRFSecret
	t.Setenv("AUTH_BROWSER_CSRF_SECRET", "changed")
	if cfg.Browser.CSRFSecret != original {
		t.Fatal("loaded config changed with environment")
	}
}
