package domain

import (
	"github.com/siti-nabila/api-contracts/pkg/dictionary"
	pkgdictionary "github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/pkg/validation"
)

func (request ChangePasswordRequest) Validate() error {
	errs := dictionary.FieldErrors{}
	validation.CredentialLength(errs, "current_password", request.CurrentPassword)
	validation.CredentialLength(errs, "new_password", request.NewPassword)
	validation.NewPasswordCharacters(errs, "new_password", request.NewPassword)
	if request.CurrentPassword != "" && request.CurrentPassword == request.NewPassword {
		errs.Add("new_password", pkgdictionary.ErrPasswordMustDiffer)
	}
	if errs.Empty() {
		return nil
	}
	return errs
}
