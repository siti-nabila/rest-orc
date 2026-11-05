package validation

import (
	"regexp"
	"strings"

	"github.com/siti-nabila/api-contracts/pkg/dictionary"
	authdictionary "github.com/siti-nabila/api-contracts/pkg/dictionary/auth"
	pkgdictionary "github.com/siti-nabila/rest-orc/pkg/dictionary"
)

const (
	authMinBytes = 6
	authMaxBytes = 50
)

// Preserve the existing public register/login pattern, including plus rejection.
var authEmailPattern = regexp.MustCompile(
	`^[A-Za-z0-9](?:[A-Za-z0-9._~\-]*[A-Za-z0-9])?@` +
		`[A-Za-z0-9](?:[A-Za-z0-9._~\-]*[A-Za-z0-9])?(?:\.[A-Za-z0-9](?:[A-Za-z0-9._~\-]*[A-Za-z0-9])?)+$`,
)

// NewPasswordCharacters applies only to newly chosen credentials, never legacy login/current passwords.
func NewPasswordCharacters(errs dictionary.FieldErrors, field, value string) {
	for i := 0; i < len(value); i++ {
		if value[i] < 0x21 || value[i] > 0x7E {
			errs.Add(field, pkgdictionary.ErrInvalidPasswordCharacters)
			return
		}
	}
}

// CredentialLength preserves byte bounds and required/min/max error order.
func CredentialLength(errs dictionary.FieldErrors, field, value string) {
	if value == "" {
		errs.Add(field, authdictionary.ErrRequired)
	}
	if len(value) < authMinBytes {
		errs.Add(field, authdictionary.ErrMinLength(authMinBytes))
	}
	if len(value) > authMaxBytes {
		errs.Add(field, authdictionary.ErrMaxLength(authMaxBytes))
	}
}

func asciiOnly(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] > 0x7F {
			return false
		}
	}
	return true
}

// Email validates a separate view; callers forward the original credential unchanged.
func Email(errs dictionary.FieldErrors, field, value string) {
	ascii := asciiOnly(value)
	email := strings.ToLower(strings.TrimSpace(value))
	CredentialLength(errs, field, email)
	if email != "" && (!ascii || !authEmailPattern.MatchString(email)) {
		errs.Add(field, authdictionary.ErrInvalidEmail)
	}
}
