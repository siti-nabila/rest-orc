package test_scenarios

import (
	"errors"
	"strings"
	"testing"

	"github.com/siti-nabila/rest-orc/internal/app"
	"github.com/siti-nabila/rest-orc/internal/config"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/test/app/fixtures"
)

func invalidServiceConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*config.Config)
		field  string
		cause  error
	}{
		{"missing", func(c *config.Config) { c.Clients.AuthGRPC.ServiceToken = "" }, "clients.auth_grpc.service_token", dictionary.ErrValueRequired},
		{"invalid", func(c *config.Config) { c.Clients.AuthGRPC.ServiceToken = "secret-sentinel" }, "clients.auth_grpc.service_token", dictionary.ErrInvalidServiceToken},
		{"insecure", func(c *config.Config) { c.Clients.AuthGRPC.TLS.Enabled = false }, "clients.auth_grpc.tls.enabled", dictionary.ErrServiceCredentialsRequireTLS},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := fixtures.ValidConfig()
			// A failing CA read would replace the expected error if resource
			// initialization ran before configuration validation.
			cfg.Clients.AuthGRPC.TLS.CAFile = "/missing/service-credential-ca.pem"
			tc.change(cfg)
			application, err := app.New(cfg)
			value, ok := errors.AsType[*dictionary.ValidationError](err)
			if application != nil || !ok || value.Field != tc.field || !errors.Is(err, tc.cause) {
				t.Fatal("application did not reject invalid auth configuration before resource initialization")
			}
			if strings.Contains(err.Error(), "secret-sentinel") {
				t.Error("startup error exposed a service credential")
			}
		})
	}
}
