package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

const minimumGRPCKeepaliveTime = 5 * time.Minute

type ValidationError struct {
	Field   string
	Problem string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("invalid configuration field %s: %s", e.Field, e.Problem)
}

func (cfg *Config) Validate() error {
	if strings.TrimSpace(cfg.App.Name) == "" {
		return validationError("app.name", "must not be empty")
	}
	if strings.TrimSpace(cfg.App.Env) == "" {
		return validationError("app.env", "must not be empty")
	}
	if cfg.App.Port < 1 || cfg.App.Port > 65535 {
		return validationError("app.port", "must be between 1 and 65535")
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
			return validationError(item.field, "must be greater than zero")
		}
	}

	if err := cfg.Clients.AuthGRPC.validate("clients.auth_grpc"); err != nil {
		return err
	}
	return cfg.Clients.BackendHTTP.validate("clients.backend_http")
}

func (cfg GRPCClientConfig) validate(fieldPrefix string) error {
	if strings.TrimSpace(cfg.Target) == "" {
		return validationError(fieldPrefix+".target", "must not be empty")
	}
	if cfg.RequestTimeout.Duration <= 0 {
		return validationError(fieldPrefix+".request_timeout", "must be greater than zero")
	}
	if cfg.MaxReceiveMessageBytes <= 0 {
		return validationError(fieldPrefix+".max_receive_message_bytes", "must be greater than zero")
	}
	if cfg.MaxSendMessageBytes <= 0 {
		return validationError(fieldPrefix+".max_send_message_bytes", "must be greater than zero")
	}
	if cfg.Keepalive.Time.Duration <= 0 {
		return validationError(fieldPrefix+".keepalive.time", "must be greater than zero")
	}
	if cfg.Keepalive.Timeout.Duration <= 0 {
		return validationError(fieldPrefix+".keepalive.timeout", "must be greater than zero")
	}
	if cfg.Keepalive.Enabled && cfg.Keepalive.Time.Duration < minimumGRPCKeepaliveTime {
		return validationError(
			fieldPrefix+".keepalive.time",
			fmt.Sprintf("must be at least %s when enabled", minimumGRPCKeepaliveTime),
		)
	}
	return nil
}

func (cfg HTTPClientConfig) validate(fieldPrefix string) error {
	parsed, err := url.Parse(cfg.BaseURL)
	if err != nil || parsed.Host == "" {
		return validationError(fieldPrefix+".base_url", "must be an absolute URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return validationError(fieldPrefix+".base_url", "scheme must be http or https")
	}
	if cfg.TLS.Enabled && parsed.Scheme != "https" {
		return validationError(fieldPrefix+".tls.enabled", "requires an https base_url")
	}
	if !cfg.TLS.Enabled && parsed.Scheme == "https" {
		return validationError(fieldPrefix+".tls.enabled", "must be true for an https base_url")
	}
	if cfg.RequestTimeout.Duration <= 0 {
		return validationError(fieldPrefix+".request_timeout", "must be greater than zero")
	}

	keepalive := cfg.Keepalive
	if keepalive.MaxIdleConnections <= 0 {
		return validationError(fieldPrefix+".keepalive.max_idle_connections", "must be greater than zero")
	}
	if keepalive.MaxIdleConnectionsPerHost <= 0 {
		return validationError(fieldPrefix+".keepalive.max_idle_connections_per_host", "must be greater than zero")
	}
	if keepalive.MaxConnectionsPerHost <= 0 {
		return validationError(fieldPrefix+".keepalive.max_connections_per_host", "must be greater than zero")
	}
	if keepalive.IdleConnectionTimeout.Duration <= 0 {
		return validationError(fieldPrefix+".keepalive.idle_connection_timeout", "must be greater than zero")
	}
	if keepalive.ResponseHeaderTimeout.Duration <= 0 {
		return validationError(fieldPrefix+".keepalive.response_header_timeout", "must be greater than zero")
	}
	return nil
}

func validationError(field, problem string) error {
	return &ValidationError{Field: field, Problem: problem}
}
