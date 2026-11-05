package test_scenarios

import (
	"errors"
	"strings"
	"testing"

	contractdictionary "github.com/siti-nabila/api-contracts/pkg/dictionary"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func UserDetailScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "detail target bounds and localized safe validation", Run: userDetailIDs}}
}

func userDetailIDs(t *testing.T) {
	for _, tc := range []struct {
		raw string
		id  uint64
	}{{"1", 1}, {"2147483647", 2147483647}, {"00023", 23}} {
		id, err := domain.ParseUserDetailID(tc.raw)
		if err != nil || id != tc.id || domain.ValidateUserDetailID(id) != nil {
			t.Fatal(tc.raw, id, err)
		}
	}
	for _, raw := range []string{"", "0", "000", "2147483648", "18446744073709551616", strings.Repeat("9", 100), "-1", "+1", " 1", "1 ", "１", "0x1", "1.0", "1\n"} {
		id, err := domain.ParseUserDetailID(raw)
		var fields contractdictionary.FieldErrors
		if id != 0 || !errors.As(err, &fields) {
			t.Fatal(raw, id, err)
		}
		for _, lang := range []locale.Language{locale.English, locale.Indonesian, locale.Chinese} {
			if len(fields.Messages(lang)["user_id"]) != 1 {
				t.Fatal("localized field error missing")
			}
		}
	}
	for _, id := range []uint64{0, 2147483648, ^uint64(0)} {
		if domain.ValidateUserDetailID(id) == nil {
			t.Fatal(id)
		}
	}
}
