package test_scenarios

import (
	"errors"
	"fmt"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/api-contracts/pkg/dictionary"
	authdictionary "github.com/siti-nabila/api-contracts/pkg/dictionary/auth"
	commondictionary "github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	errorpackage "github.com/siti-nabila/error-package"
	localdictionary "github.com/siti-nabila/rest-orc/pkg/dictionary"
)

func localValidationErrors(t *testing.T) {
	lengthMessages := map[string][]string{
		"en":    {"minimum length is 6", "maximum length is 100"},
		"id-ID": {"minimal panjang karakter adalah 6", "maksimal panjang karakter adalah 100"},
		"zh-CN": {"最小长度为 6", "最大长度为 100"},
	}
	fields := map[string][]error{
		"email": {authdictionary.ErrRequired, authdictionary.ErrInvalidEmail},
		"name":  {authdictionary.ErrMinLength(6), authdictionary.ErrMaxLength(100), authdictionary.ErrAlphaOnly},
		"profile": {errorpackage.Errors{
			"contact": {dictionary.FieldErrors{"email": {authdictionary.ErrRequired}}},
		}},
	}
	for _, container := range []struct {
		name string
		err  error
	}{
		{"dictionary", dictionary.FieldErrors(fields)},
		{"error-package", errorpackage.Errors(fields)},
	} {
		for _, wrapped := range []bool{false, true} {
			for _, language := range []string{"en", "id-ID", "zh-CN"} {
				t.Run(fmt.Sprintf("%s/wrapped=%t/%s", container.name, wrapped, language), func(t *testing.T) {
					err := container.err
					if wrapped {
						err = fmt.Errorf("private wrapper: %w", err)
					}
					lang := locale.Parse(language)
					// Expected public messages come from the existing catalog, not Error().
					want := map[string][]string{
						"email": {authdictionary.ErrRequired.Message(lang), authdictionary.ErrInvalidEmail.Message(lang)},
						"name": {
							lengthMessages[language][0],
							lengthMessages[language][1],
							authdictionary.ErrAlphaOnly.Message(lang),
						},
						"profile.contact.email": {authdictionary.ErrRequired.Message(lang)},
					}
					assertErrorResponse(t, errorResponse(t, err, language), 400, "BR", want)
				})
			}
		}
	}

	t.Run("unknown field details are masked", func(t *testing.T) {
		fields := dictionary.FieldErrors{"email": {errors.New("database password private")}}
		assertErrorResponse(t, errorResponse(t, fields, "en"), 400, "BR",
			map[string][]string{"email": {commondictionary.ErrBadRequest.Message(locale.English)}})
	})
}

func localCatalogErrors(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		code   string
		text   string
	}{
		{"catalog conflict", authdictionary.ErrDataExists, 409, "EX", "Already exists."},
		{"bad request", fiber.ErrBadRequest, 400, "BR", "Bad request."},
		{"unauthorized", fiber.ErrUnauthorized, 401, commondictionary.ErrUnauthorized.Code(), commondictionary.ErrUnauthorized.Message(locale.English)},
		{"forbidden", fiber.ErrForbidden, 403, commondictionary.ErrForbidden.Code(), commondictionary.ErrForbidden.Message(locale.English)},
		{"missing endpoint", fiber.ErrNotFound, 404, "NF", "Endpoint not found."},
		{"unknown fiber error", fiber.NewError(418, "private message"), 500, "IS", "Internal server error."},
		{"configuration", localdictionary.NewValidationError("APP_PORT", errors.New("private detail")), 500, "IS", "Internal server error."},
		{"unknown error", errors.New("private credentials"), 500, "IS", "Internal server error."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := fmt.Errorf("private context: %w", tc.err)
			assertErrorResponse(t, errorResponse(t, err, "en"), tc.status, tc.code, map[string]string{"description": tc.text})
		})
	}
}
