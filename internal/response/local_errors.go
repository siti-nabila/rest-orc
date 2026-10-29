package response

import (
	"errors"

	"github.com/siti-nabila/api-contracts/pkg/dictionary"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	errorpackage "github.com/siti-nabila/error-package"
)

func localFieldErrors(err error, language locale.Language) (map[string][]string, bool) {
	var fields map[string][]error
	if typed, ok := errors.AsType[dictionary.FieldErrors](err); ok {
		fields = typed
	} else if typed, ok := errors.AsType[errorpackage.Errors](err); ok {
		fields = typed
	} else {
		return nil, false
	}
	messages := make(map[string][]string)
	appendFieldMessages(messages, "", fields, language)
	return messages, true
}

func appendFieldMessages(messages map[string][]string, prefix string, fields map[string][]error, language locale.Language) {
	for field, list := range fields {
		name := field
		if prefix != "" {
			name = prefix + "." + field
		}
		for _, err := range list {
			switch nested := err.(type) {
			case dictionary.FieldErrors:
				appendFieldMessages(messages, name, nested, language)
			case errorpackage.Errors:
				appendFieldMessages(messages, name, nested, language)
			default:
				messages[name] = append(messages[name], localFieldMessage(err, language))
			}
		}
	}
}

func localFieldMessage(err error, language locale.Language) string {
	switch typed := err.(type) {
	case interface{ Message(locale.Language) string }:
		return typed.Message(language)
	case interface {
		Message(errorpackage.LanguageCode) string
	}:
		return typed.Message(errorpackage.LanguageCode(language))
	default:
		return common.ErrBadRequest.Message(language)
	}
}
