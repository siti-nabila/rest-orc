package config

import "os"

type Config struct {
	App     AppConfig
	Server  ServerConfig
	Clients ClientsConfig
	Browser BrowserConfig
}

type AppConfig struct {
	Name string
	Env  string
	Port int
}

type ServerConfig struct {
	ReadTimeout     Duration
	WriteTimeout    Duration
	IdleTimeout     Duration
	ShutdownTimeout Duration
}

type ClientsConfig struct {
	AuthGRPC GRPCClientConfig
	// BackendHTTP HTTPClientConfig
}

type GRPCClientConfig struct {
	Target                 string
	ServiceToken           string
	RequestTimeout         Duration
	MaxReceiveMessageBytes int
	MaxSendMessageBytes    int
	TLS                    TLSConfig
	Keepalive              GRPCKeepaliveConfig
}

type HTTPClientConfig struct {
	BaseURL        string
	RequestTimeout Duration
	TLS            TLSConfig
	Keepalive      HTTPKeepaliveConfig
}

type TLSConfig struct {
	Enabled    bool
	ServerName string
	CAFile     string
}

type GRPCKeepaliveConfig struct {
	Enabled bool
	Time    Duration
	Timeout Duration
}

type HTTPKeepaliveConfig struct {
	Enabled                   bool
	MaxIdleConnections        int
	MaxIdleConnectionsPerHost int
	MaxConnectionsPerHost     int
	IdleConnectionTimeout     Duration
	ResponseHeaderTimeout     Duration
}

func Load() (*Config, error) {
	var cfg Config
	if err := applyEnvironment(&cfg, os.LookupEnv); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}
