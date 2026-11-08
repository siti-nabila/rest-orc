package domain

import (
	"github.com/siti-nabila/api-contracts/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/pkg/validation"
)

func (request AuthRequest) ValidateRegister() error {
	errs := request.validationErrors()
	validation.NewPasswordCharacters(errs, "password", request.Password)
	if errs.Empty() {
		return nil
	}
	return errs
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
	validation.Email(errs, "email", request.Email)
	validation.CredentialLength(errs, "password", request.Password)
	return errs
}
