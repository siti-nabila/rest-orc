package fixtures

import (
	"time"

	"github.com/siti-nabila/rest-orc/internal/config"
)

func ValidConfig(baseURL string, keepaliveEnabled bool) config.HTTPClientConfig {
	return config.HTTPClientConfig{
		BaseURL:        baseURL,
		RequestTimeout: config.Duration{Duration: 5 * time.Second},
		Keepalive: config.HTTPKeepaliveConfig{
			Enabled:                   keepaliveEnabled,
			MaxIdleConnections:        100,
			MaxIdleConnectionsPerHost: 20,
			MaxConnectionsPerHost:     100,
			IdleConnectionTimeout:     config.Duration{Duration: 90 * time.Second},
			ResponseHeaderTimeout:     config.Duration{Duration: 5 * time.Second},
		},
	}
}

func TLSMismatchConfig() config.HTTPClientConfig {
	return ValidConfig("https://service-proxy:8443", true)
}
