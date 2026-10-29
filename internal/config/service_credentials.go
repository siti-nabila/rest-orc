package config

import (
	"encoding/base64"

	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

// ValidateServiceCredentials checks the backend's opaque credential format.
// Registration, expiry, and caller authority are still verified by the backend.
func (cfg GRPCClientConfig) ValidateServiceCredentials(prefix string) error {
	if cfg.ServiceToken == "" {
		return dictionary.NewValidationError(prefix+".service_token", dictionary.ErrValueRequired)
	}
	if !validServiceToken(cfg.ServiceToken) {
		return dictionary.NewValidationError(prefix+".service_token", dictionary.ErrInvalidServiceToken)
	}
	if !cfg.TLS.Enabled {
		return dictionary.NewValidationError(prefix+".tls.enabled", dictionary.ErrServiceCredentialsRequireTLS)
	}
	return nil
}

func validServiceToken(token string) bool {
	if len(token) < 43 || len(token) > 342 {
		return false
	}
	raw, err := base64.RawURLEncoding.Strict().DecodeString(token)
	return err == nil && len(raw) >= 32 && len(raw) <= 256 &&
		base64.RawURLEncoding.EncodeToString(raw) == token
}
