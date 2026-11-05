package domain

import (
	"strconv"

	contractdictionary "github.com/siti-nabila/api-contracts/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

func ParseUserDetailID(value string) (uint64, error) {
	id, err := strconv.ParseUint(value, 10, 31)
	if !asciiDigits(value) || err != nil || id == 0 {
		return 0, contractdictionary.FieldErrors{"user_id": {dictionary.ErrInvalidUserID}}
	}
	return id, nil
}

func ValidateUserDetailID(id uint64) error {
	if id == 0 || id > 2147483647 {
		return contractdictionary.FieldErrors{"user_id": {dictionary.ErrInvalidUserID}}
	}
	return nil
}
