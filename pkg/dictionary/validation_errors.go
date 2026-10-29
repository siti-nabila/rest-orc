package dictionary

import (
	_ "embed"

	contractdictionary "github.com/siti-nabila/api-contracts/pkg/dictionary"
)

//go:embed validation_errors.yaml
var validationCatalog []byte

// Preserve the legacy namespace so moving the catalog does not change error identity.
var validationRegistry = contractdictionary.MustLoadYAML("users", validationCatalog)

var ErrInvalidPasswordCharacters = validationRegistry.MustNew("invalid_password_characters")

var (
	ErrPasswordMustDiffer       = validationRegistry.MustNew("password_must_differ")
	ErrInvalidProfileCharacters = validationRegistry.MustNew("invalid_profile_characters")
	ErrInvalidCountryCode       = validationRegistry.MustNew("invalid_country_code")
	ErrInvalidPhone             = validationRegistry.MustNew("invalid_phone")
)

var (
	ErrInvalidSessionID          = validationRegistry.MustNew("invalid_session_id")
	ErrInvalidSessionPagination  = validationRegistry.MustNew("invalid_session_pagination")
	ErrUnsupportedSessionControl = validationRegistry.MustNew("unsupported_session_control")
)
