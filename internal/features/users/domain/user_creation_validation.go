package domain

import (
	"github.com/siti-nabila/api-contracts/pkg/dictionary"
	pkgdictionary "github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/pkg/validation"
)

func ParseUserCreationKey(keys []string) (string, error) {
	if len(keys) != 1 {
		return "", dictionary.FieldErrors{"idempotency_key": {pkgdictionary.ErrInvalidSessionIdempotencyKey}}
	}
	if err := ValidateSessionIdempotencyKey(keys[0]); err != nil {
		return "", err
	}
	return keys[0], nil
}
func (request CreateUserRequest) NormalizeAndValidate() (CreateUserRequest, error) {
	if err := ValidateSessionIdempotencyKey(request.IdempotencyKey); err != nil {
		return CreateUserRequest{}, err
	}
	errs := dictionary.FieldErrors{}
	body := request.Body
	// Required nil inputs validate as empty; no field defaults are taken from the actor.
	email, password := creationValue(body.Email), creationValue(body.Password)
	validation.Email(errs, "email", email)
	validation.CredentialLength(errs, "password", password)
	validation.NewPasswordCharacters(errs, "password", password)
	name, phone, country := creationValue(body.Name), creationValue(body.Phone), creationValue(body.CountryCode)
	profile, err := (UpdateProfileRequest{Name: &name, Phone: &phone, CountryCode: &country, Address: body.Address}).NormalizeAndValidate()
	if fields, ok := err.(dictionary.FieldErrors); ok {
		for field, list := range fields {
			errs[field] = append(errs[field], list...)
		}
	}
	status := "PENDING"
	if body.Status != nil && *body.Status != status {
		errs.Add("status", pkgdictionary.ErrInvalidInitialUserStatus)
	}
	body = CreateUserBody{Email: &email, Password: &password, Name: profile.Name, Phone: profile.Phone, CountryCode: profile.CountryCode, Address: profile.Address, Status: &status}
	if !errs.Empty() {
		return CreateUserRequest{}, errs
	}
	return CreateUserRequest{Body: body, IdempotencyKey: request.IdempotencyKey}, nil
}
func creationValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
