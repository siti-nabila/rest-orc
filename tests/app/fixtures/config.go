package fixtures

import (
	"time"

	"github.com/siti-nabila/rest-orc/internal/config"
)

func ValidConfig() *config.Config {
	return &config.Config{
		App: config.AppConfig{
			Name: "rest-orc",
			Env:  "test",
			Port: 8080,
		},
		Server: config.ServerConfig{
			ReadTimeout:     duration(10 * time.Second),
			WriteTimeout:    duration(10 * time.Second),
			IdleTimeout:     duration(60 * time.Second),
			ShutdownTimeout: duration(10 * time.Second),
		},
		Clients: config.ClientsConfig{
			AuthGRPC: config.GRPCClientConfig{
				Target:                 "passthrough:///auth-proxy:50051",
				RequestTimeout:         duration(5 * time.Second),
				MaxReceiveMessageBytes: 4 * 1024 * 1024,
				MaxSendMessageBytes:    4 * 1024 * 1024,
				Keepalive: config.GRPCKeepaliveConfig{
					Time:    duration(5 * time.Minute),
					Timeout: duration(20 * time.Second),
				},
			},
			BackendHTTP: config.HTTPClientConfig{
				BaseURL:        "http://backend-proxy:8080",
				RequestTimeout: duration(10 * time.Second),
				Keepalive: config.HTTPKeepaliveConfig{
					Enabled:                   true,
					MaxIdleConnections:        100,
					MaxIdleConnectionsPerHost: 20,
					MaxConnectionsPerHost:     100,
					IdleConnectionTimeout:     duration(90 * time.Second),
					ResponseHeaderTimeout:     duration(10 * time.Second),
				},
			},
		},
	}
}

func ConfigWithUnreadableBackendCA() *config.Config {
	cfg := ValidConfig()
	cfg.Clients.BackendHTTP.TLS.Enabled = true
	cfg.Clients.BackendHTTP.BaseURL = "https://backend-proxy:8443"
	cfg.Clients.BackendHTTP.TLS.CAFile = "/path/that/does/not/exist/ca.pem"
	return cfg
}

func duration(value time.Duration) config.Duration {
	return config.Duration{Duration: value}
}
