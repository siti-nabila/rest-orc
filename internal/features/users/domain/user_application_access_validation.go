package domain

import (
	cd "github.com/siti-nabila/api-contracts/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

func (r ApplicationAccessCommandRequest) normalize(reason *string, errs cd.FieldErrors) (ApplicationAccessCommand, error) {
	id, err := ParseUserDetailID(r.UserID)
	if err != nil {
		return ApplicationAccessCommand{}, err
	}
	out := ApplicationAccessCommand{UserID: id, ApplicationCode: r.ApplicationCode, Version: r.Version, IdempotencyKey: r.IdempotencyKey}
	if err = validateUserCommandPreconditions(id, r.Version, r.IdempotencyKey); err != nil {
		return ApplicationAccessCommand{}, err
	}
	if !validUserRoleApplication(r.ApplicationCode) {
		errs.Add("application_code", dictionary.ErrInvalidRoleApplication)
	}
	out.Reason = normalizeAssignmentReason(reason, errs)
	return out, nil
}

func (r GrantApplicationAccessRequest) NormalizeAndValidate() (GrantApplicationAccessCommand, error) {
	errs := cd.FieldErrors{}
	command, err := r.ApplicationAccessCommandRequest.normalize(r.Reason, errs)
	if err != nil {
		return GrantApplicationAccessCommand{}, err
	}
	roles := normalizeAssignmentRoles(r.RoleIDs, errs)
	if !errs.Empty() {
		return GrantApplicationAccessCommand{}, errs
	}
	return GrantApplicationAccessCommand{ApplicationAccessCommand: command, RoleIDs: roles}, nil
}

func (r RevokeApplicationAccessRequest) NormalizeAndValidate() (ApplicationAccessCommand, error) {
	errs := cd.FieldErrors{}
	command, err := r.ApplicationAccessCommandRequest.normalize(r.Reason, errs)
	if err != nil {
		return ApplicationAccessCommand{}, err
	}
	if !errs.Empty() {
		return ApplicationAccessCommand{}, errs
	}
	return command, nil
}

func (r ApplicationAccessCommand) Validate() error {
	if err := validateUserCommandPreconditions(r.UserID, r.Version, r.IdempotencyKey); err != nil {
		return err
	}
	errs := cd.FieldErrors{}
	if !validUserRoleApplication(r.ApplicationCode) {
		errs.Add("application_code", dictionary.ErrInvalidRoleApplication)
	}
	if !validAddress(r.Reason) {
		errs.Add("reason", dictionary.ErrInvalidStatusReasonCharacters)
	}
	validateProfileLength(errs, "reason", r.Reason, 3, 500)
	if !errs.Empty() {
		return errs
	}
	return nil
}

func (r GrantApplicationAccessCommand) Validate() error {
	if err := r.ApplicationAccessCommand.Validate(); err != nil {
		return err
	}
	errs := cd.FieldErrors{}
	validateAssignmentRoles(r.RoleIDs, errs)
	if !errs.Empty() {
		return errs
	}
	return nil
}
