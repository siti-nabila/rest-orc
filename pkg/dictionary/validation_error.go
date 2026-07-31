package dictionary

type ValidationError struct {
	Field   string
	Problem error
	message error
}

func (err *ValidationError) Error() string {
	if err.message == nil {
		return parameterizedErrorPack.Newf(
			"invalid_configuration_field",
			err.Field,
			err.Problem,
		).Error()
	}
	return err.message.Error()
}

func (err *ValidationError) Unwrap() []error {
	if err.message == nil {
		if err.Problem == nil {
			return []error{ErrInvalidConfigurationField}
		}
		return []error{ErrInvalidConfigurationField, err.Problem}
	}
	if err.Problem == nil {
		return []error{err.message}
	}
	return []error{err.message, err.Problem}
}

func NewValidationError(field string, problem error) *ValidationError {
	return &ValidationError{
		Field:   field,
		Problem: problem,
		message: parameterizedErrorPack.Newf("invalid_configuration_field", field, problem),
	}
}
