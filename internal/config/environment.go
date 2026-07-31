package config

import (
	"strconv"
	"time"

	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

type lookupEnvironment func(string) (string, bool)

func applyEnvironment(cfg *Config, lookup lookupEnvironment) error {
	stringOverride(lookup, "APP_NAME", &cfg.App.Name)
	stringOverride(lookup, "APP_ENV", &cfg.App.Env)
	stringOverride(lookup, "AUTH_GRPC_TARGET", &cfg.Clients.AuthGRPC.Target)
	stringOverride(lookup, "AUTH_GRPC_TLS_SERVER_NAME", &cfg.Clients.AuthGRPC.TLS.ServerName)
	stringOverride(lookup, "AUTH_GRPC_TLS_CA_FILE", &cfg.Clients.AuthGRPC.TLS.CAFile)
	stringOverride(lookup, "BACKEND_HTTP_BASE_URL", &cfg.Clients.BackendHTTP.BaseURL)
	stringOverride(lookup, "BACKEND_HTTP_TLS_SERVER_NAME", &cfg.Clients.BackendHTTP.TLS.ServerName)
	stringOverride(lookup, "BACKEND_HTTP_TLS_CA_FILE", &cfg.Clients.BackendHTTP.TLS.CAFile)

	intOverrides := []struct {
		key    string
		target *int
	}{
		{"APP_PORT", &cfg.App.Port},
		{"AUTH_GRPC_MAX_RECEIVE_MESSAGE_BYTES", &cfg.Clients.AuthGRPC.MaxReceiveMessageBytes},
		{"AUTH_GRPC_MAX_SEND_MESSAGE_BYTES", &cfg.Clients.AuthGRPC.MaxSendMessageBytes},
		{"BACKEND_HTTP_MAX_IDLE_CONNECTIONS", &cfg.Clients.BackendHTTP.Keepalive.MaxIdleConnections},
		{"BACKEND_HTTP_MAX_IDLE_CONNECTIONS_PER_HOST", &cfg.Clients.BackendHTTP.Keepalive.MaxIdleConnectionsPerHost},
		{"BACKEND_HTTP_MAX_CONNECTIONS_PER_HOST", &cfg.Clients.BackendHTTP.Keepalive.MaxConnectionsPerHost},
	}
	for _, override := range intOverrides {
		if err := intOverride(lookup, override.key, override.target); err != nil {
			return err
		}
	}

	boolOverrides := []struct {
		key    string
		target *bool
	}{
		{"AUTH_GRPC_TLS_ENABLED", &cfg.Clients.AuthGRPC.TLS.Enabled},
		{"AUTH_GRPC_KEEPALIVE_ENABLED", &cfg.Clients.AuthGRPC.Keepalive.Enabled},
		{"BACKEND_HTTP_TLS_ENABLED", &cfg.Clients.BackendHTTP.TLS.Enabled},
		{"BACKEND_HTTP_KEEPALIVE_ENABLED", &cfg.Clients.BackendHTTP.Keepalive.Enabled},
	}
	for _, override := range boolOverrides {
		if err := boolOverride(lookup, override.key, override.target); err != nil {
			return err
		}
	}

	durationOverrides := []struct {
		key    string
		target *Duration
	}{
		{"SERVER_READ_TIMEOUT", &cfg.Server.ReadTimeout},
		{"SERVER_WRITE_TIMEOUT", &cfg.Server.WriteTimeout},
		{"SERVER_IDLE_TIMEOUT", &cfg.Server.IdleTimeout},
		{"SERVER_SHUTDOWN_TIMEOUT", &cfg.Server.ShutdownTimeout},
		{"AUTH_GRPC_REQUEST_TIMEOUT", &cfg.Clients.AuthGRPC.RequestTimeout},
		{"AUTH_GRPC_KEEPALIVE_TIME", &cfg.Clients.AuthGRPC.Keepalive.Time},
		{"AUTH_GRPC_KEEPALIVE_TIMEOUT", &cfg.Clients.AuthGRPC.Keepalive.Timeout},
		{"BACKEND_HTTP_REQUEST_TIMEOUT", &cfg.Clients.BackendHTTP.RequestTimeout},
		{"BACKEND_HTTP_IDLE_CONNECTION_TIMEOUT", &cfg.Clients.BackendHTTP.Keepalive.IdleConnectionTimeout},
		{"BACKEND_HTTP_RESPONSE_HEADER_TIMEOUT", &cfg.Clients.BackendHTTP.Keepalive.ResponseHeaderTimeout},
	}
	for _, override := range durationOverrides {
		if err := durationOverride(lookup, override.key, override.target); err != nil {
			return err
		}
	}

	return nil
}

func stringOverride(lookup lookupEnvironment, key string, target *string) {
	if value, exists := lookup(key); exists {
		*target = value
	}
}

func intOverride(lookup lookupEnvironment, key string, target *int) error {
	value, exists := lookup(key)
	if !exists {
		return nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return environmentError(key, value, err)
	}

	*target = parsed
	return nil
}

func boolOverride(lookup lookupEnvironment, key string, target *bool) error {
	value, exists := lookup(key)
	if !exists {
		return nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return environmentError(key, value, err)
	}

	*target = parsed
	return nil
}

func durationOverride(lookup lookupEnvironment, key string, target *Duration) error {
	value, exists := lookup(key)
	if !exists {
		return nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return environmentError(key, value, err)
	}

	*target = newDuration(parsed)
	return nil
}

func environmentError(key, value string, cause error) error {
	return dictionary.ParseEnvironmentVariable(key, value, cause)
}
