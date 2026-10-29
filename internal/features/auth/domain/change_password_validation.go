package domain

import (
	"github.com/siti-nabila/api-contracts/pkg/dictionary"
	pkgdictionary "github.com/siti-nabila/rest-orc/pkg/dictionary"
)

func (request ChangePasswordRequest) Validate() error {
	errs := dictionary.FieldErrors{}
	validateAuthLength(errs, "current_password", request.CurrentPassword)
	validateAuthLength(errs, "new_password", request.NewPassword)
	validateNewPasswordCharacters(errs, "new_password", request.NewPassword)
	if request.CurrentPassword != "" && request.CurrentPassword == request.NewPassword {
		errs.Add("new_password", pkgdictionary.ErrPasswordMustDiffer)
	}
	if errs.Empty() {
		return nil
	}
	return errs
}
