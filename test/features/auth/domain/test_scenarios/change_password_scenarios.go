package test_scenarios

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/siti-nabila/api-contracts/pkg/dictionary"
	authdictionary "github.com/siti-nabila/api-contracts/pkg/dictionary/auth"
	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	pkgdictionary "github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func PasswordScenarios() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "password change checks byte boundaries without mutation", Run: changePasswordLengths},
		{Name: "current password permits legacy characters and new password uses register policy", Run: changePasswordCharacters},
		{Name: "password change aggregates equality and presence errors", Run: changePasswordEquality},
	}
}

func changePasswordLengths(t *testing.T) {
	for _, field := range []string{"current_password", "new_password"} {
		for _, size := range []int{0, 5, 6, 50, 51} {
			request := domain.ChangePasswordRequest{CurrentPassword: "legacy-password", NewPassword: "new-password"}
			value := strings.Repeat("a", size)
			if field == "current_password" {
				request.CurrentPassword = value
			} else {
				request.NewPassword = value
			}
			want := dictionary.FieldErrors{}
			if size == 0 {
				want.Add(field, authdictionary.ErrRequired)
			}
			if size < 6 {
				want.Add(field, authdictionary.ErrMinLength(6))
			}
			if size > 50 {
				want.Add(field, authdictionary.ErrMaxLength(50))
			}
			before := request
			assertFieldErrors(t, request.Validate(), want)
			if request != before {
				t.Fatal("validation mutated a credential")
			}
		}
	}
}

func changePasswordCharacters(t *testing.T) {
	for _, value := range []string{"      ", " PaSs ", "ééé", strings.Repeat("é", 25), "ab\x00\t\r\n", "ab\x7fxyz"} {
		request := domain.ChangePasswordRequest{CurrentPassword: value, NewPassword: "new-password"}
		assertFieldErrors(t, request.Validate(), nil)
		request = domain.ChangePasswordRequest{CurrentPassword: "legacy-password", NewPassword: value}
		assertFieldErrors(t, request.Validate(), dictionary.FieldErrors{
			"new_password": {pkgdictionary.ErrInvalidPasswordCharacters},
		})
	}
	for _, value := range []string{"aaaaaa", "123456", "!!!!!!", "!aaaa~", strings.Repeat("~", 50)} {
		request := domain.ChangePasswordRequest{CurrentPassword: "legacy-password", NewPassword: value}
		assertFieldErrors(t, request.Validate(), nil)
	}
	// A single rule contributes one character message even with several forbidden bytes.
	request := domain.ChangePasswordRequest{CurrentPassword: "short", NewPassword: " \t"}
	assertFieldErrors(t, request.Validate(), dictionary.FieldErrors{
		"current_password": {authdictionary.ErrMinLength(6)},
		"new_password":     {authdictionary.ErrMinLength(6), pkgdictionary.ErrInvalidPasswordCharacters},
	})
}

func changePasswordEquality(t *testing.T) {
	request := domain.ChangePasswordRequest{CurrentPassword: "same-password", NewPassword: "same-password"}
	assertFieldErrors(t, request.Validate(), dictionary.FieldErrors{
		"new_password": {pkgdictionary.ErrPasswordMustDiffer},
	})
	request.NewPassword = "Same-password"
	assertFieldErrors(t, request.Validate(), nil)
	for _, payload := range []string{`{}`, `null`, `{"current_password":null,"new_password":null}`} {
		var request domain.ChangePasswordRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			t.Fatal(err)
		}
		assertFieldErrors(t, request.Validate(), dictionary.FieldErrors{
			"current_password": {authdictionary.ErrRequired, authdictionary.ErrMinLength(6)},
			"new_password":     {authdictionary.ErrRequired, authdictionary.ErrMinLength(6)},
		})
	}
	request = domain.ChangePasswordRequest{CurrentPassword: "a b", NewPassword: "a b"}
	assertFieldErrors(t, request.Validate(), dictionary.FieldErrors{
		"current_password": {authdictionary.ErrMinLength(6)},
		"new_password":     {authdictionary.ErrMinLength(6), pkgdictionary.ErrInvalidPasswordCharacters, pkgdictionary.ErrPasswordMustDiffer},
	})
}
