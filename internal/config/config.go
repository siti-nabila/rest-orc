package config

import (
	"os"

	"github.com/goccy/go-yaml"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

type Config struct {
	App     AppConfig     `yaml:"app"`
	Server  ServerConfig  `yaml:"server"`
	Clients ClientsConfig `yaml:"clients"`
}

type AppConfig struct {
	Name string `yaml:"name"`
	Env  string `yaml:"env"`
	Port int    `yaml:"port"`
}

type ServerConfig struct {
	ReadTimeout     Duration `yaml:"read_timeout"`
	WriteTimeout    Duration `yaml:"write_timeout"`
	IdleTimeout     Duration `yaml:"idle_timeout"`
	ShutdownTimeout Duration `yaml:"shutdown_timeout"`
}

type ClientsConfig struct {
	AuthGRPC    GRPCClientConfig `yaml:"auth_grpc"`
	BackendHTTP HTTPClientConfig `yaml:"backend_http"`
}

type GRPCClientConfig struct {
	Target                 string              `yaml:"target"`
	RequestTimeout         Duration            `yaml:"request_timeout"`
	MaxReceiveMessageBytes int                 `yaml:"max_receive_message_bytes"`
	MaxSendMessageBytes    int                 `yaml:"max_send_message_bytes"`
	TLS                    TLSConfig           `yaml:"tls"`
	Keepalive              GRPCKeepaliveConfig `yaml:"keepalive"`
}

type HTTPClientConfig struct {
	BaseURL        string              `yaml:"base_url"`
	RequestTimeout Duration            `yaml:"request_timeout"`
	TLS            TLSConfig           `yaml:"tls"`
	Keepalive      HTTPKeepaliveConfig `yaml:"keepalive"`
}

type TLSConfig struct {
	Enabled    bool   `yaml:"enabled"`
	ServerName string `yaml:"server_name"`
	CAFile     string `yaml:"ca_file"`
}

type GRPCKeepaliveConfig struct {
	Enabled bool     `yaml:"enabled"`
	Time    Duration `yaml:"time"`
	Timeout Duration `yaml:"timeout"`
}

type HTTPKeepaliveConfig struct {
	Enabled                   bool     `yaml:"enabled"`
	MaxIdleConnections        int      `yaml:"max_idle_connections"`
	MaxIdleConnectionsPerHost int      `yaml:"max_idle_connections_per_host"`
	MaxConnectionsPerHost     int      `yaml:"max_connections_per_host"`
	IdleConnectionTimeout     Duration `yaml:"idle_connection_timeout"`
	ResponseHeaderTimeout     Duration `yaml:"response_header_timeout"`
}

func Load(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, dictionary.OpenConfiguration(path, err)
	}

	var cfg Config
	decodeErr := yaml.NewDecoder(file, yaml.Strict()).Decode(&cfg)
	closeErr := file.Close()
	if decodeErr != nil {
		return nil, dictionary.DecodeConfiguration(path, decodeErr)
	}
	if closeErr != nil {
		return nil, dictionary.CloseConfiguration(path, closeErr)
	}

	if err := applyEnvironment(&cfg, os.LookupEnv); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}
