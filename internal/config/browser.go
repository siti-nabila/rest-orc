package config

import (
	"encoding/base64"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

type BrowserConfig struct {
	AllowedOrigins []string
	CSRFSecret     string
}

func (cfg BrowserConfig) Validate(serviceToken string) error {
	if len(cfg.AllowedOrigins) == 0 {
		return dictionary.NewValidationError("browser.allowed_origins", dictionary.ErrInvalidBrowserOrigins)
	}
	for _, origin := range cfg.AllowedOrigins {
		if !validBrowserOrigin(origin) {
			return dictionary.NewValidationError("browser.allowed_origins", dictionary.ErrInvalidBrowserOrigins)
		}
	}
	raw, err := base64.RawURLEncoding.Strict().DecodeString(cfg.CSRFSecret)
	if err != nil || len(raw) != 32 || base64.RawURLEncoding.EncodeToString(raw) != cfg.CSRFSecret || cfg.CSRFSecret == serviceToken {
		return dictionary.NewValidationError("browser.csrf_secret", dictionary.ErrInvalidCSRFSecret)
	}
	return nil
}

func validBrowserOrigin(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawFragment != "" || origin != "https://"+u.Host || strings.ContainsAny(u.Host, "*%\\ \t\r\n") {
		return false
	}
	if port := u.Port(); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return false
		}
	} else if strings.HasSuffix(u.Host, ":") {
		return false
	}
	host := u.Hostname()
	if net.ParseIP(host) != nil {
		return true
	}
	for _, label := range strings.Split(host, ".") {
		if label == "" || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, b := range []byte(label) {
			if !(b >= 'a' && b <= 'z' || b >= '0' && b <= '9' || b == '-') {
				return false
			}
		}
	}
	return true
}
