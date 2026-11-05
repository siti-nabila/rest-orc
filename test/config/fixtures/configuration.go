package fixtures

import (
	"bytes"
	"encoding/base64"
	"testing"
)

var validEnvironment = map[string]string{
	"AUTH_BROWSER_ALLOWED_ORIGINS":        "https://app.example.test",
	"AUTH_BROWSER_CSRF_SECRET":            base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{19}, 32)),
	"APP_NAME":                            "rest-orc",
	"APP_ENV":                             "test",
	"APP_PORT":                            "8080",
	"SERVER_READ_TIMEOUT":                 "10s",
	"SERVER_WRITE_TIMEOUT":                "10s",
	"SERVER_IDLE_TIMEOUT":                 "60s",
	"SERVER_SHUTDOWN_TIMEOUT":             "10s",
	"AUTH_GRPC_TARGET":                    "auth-proxy:50051",
	"AUTH_GRPC_REQUEST_TIMEOUT":           "5s",
	"AUTH_GRPC_MAX_RECEIVE_MESSAGE_BYTES": "4194304",
	"AUTH_GRPC_MAX_SEND_MESSAGE_BYTES":    "4194304",
	"AUTH_GRPC_TLS_ENABLED":               "true",
	"AUTH_GRPC_SERVICE_TOKEN":             base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{17}, 32)),
	"AUTH_GRPC_TLS_SERVER_NAME":           "",
	"AUTH_GRPC_TLS_CA_FILE":               "",
	"AUTH_GRPC_KEEPALIVE_ENABLED":         "false",
	"AUTH_GRPC_KEEPALIVE_TIME":            "5m",
	"AUTH_GRPC_KEEPALIVE_TIMEOUT":         "20s",
}

func SetValidEnvironment(t *testing.T) {
	t.Helper()
	ClearConfigurationEnvironment(t)
	for key, value := range validEnvironment {
		t.Setenv(key, value)
	}
}
