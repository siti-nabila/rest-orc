package test_scenarios

import (
	"encoding/json"
	"errors"
	contractdictionary "github.com/siti-nabila/api-contracts/pkg/dictionary"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"reflect"
	"strings"
	"testing"
)

func All() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "normalization boundaries and raw character allowlists", Run: boundaries},
		{Name: "presence null and independent localized field errors", Run: presence},
	}
}
func boundaries(t *testing.T) {
	type example struct {
		field, input, country, want string
		invalid                     bool
	}
	cases := []example{
		{field: "name", input: "  NASA  o'neil-jean  ", want: "NASA O'Neil-Jean"},
		{field: "name", input: "ab", invalid: true}, {field: "name", input: "abc", want: "Abc"},
		{field: "name", input: strings.Repeat("A", 100), want: strings.Repeat("A", 100)},
		{field: "name", input: strings.Repeat("A", 101), invalid: true},
		{field: "address", input: " a\r\nb ", want: " a\r\nb "},
		{field: "address", input: "abc", want: "abc"}, {field: "address", input: "ab", invalid: true},
		{field: "address", input: strings.Repeat("x", 200), want: strings.Repeat("x", 200)},
		{field: "address", input: strings.Repeat("x", 201), invalid: true},
	}
	for _, value := range []string{"", "a1b", "a\tb", "a\rb", "a\nb", "a\x00b", "aéz", "a\u00a0b"} {
		cases = append(cases, example{field: "name", input: value, invalid: true})
	}
	for _, value := range []string{"", "ab\t", "ab\x00", "ab\x7f", "ab\x01", "中文"} {
		cases = append(cases, example{field: "address", input: value, invalid: true})
	}
	for _, value := range []string{"1", "62", "123", "+62"} {
		want := value
		if value[0] != '+' {
			want = "+" + value
		}
		cases = append(cases, example{field: "country_code", input: value, want: want})
	}
	for _, value := range []string{"", "0", "01", "+01", "1234", "+1234", " 62", "62 ", "６２", "++62", "+"} {
		cases = append(cases, example{field: "country_code", input: value, invalid: true})
	}
	for _, value := range []string{"+62812345678901", "62812345678901", "0812345678901", "812345678901"} {
		cases = append(cases, example{field: "phone", input: value, country: "62", want: "812345678901"})
	}
	for _, value := range []string{"0812345678901", "62812345678901", "812345678901", strings.Repeat("9", 20)} {
		cases = append(cases, example{field: "phone", input: value, want: value})
	}
	for _, value := range []string{"", strings.Repeat("1", 11), strings.Repeat("1", 21), "+62812345678901", "8123 45678901", "8123-45678901", "８12345678901"} {
		cases = append(cases, example{field: "phone", input: value, invalid: true})
	}
	cases = append(cases, example{field: "phone", input: "081234567890", country: "+62", invalid: true}, example{field: "phone", input: "0812345678901", country: "1", want: "0812345678901"}, example{field: "phone", input: "++62812345678901", country: "+62", invalid: true}, example{field: "phone", input: "0062812345678901", country: "+62", want: "062812345678901"})
	for _, tc := range cases {
		t.Run(tc.field+"/"+tc.input, func(t *testing.T) {
			value := tc.input
			request := domain.UpdateProfileRequest{}
			switch tc.field {
			case "name":
				request.Name = &value
			case "phone":
				request.Phone = &value
			case "country_code":
				request.CountryCode = &value
			case "address":
				request.Address = &value
			}
			if tc.country != "" {
				request.CountryCode = &tc.country
			}
			before, _ := json.Marshal(request)
			got, err := request.NormalizeAndValidate()
			if (err != nil) != tc.invalid {
				t.Fatalf("error=%v, invalid=%v", err, tc.invalid)
			}
			after, _ := json.Marshal(request)
			if string(before) != string(after) {
				t.Fatal("input mutated")
			}
			if tc.invalid {
				return
			}
			var actual *string
			switch tc.field {
			case "name":
				actual = got.Name
			case "phone":
				actual = got.Phone
			case "country_code":
				actual = got.CountryCode
			case "address":
				actual = got.Address
			}
			if actual == nil || *actual != tc.want {
				t.Fatalf("result=%v, want %q", actual, tc.want)
			}
			if actual == &value {
				t.Fatal("normalized value aliases caller")
			}
		})
	}
}
func presence(t *testing.T) {
	for _, raw := range []string{`{}`, `null`, `{"name":null,"phone":null,"country_code":null,"address":null}`, `{"user_id":88,"field_mask":["phone"]}`} {
		var request domain.UpdateProfileRequest
		if err := json.Unmarshal([]byte(raw), &request); err != nil {
			t.Fatal(err)
		}
		got, err := request.NormalizeAndValidate()
		if err != nil || got.HasFields() {
			t.Fatalf("%s: %#v %v", raw, got, err)
		}
	}
	empty := ""
	request := domain.UpdateProfileRequest{Name: &empty, Phone: &empty, CountryCode: &empty, Address: &empty}
	_, err := request.NormalizeAndValidate()
	var fields contractdictionary.FieldErrors
	if !errors.As(err, &fields) {
		t.Fatalf("error=%v", err)
	}
	for _, language := range []locale.Language{locale.English, locale.Indonesian, locale.Chinese} {
		messages := fields.Messages(language)
		if len(messages) != 4 {
			t.Fatalf("messages=%v", messages)
		}
		for _, field := range []string{"name", "phone", "country_code", "address"} {
			if len(messages[field]) == 0 {
				t.Fatalf("missing %s", field)
			}
		}
	}
	country := "62"
	got, err := (domain.UpdateProfileRequest{CountryCode: &country}).NormalizeAndValidate()
	if err != nil || got.Phone != nil || !reflect.DeepEqual(*got.CountryCode, "+62") {
		t.Fatalf("country only: %#v %v", got, err)
	}
}
