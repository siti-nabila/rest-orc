package domain

import (
	"strings"

	"github.com/siti-nabila/api-contracts/pkg/dictionary"
	authdictionary "github.com/siti-nabila/api-contracts/pkg/dictionary/auth"
	pkgdictionary "github.com/siti-nabila/rest-orc/pkg/dictionary"
)

func (request UpdateProfileRequest) NormalizeAndValidate() (UpdateProfileRequest, error) {
	result := request
	errs := dictionary.FieldErrors{}
	if request.Name != nil {
		value := normalizeName(*request.Name, errs)
		result.Name = &value
	}
	if request.CountryCode != nil {
		value := normalizeCountryCode(*request.CountryCode, errs)
		result.CountryCode = &value
	}
	if request.Phone != nil {
		value := normalizePhone(*request.Phone, result.CountryCode, errs)
		result.Phone = &value
	}
	if request.Address != nil {
		value := *request.Address
		validateProfileLength(errs, "address", value, 3, 200)
		if !validAddress(value) {
			errs.Add("address", pkgdictionary.ErrInvalidProfileCharacters)
		}
		result.Address = &value
	}
	if !errs.Empty() {
		return result, errs
	}
	return result, nil
}

func normalizeCountryCode(value string, errs dictionary.FieldErrors) string {
	if asciiDigits(value) {
		value = "+" + value
	}
	if value == "" {
		errs.Add("country_code", authdictionary.ErrRequired)
	}
	if !validCountryCode(value) {
		errs.Add("country_code", pkgdictionary.ErrInvalidCountryCode)
	}
	return value
}

func normalizePhone(value string, countryCode *string, errs dictionary.FieldErrors) string {
	if countryCode != nil && *countryCode == "+62" {
		if strings.HasPrefix(value, "+62") {
			value = strings.TrimPrefix(value, "+62")
		} else {
			value = strings.TrimPrefix(value, "62")
		}
		value = strings.TrimPrefix(value, "0")
	}
	validateProfileLength(errs, "phone", value, 12, 20)
	if value != "" && !asciiDigits(value) {
		errs.Add("phone", pkgdictionary.ErrInvalidPhone)
	}
	return value
}

func normalizeName(value string, errs dictionary.FieldErrors) string {
	// Check raw bytes before normalization can discard any characters.
	if !validName(value) {
		errs.Add("name", pkgdictionary.ErrInvalidProfileCharacters)
	}
	value = strings.Join(strings.FieldsFunc(value, func(c rune) bool { return c == ' ' }), " ")
	chars := []byte(value)
	start := true
	for i, c := range chars {
		if start && c >= 'a' && c <= 'z' {
			chars[i] = c - ('a' - 'A')
		}
		start = c == ' ' || c == '\'' || c == '-'
	}
	normalized := string(chars)
	validateProfileLength(errs, "name", normalized, 3, 100)
	return normalized
}

func validName(value string) bool {
	for i := 0; i < len(value); i++ {
		c := value[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == ' ' || c == '\'' || c == '-') {
			return false
		}
	}
	return true
}

func asciiDigits(value string) bool {
	if value == "" {
		return false
	}
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return true
}

func validCountryCode(value string) bool {
	return len(value) >= 2 && len(value) <= 4 && value[0] == '+' && value[1] >= '1' && value[1] <= '9' && asciiDigits(value[1:])
}

func validAddress(value string) bool {
	for i := 0; i < len(value); i++ {
		c := value[i]
		if c != '\r' && c != '\n' && (c < 0x20 || c > 0x7e) {
			return false
		}
	}
	return true
}

func validateProfileLength(errs dictionary.FieldErrors, field, value string, min, max int) {
	if value == "" {
		errs.Add(field, authdictionary.ErrRequired)
	}
	if len(value) < min {
		errs.Add(field, authdictionary.ErrMinLength(min))
	}
	if len(value) > max {
		errs.Add(field, authdictionary.ErrMaxLength(max))
	}
}
