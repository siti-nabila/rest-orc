package domain

import (
	contractdictionary "github.com/siti-nabila/api-contracts/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

// ParseUserProfileHeaders distinguishes an absent optional key from an empty one.
func ParseUserProfileHeaders(keys, versions []string) (*string, uint64, error) {
	if len(versions) != 1 {
		return nil, 0, contractdictionary.FieldErrors{"version": {dictionary.ErrInvalidSessionVersion}}
	}
	version, err := ParseAdminSessionVersion(versions[0])
	if err != nil {
		return nil, 0, err
	}
	if len(keys) == 0 {
		return nil, version, nil
	}
	if len(keys) != 1 {
		return nil, 0, contractdictionary.FieldErrors{"idempotency_key": {dictionary.ErrInvalidSessionIdempotencyKey}}
	}
	if err := ValidateSessionIdempotencyKey(keys[0]); err != nil {
		return nil, 0, err
	}
	key := keys[0]
	return &key, version, nil
}

func (request UpdateUserProfileRequest) ValidatePreconditions() error {
	if err := ValidateUserDetailID(request.UserID); err != nil {
		return err
	}
	if request.Version == 0 || request.Version > 9223372036854775807 {
		return contractdictionary.FieldErrors{"version": {dictionary.ErrInvalidSessionVersion}}
	}
	if request.IdempotencyKey != nil {
		return ValidateSessionIdempotencyKey(*request.IdempotencyKey)
	}
	return nil
}

func (request UpdateUserProfileRequest) NormalizeAndValidate() (UpdateUserProfileRequest, error) {
	if err := request.ValidatePreconditions(); err != nil {
		return UpdateUserProfileRequest{}, err
	}
	profile, err := request.Profile.NormalizeAndValidate()
	if err != nil {
		return UpdateUserProfileRequest{}, err
	}
	result := request
	result.Profile = profile
	if request.IdempotencyKey != nil {
		key := *request.IdempotencyKey
		result.IdempotencyKey = &key
	}
	return result, nil
}
