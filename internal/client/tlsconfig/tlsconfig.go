package tlsconfig

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/siti-nabila/rest-orc/internal/config"
)

func New(cfg config.TLSConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: cfg.ServerName,
	}
	if cfg.CAFile == "" {
		return tlsConfig, nil
	}

	certificates, err := os.ReadFile(cfg.CAFile)
	if err != nil {
		return nil, fmt.Errorf("read TLS CA file %q: %w", cfg.CAFile, err)
	}

	roots, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("load system certificate pool: %w", err)
	}
	if roots == nil {
		roots = x509.NewCertPool()
	}
	if ok := roots.AppendCertsFromPEM(certificates); !ok {
		return nil, fmt.Errorf("parse TLS CA file %q: no certificates found", cfg.CAFile)
	}

	tlsConfig.RootCAs = roots
	return tlsConfig, nil
}
