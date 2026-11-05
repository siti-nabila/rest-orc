package test_scenarios

import (
	"bytes"
	"encoding/base64"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/siti-nabila/rest-orc/internal/config"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/test/config/fixtures"
)

func serviceCredentials(t *testing.T) {
	encode := func(n int) string { return base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{17}, n)) }
	for _, tc := range []struct {
		name, token string
		missing     bool
		problem     error
	}{
		{"missing", "", true, dictionary.ErrValueRequired},
		{"empty", "", false, dictionary.ErrValueRequired},
		{"whitespace", " \t\n", false, dictionary.ErrInvalidServiceToken},
		{"31 bytes", encode(31), false, dictionary.ErrInvalidServiceToken},
		{"32 bytes", encode(32), false, nil},
		{"256 bytes", encode(256), false, nil},
		{"257 bytes", encode(257), false, dictionary.ErrInvalidServiceToken},
		{"padding", encode(32) + "=", false, dictionary.ErrInvalidServiceToken},
		{"alphabet", strings.Repeat("+", 43), false, dictionary.ErrInvalidServiceToken},
		{"noncanonical tail", strings.Repeat("A", 42) + "B", false, dictionary.ErrInvalidServiceToken},
		{"CRLF", encode(32)[:20] + "\r\n" + encode(32)[20:], false, dictionary.ErrInvalidServiceToken},
		{"no trimming", " " + encode(32), false, dictionary.ErrInvalidServiceToken},
		{"sentinel", "private-service-token-sentinel", false, dictionary.ErrInvalidServiceToken},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixtures.SetValidEnvironment(t)
			t.Setenv("AUTH_GRPC_SERVICE_TOKEN", tc.token)
			if tc.missing {
				if err := os.Unsetenv("AUTH_GRPC_SERVICE_TOKEN"); err != nil {
					t.Fatal(err)
				}
			}
			cfg, err := config.Load()
			if tc.problem == nil {
				if err != nil || cfg.Clients.AuthGRPC.ServiceToken != tc.token {
					t.Fatal("valid token was rejected or rewritten")
				}
				return
			}
			assertCredentialError(t, err, "clients.auth_grpc.service_token", tc.problem)
			if tc.token != "" && strings.Contains(err.Error(), tc.token) {
				t.Error("configuration error exposed a secret")
			}
		})
	}
	fixtures.SetValidEnvironment(t)
	t.Setenv("AUTH_GRPC_TLS_ENABLED", "false")
	_, err := config.Load()
	assertCredentialError(t, err, "clients.auth_grpc.tls.enabled", dictionary.ErrServiceCredentialsRequireTLS)
}

func assertCredentialError(t *testing.T, err error, field string, problem error) {
	t.Helper()
	value, ok := errors.AsType[*dictionary.ValidationError](err)
	if !ok || value.Field != field || !errors.Is(err, problem) {
		t.Fatal("credential validation did not preserve the field, type and catalog identity")
	}
}
