package fixtures

import (
	"bytes"
	"crypto/tls"
	"embed"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

// All PEMs are deterministic test-only fixtures for synctest's year 2000.
// Never use them as deployment certificates or private keys.
//
//go:embed *.pem
var certificates embed.FS

func Token(seed byte) string {
	return base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{seed}, 32))
}

func CAFile(t *testing.T, name string) string {
	t.Helper()
	data, err := certificates.ReadFile(name + ".pem")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func ServerCertificate(t *testing.T, name string) tls.Certificate {
	t.Helper()
	cert, err := certificates.ReadFile(name + ".pem")
	if err != nil {
		t.Fatal(err)
	}
	key, err := certificates.ReadFile("server_key.pem")
	if err != nil {
		t.Fatal(err)
	}
	pair, err := tls.X509KeyPair(cert, key)
	if err != nil {
		t.Fatal("invalid test certificate pair")
	}
	return pair
}
