package config

import (
	"net/url"
	"strings"
	"time"

	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

const minimumGRPCKeepaliveTime = 5 * time.Minute

func (cfg *Config) Validate() error {
	if strings.TrimSpace(cfg.App.Name) == "" {
		return dictionary.NewValidationError("app.name", dictionary.ErrValueRequired)
	}
	if strings.TrimSpace(cfg.App.Env) == "" {
		return dictionary.NewValidationError("app.env", dictionary.ErrValueRequired)
	}
	if cfg.App.Port < 1 || cfg.App.Port > 65535 {
		return dictionary.NewValidationError("app.port", dictionary.ErrPortOutOfRange)
	}

	serverDurations := []struct {
		field string
		value Duration
	}{
		{"server.read_timeout", cfg.Server.ReadTimeout},
		{"server.write_timeout", cfg.Server.WriteTimeout},
		{"server.idle_timeout", cfg.Server.IdleTimeout},
		{"server.shutdown_timeout", cfg.Server.ShutdownTimeout},
	}
	for _, item := range serverDurations {
		if item.value.Duration <= 0 {
			return dictionary.NewValidationError(item.field, dictionary.ErrValueNotPositive)
		}
	}

	if err := cfg.Clients.AuthGRPC.validate("clients.auth_grpc"); err != nil {
		return err
	}
	return cfg.Clients.BackendHTTP.validate("clients.backend_http")
}

func (cfg GRPCClientConfig) validate(fieldPrefix string) error {
	if strings.TrimSpace(cfg.Target) == "" {
		return dictionary.NewValidationError(fieldPrefix+".target", dictionary.ErrValueRequired)
	}
	if cfg.RequestTimeout.Duration <= 0 {
		return dictionary.NewValidationError(fieldPrefix+".request_timeout", dictionary.ErrValueNotPositive)
	}
	if cfg.MaxReceiveMessageBytes <= 0 {
		return dictionary.NewValidationError(fieldPrefix+".max_receive_message_bytes", dictionary.ErrValueNotPositive)
	}
	if cfg.MaxSendMessageBytes <= 0 {
		return dictionary.NewValidationError(fieldPrefix+".max_send_message_bytes", dictionary.ErrValueNotPositive)
	}
	if cfg.Keepalive.Time.Duration <= 0 {
		return dictionary.NewValidationError(fieldPrefix+".keepalive.time", dictionary.ErrValueNotPositive)
	}
	if cfg.Keepalive.Timeout.Duration <= 0 {
		return dictionary.NewValidationError(fieldPrefix+".keepalive.timeout", dictionary.ErrValueNotPositive)
	}
	if cfg.Keepalive.Enabled && cfg.Keepalive.Time.Duration < minimumGRPCKeepaliveTime {
		return dictionary.NewValidationError(
			fieldPrefix+".keepalive.time",
			dictionary.MinimumDurationWhenEnabled(minimumGRPCKeepaliveTime),
		)
	}
	return nil
}

func (cfg HTTPClientConfig) validate(fieldPrefix string) error {
	parsed, err := url.Parse(cfg.BaseURL)
	if err != nil || parsed.Host == "" {
		return dictionary.NewValidationError(fieldPrefix+".base_url", dictionary.ErrAbsoluteURLRequired)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return dictionary.NewValidationError(fieldPrefix+".base_url", dictionary.ErrUnsupportedURLScheme)
	}
	if cfg.TLS.Enabled && parsed.Scheme != "https" {
		return dictionary.NewValidationError(fieldPrefix+".tls.enabled", dictionary.ErrHTTPSRequired)
	}
	if !cfg.TLS.Enabled && parsed.Scheme == "https" {
		return dictionary.NewValidationError(fieldPrefix+".tls.enabled", dictionary.ErrTLSEnabledForHTTPS)
	}
	if cfg.RequestTimeout.Duration <= 0 {
		return dictionary.NewValidationError(fieldPrefix+".request_timeout", dictionary.ErrValueNotPositive)
	}

	keepalive := cfg.Keepalive
	if keepalive.MaxIdleConnections <= 0 {
		return dictionary.NewValidationError(fieldPrefix+".keepalive.max_idle_connections", dictionary.ErrValueNotPositive)
	}
	if keepalive.MaxIdleConnectionsPerHost <= 0 {
		return dictionary.NewValidationError(fieldPrefix+".keepalive.max_idle_connections_per_host", dictionary.ErrValueNotPositive)
	}
	if keepalive.MaxConnectionsPerHost <= 0 {
		return dictionary.NewValidationError(fieldPrefix+".keepalive.max_connections_per_host", dictionary.ErrValueNotPositive)
	}
	if keepalive.IdleConnectionTimeout.Duration <= 0 {
		return dictionary.NewValidationError(fieldPrefix+".keepalive.idle_connection_timeout", dictionary.ErrValueNotPositive)
	}
	if keepalive.ResponseHeaderTimeout.Duration <= 0 {
		return dictionary.NewValidationError(fieldPrefix+".keepalive.response_header_timeout", dictionary.ErrValueNotPositive)
	}
	return nil
}
