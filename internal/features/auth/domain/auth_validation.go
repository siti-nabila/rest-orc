package domain

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

// Keep the accepted pattern aligned with grpc-auth's domain/auth.go.
var authEmailPattern = regexp.MustCompile(
	`^[A-Za-z0-9](?:[A-Za-z0-9._~\-]*[A-Za-z0-9])?@` +
		`[A-Za-z0-9](?:[A-Za-z0-9._~\-]*[A-Za-z0-9])?(?:\.[A-Za-z0-9](?:[A-Za-z0-9._~\-]*[A-Za-z0-9])?)+$`,
)

func (request AuthRequest) ValidateRegister() error {
	errs := request.validationErrors()
	validateNewPasswordCharacters(errs, "password", request.Password)
	if errs.Empty() {
		return nil
	}
	return errs
}

func validateNewPasswordCharacters(errs dictionary.FieldErrors, field, value string) {
	for i := 0; i < len(value); i++ {
		if value[i] < 0x21 || value[i] > 0x7E {
			errs.Add(field, pkgdictionary.ErrInvalidPasswordCharacters)
			return
		}
	}
}

func (request AuthRequest) ValidateLogin() error {
	errs := request.validationErrors()
	if errs.Empty() {
		return nil
	}
	return errs
}

func (request AuthRequest) validationErrors() dictionary.FieldErrors {
	errs := dictionary.FieldErrors{}
	// Validate the original character set before normalizing a copy. The
	// request itself must reach the service unchanged.
	ascii := asciiOnly(request.Email)
	email := strings.ToLower(strings.TrimSpace(request.Email))
	validateAuthLength(errs, "email", email)
	if email != "" && (!ascii || !authEmailPattern.MatchString(email)) {
		errs.Add("email", authdictionary.ErrInvalidEmail)
	}
	validateAuthLength(errs, "password", request.Password)
	return errs
}

func validateAuthLength(errs dictionary.FieldErrors, field, value string) {
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
