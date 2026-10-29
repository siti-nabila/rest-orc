package fixtures

import (
	"time"

	"github.com/siti-nabila/rest-orc/internal/config"
)

func ValidConfig() config.GRPCClientConfig {
	return config.GRPCClientConfig{
		Target:                 "passthrough:///auth-proxy:50051",
		RequestTimeout:         config.Duration{Duration: 5 * time.Second},
		MaxReceiveMessageBytes: 4 * 1024 * 1024,
		MaxSendMessageBytes:    4 * 1024 * 1024,
		Keepalive: config.GRPCKeepaliveConfig{
			Time:    config.Duration{Duration: 5 * time.Minute},
			Timeout: config.Duration{Duration: 20 * time.Second},
		},
	}
}

func ConfigWithKeepalive(enabled bool) config.GRPCClientConfig {
	cfg := ValidConfig()
	cfg.Keepalive.Enabled = enabled
	return cfg
}

func ConfigWithTarget(target string) config.GRPCClientConfig {
	cfg := ValidConfig()
	cfg.Target = target
	return cfg
}

func ConfigWithUnreadableCA() config.GRPCClientConfig {
	cfg := ValidConfig()
	cfg.TLS.Enabled = true
	cfg.TLS.CAFile = "/path/that/does/not/exist/ca.pem"
	return cfg
}
