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

var (
	ErrInvalidSessionUserID         = validationRegistry.MustNew("invalid_session_user_id")
	ErrInvalidSessionIdempotencyKey = validationRegistry.MustNew("invalid_session_idempotency_key")
	ErrInvalidSessionVersion        = validationRegistry.MustNew("invalid_session_version")
)

var ErrInvalidUserID = validationRegistry.MustNew("invalid_user_id")

var ErrUnsupportedAccessControl = validationRegistry.MustNew("unsupported_access_control")

var (
	ErrInvalidUserStatus             = validationRegistry.MustNew("invalid_user_status")
	ErrInvalidStatusReasonCharacters = validationRegistry.MustNew("invalid_status_reason_characters")
)

var ErrInvalidInitialUserStatus = validationRegistry.MustNew("invalid_initial_user_status")

var (
	ErrInvalidRoleID          = validationRegistry.MustNew("invalid_role_id")
	ErrInvalidRolePagination  = validationRegistry.MustNew("invalid_role_pagination")
	ErrRepeatedRoleQuery      = validationRegistry.MustNew("repeated_role_query")
	ErrInvalidRoleApplication = validationRegistry.MustNew("invalid_role_application")
	ErrInvalidRoleStatuses    = validationRegistry.MustNew("invalid_role_statuses")
	ErrInvalidRoleKeyword     = validationRegistry.MustNew("invalid_role_keyword")
	ErrInvalidRoleCursor      = validationRegistry.MustNew("invalid_role_cursor")
	ErrUnsupportedRoleControl = validationRegistry.MustNew("unsupported_role_control")
)

var (
	ErrInvalidAssignmentRoleIDs     = validationRegistry.MustNew("invalid_assignment_role_ids")
	ErrInvalidAssignmentStatuses    = validationRegistry.MustNew("invalid_assignment_statuses")
	ErrUnsupportedAssignmentControl = validationRegistry.MustNew("unsupported_assignment_control")
)

var (
	ErrInvalidApplicationPagination  = validationRegistry.MustNew("invalid_application_pagination")
	ErrRepeatedApplicationQuery      = validationRegistry.MustNew("repeated_application_query")
	ErrInvalidApplicationStatuses    = validationRegistry.MustNew("invalid_application_statuses")
	ErrInvalidApplicationKeyword     = validationRegistry.MustNew("invalid_application_keyword")
	ErrInvalidApplicationCursor      = validationRegistry.MustNew("invalid_application_cursor")
	ErrUnsupportedApplicationControl = validationRegistry.MustNew("unsupported_application_control")
)
