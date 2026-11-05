package test_scenarios

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	cd "github.com/siti-nabila/api-contracts/pkg/dictionary"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func UserProfileScenarios() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "target profile header presence and numeric boundaries", Run: targetProfileHeaders},
		{Name: "target profile selected subsets reuse normalized values without aliasing", Run: targetProfileValues},
	}
}
func targetProfileHeaders(t *testing.T) {
	userDetailIDs(t)
	for _, v := range []string{"1", "00023", "9007199254740993", "9223372036854775807"} {
		key, version, err := domain.ParseUserProfileHeaders(nil, []string{v})
		if err != nil || key != nil || version == 0 {
			t.Fatal(key, version, err)
		}
		for _, k := range []string{"x", "opaque, key", " x ", strings.Repeat("x", 200)} {
			key, _, err = domain.ParseUserProfileHeaders([]string{k}, []string{v})
			if err != nil || key == nil || *key != k {
				t.Fatal(key, err)
			}
		}
	}
	for _, keys := range [][]string{{""}, {" "}, {"a\t"}, {"a\n"}, {"é"}, {"a\x7f"}, {strings.Repeat("a", 201)}, {"key", "key"}} {
		if _, _, err := domain.ParseUserProfileHeaders(keys, []string{"1"}); err == nil {
			t.Fatal("key accepted", keys)
		}
	}
	for _, vs := range [][]string{nil, {"1", "1"}, {""}, {"0"}, {"+1"}, {"-1"}, {" 1"}, {"1 "}, {"１"}, {"0x1"}, {"1.0"}, {"9223372036854775808"}} {
		if _, _, err := domain.ParseUserProfileHeaders(nil, vs); err == nil {
			t.Fatal("version accepted", vs)
		}
	}
	for _, mode := range []string{"id zero", "id range", "version zero", "version range", "empty key"} {
		r := domain.UpdateUserProfileRequest{UserID: 23, Version: 1}
		switch mode {
		case "id zero":
			r.UserID = 0
		case "id range":
			r.UserID = 2147483648
		case "version zero":
			r.Version = 0
		case "version range":
			r.Version = 1 << 63
		case "empty key":
			empty := ""
			r.IdempotencyKey = &empty
		}
		if _, err := r.NormalizeAndValidate(); err == nil {
			t.Fatal(mode)
		}
	}
}
func targetProfileValues(t *testing.T) {
	for mask := 0; mask < 16; mask++ {
		t.Run(string(rune('A'+mask)), func(t *testing.T) {
			name, address, phone, country, key := "  NASA  o'neil-jean  ", " X\r\nY ", "0812345678901", "62", "opaque, key"
			p := domain.UpdateProfileRequest{}
			if mask&1 != 0 {
				p.Name = &name
			}
			if mask&2 != 0 {
				p.Address = &address
			}
			if mask&4 != 0 {
				p.Phone = &phone
			}
			if mask&8 != 0 {
				p.CountryCode = &country
			}
			r := domain.UpdateUserProfileRequest{UserID: 23, Version: 9007199254740993, IdempotencyKey: &key, Profile: p}
			before, _ := json.Marshal(r)
			want, err := p.NormalizeAndValidate()
			if err != nil {
				t.Fatal(err)
			}
			got, err := r.NormalizeAndValidate()
			if err != nil || !reflect.DeepEqual(got.Profile, want) || got.Profile.HasFields() != (mask != 0) || got.IdempotencyKey == &key {
				t.Fatal(got, err)
			}
			if got.Profile.Name != nil {
				*got.Profile.Name = "changed"
			}
			if got.Profile.Address != nil {
				*got.Profile.Address = "changed"
			}
			if got.Profile.Phone != nil {
				*got.Profile.Phone = "changed"
			}
			if got.Profile.CountryCode != nil {
				*got.Profile.CountryCode = "changed"
			}
			*got.IdempotencyKey = "changed"
			after, _ := json.Marshal(r)
			if string(before) != string(after) {
				t.Fatal("aliased")
			}
		})
	}
	for _, raw := range []string{`null`, `{}`, `{"name":null,"address":null,"phone":null,"country_code":null}`, `{"field_mask":["phone"],"user_id":99,"version":7,"idempotency_key":"forged"}`} {
		var p domain.UpdateProfileRequest
		if json.Unmarshal([]byte(raw), &p) != nil {
			t.Fatal(raw)
		}
		got, err := (domain.UpdateUserProfileRequest{UserID: 23, Version: 1, Profile: p}).NormalizeAndValidate()
		if err != nil || got.Profile.HasFields() {
			t.Fatal(got, err)
		}
	}
	invalid := ""
	_, err := (domain.UpdateUserProfileRequest{UserID: 23, Version: 1, Profile: domain.UpdateProfileRequest{Name: &invalid, Address: &invalid, Phone: &invalid, CountryCode: &invalid}}).NormalizeAndValidate()
	var fields cd.FieldErrors
	if !errors.As(err, &fields) || len(fields) != 4 {
		t.Fatal(err)
	}
	for _, lang := range []locale.Language{locale.English, locale.Indonesian, locale.Chinese} {
		if len(fields.Messages(lang)) != 4 {
			t.Fatal("locale")
		}
	}
}
