package dictionary

type wrappedError struct {
	registered error
	cause      error
}

func (err *wrappedError) Error() string {
	return err.registered.Error()
}

func (err *wrappedError) Unwrap() []error {
	if err.cause == nil {
		return []error{err.registered}
	}
	return []error{err.registered, err.cause}
}

func newErrorWithCause(key string, cause error, args ...any) error {
	return &wrappedError{
		registered: parameterizedErrorPack.Newf(key, args...),
		cause:      cause,
	}
}
