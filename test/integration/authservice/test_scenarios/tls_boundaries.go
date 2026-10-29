package test_scenarios

import (
	"context"
	"crypto/tls"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/siti-nabila/rest-orc/internal/client/grpcclient"
	"github.com/siti-nabila/rest-orc/internal/config"
	"github.com/siti-nabila/rest-orc/test/integration/authservice/fixtures"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func tlsBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*testing.T, *config.GRPCClientConfig, *tls.Config) *tls.Config
		valid  bool
	}{
		{"target authority", func(_ *testing.T, _ *config.GRPCClientConfig, s *tls.Config) *tls.Config { return s }, true},
		{"server name override", func(_ *testing.T, c *config.GRPCClientConfig, s *tls.Config) *tls.Config {
			c.Target = "passthrough:///other.test"
			c.TLS.ServerName = "auth.test"
			return s
		}, true},
		{"SAN mismatch", func(_ *testing.T, c *config.GRPCClientConfig, s *tls.Config) *tls.Config {
			c.TLS.ServerName = "wrong.test"
			return s
		}, false},
		{"untrusted chain", func(t *testing.T, c *config.GRPCClientConfig, s *tls.Config) *tls.Config {
			c.TLS.CAFile = fixtures.CAFile(t, "untrusted_ca")
			return s
		}, false},
		{"system roots reject private test CA", func(_ *testing.T, c *config.GRPCClientConfig, s *tls.Config) *tls.Config {
			c.TLS.CAFile = ""
			return s
		}, false},
		{"expired certificate", func(t *testing.T, _ *config.GRPCClientConfig, s *tls.Config) *tls.Config {
			s.Certificates = []tls.Certificate{fixtures.ServerCertificate(t, "expired")}
			return s
		}, false},
		{"not yet valid certificate", func(t *testing.T, _ *config.GRPCClientConfig, s *tls.Config) *tls.Config {
			s.Certificates = []tls.Certificate{fixtures.ServerCertificate(t, "future")}
			return s
		}, false},
		{"TLS 1.2 only", func(_ *testing.T, _ *config.GRPCClientConfig, s *tls.Config) *tls.Config {
			s.MinVersion, s.MaxVersion = tls.VersionTLS12, tls.VersionTLS12
			return s
		}, false},
		{"plaintext peer", func(_ *testing.T, _ *config.GRPCClientConfig, _ *tls.Config) *tls.Config { return nil }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				cfg := secureConfig(t)
				serverTLS := tc.change(t, &cfg, defaultServerTLS(t))
				h := newTransport(t, cfg, serverTLS, nil)
				defer h.close(t)
				_, err := h.client.Me(context.Background())
				calls, _ := h.snapshot()
				if tc.valid {
					if err != nil || len(calls) != 1 || calls[0].tls != tls.VersionTLS13 {
						t.Error("trusted TLS 1.3 peer did not receive one RPC")
					}
				} else if err == nil || len(calls) != 0 {
					t.Error("unverified peer received a credential or business RPC")
				}
			})
		})
	}
	for _, name := range []string{"unreadable", "malformed"} {
		t.Run(name+" CA", func(t *testing.T) {
			cfg := secureConfig(t)
			cfg.TLS.CAFile = filepath.Join(t.TempDir(), "ca.pem")
			if name == "malformed" {
				if err := os.WriteFile(cfg.TLS.CAFile, []byte("not PEM"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			connection, err := grpcclient.New(cfg)
			if err == nil || connection != nil {
				t.Fatal("invalid configured CA was ignored")
			}
			if strings.Contains(err.Error(), cfg.ServiceToken) {
				t.Error("CA error exposed service credential")
			}
		})
	}
}

func conflictingCredentials(t *testing.T) {
	for _, tc := range []struct {
		name   string
		values []string
	}{
		{"single", []string{fixtures.Token(18)}},
		{"multiple", []string{fixtures.Token(18), fixtures.Token(19)}},
		{"same as configured", []string{fixtures.Token(17)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				cfg := secureConfig(t)
				h := newTransport(t, cfg, defaultServerTLS(t), nil)
				defer h.close(t)
				md := metadata.MD{"x-service-token": tc.values, "authorization": {"Bearer unchanged"}}
				before := md.Copy()
				ctx := metadata.NewOutgoingContext(context.Background(), md)
				_, err := h.client.Me(ctx)
				calls, _ := h.snapshot()
				if status.Code(err) != codes.Unauthenticated || len(calls) != 0 {
					t.Error("conflicting credentials were dispatched or did not fail authentication")
				}
				actual, _ := metadata.FromOutgoingContext(ctx)
				if !reflect.DeepEqual(actual, before) || !reflect.DeepEqual(md, before) {
					t.Error("provider mutated original metadata")
				}
				for _, secret := range append(append([]string{}, tc.values...), cfg.ServiceToken) {
					if err != nil && strings.Contains(err.Error(), secret) {
						t.Error("conflicting credential error exposed a secret")
					}
				}
			})
		})
	}
}
