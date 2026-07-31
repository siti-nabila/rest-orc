package fixtures

import (
	"os"
	"testing"
)

var configurationEnvironmentKeys = []string{
	"APP_NAME",
	"APP_ENV",
	"APP_PORT",
	"SERVER_READ_TIMEOUT",
	"SERVER_WRITE_TIMEOUT",
	"SERVER_IDLE_TIMEOUT",
	"SERVER_SHUTDOWN_TIMEOUT",
	"AUTH_GRPC_TARGET",
	"AUTH_GRPC_REQUEST_TIMEOUT",
	"AUTH_GRPC_MAX_RECEIVE_MESSAGE_BYTES",
	"AUTH_GRPC_MAX_SEND_MESSAGE_BYTES",
	"AUTH_GRPC_TLS_ENABLED",
	"AUTH_GRPC_TLS_SERVER_NAME",
	"AUTH_GRPC_TLS_CA_FILE",
	"AUTH_GRPC_KEEPALIVE_ENABLED",
	"AUTH_GRPC_KEEPALIVE_TIME",
	"AUTH_GRPC_KEEPALIVE_TIMEOUT",
	"BACKEND_HTTP_BASE_URL",
	"BACKEND_HTTP_REQUEST_TIMEOUT",
	"BACKEND_HTTP_TLS_ENABLED",
	"BACKEND_HTTP_TLS_SERVER_NAME",
	"BACKEND_HTTP_TLS_CA_FILE",
	"BACKEND_HTTP_KEEPALIVE_ENABLED",
	"BACKEND_HTTP_MAX_IDLE_CONNECTIONS",
	"BACKEND_HTTP_MAX_IDLE_CONNECTIONS_PER_HOST",
	"BACKEND_HTTP_MAX_CONNECTIONS_PER_HOST",
	"BACKEND_HTTP_IDLE_CONNECTION_TIMEOUT",
	"BACKEND_HTTP_RESPONSE_HEADER_TIMEOUT",
}

func ClearConfigurationEnvironment(t *testing.T) {
	t.Helper()

	for _, key := range configurationEnvironmentKeys {
		previous, existed := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset environment variable %s: %v", key, err)
		}

		t.Cleanup(func() {
			if existed {
				if err := os.Setenv(key, previous); err != nil {
					t.Errorf("restore environment variable %s: %v", key, err)
				}
				return
			}
			if err := os.Unsetenv(key); err != nil {
				t.Errorf("clear environment variable %s: %v", key, err)
			}
		})
	}
}
