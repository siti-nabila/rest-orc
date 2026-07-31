package fixtures

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const ValidConfiguration = `app:
  name: rest-orc
  env: development
  port: 8080
server:
  read_timeout: 10s
  write_timeout: 10s
  idle_timeout: 60s
  shutdown_timeout: 10s
clients:
  auth_grpc:
    target: auth-proxy:50051
    request_timeout: 5s
    max_receive_message_bytes: 4194304
    max_send_message_bytes: 4194304
    tls:
      enabled: false
      server_name: ""
      ca_file: ""
    keepalive:
      enabled: false
      time: 5m
      timeout: 20s
  backend_http:
    base_url: http://backend-proxy:8080
    request_timeout: 10s
    tls:
      enabled: false
      server_name: ""
      ca_file: ""
    keepalive:
      enabled: true
      max_idle_connections: 100
      max_idle_connections_per_host: 20
      max_connections_per_host: 100
      idle_connection_timeout: 90s
      response_header_timeout: 10s
`

func WithUnknownField() string {
	return strings.Replace(
		ValidConfiguration,
		"  port: 8080",
		"  port: 8080\n  unknown: value",
		1,
	)
}

func WithMalformedDuration() string {
	return strings.Replace(
		ValidConfiguration,
		"read_timeout: 10s",
		"read_timeout: sometime",
		1,
	)
}

func WithEmptyGRPCTarget() string {
	return strings.Replace(
		ValidConfiguration,
		"target: auth-proxy:50051",
		`target: ""`,
		1,
	)
}

func WithAggressiveGRPCKeepalive() string {
	return strings.NewReplacer(
		"enabled: false\n      time: 5m",
		"enabled: true\n      time: 1m",
	).Replace(ValidConfiguration)
}

func WithEmptyAppName() string {
	return strings.Replace(
		ValidConfiguration,
		"  name: rest-orc",
		`  name: ""`,
		1,
	)
}

func Write(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "env.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config fixture: %v", err)
	}
	return path
}
