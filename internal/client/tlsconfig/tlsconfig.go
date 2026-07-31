package tlsconfig

import (
	"crypto/tls"
	"crypto/x509"
	"os"

	"github.com/siti-nabila/rest-orc/internal/config"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
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
		return nil, dictionary.ReadTLSCAFile(cfg.CAFile, err)
	}

	roots, err := x509.SystemCertPool()
	if err != nil {
		return nil, dictionary.LoadSystemCertificatePool(err)
	}
	if roots == nil {
		roots = x509.NewCertPool()
	}
	if ok := roots.AppendCertsFromPEM(certificates); !ok {
		return nil, dictionary.TLSCACertificatesRequired(cfg.CAFile)
	}

	tlsConfig.RootCAs = roots
	return tlsConfig, nil
}
