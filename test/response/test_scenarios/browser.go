package test_scenarios

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	"github.com/siti-nabila/rest-orc/internal/response"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"strings"
	"testing"
)

func browserErrors(t *testing.T) {
	for _, lang := range []locale.Language{locale.English, locale.Indonesian, locale.Chinese} {
		for _, tc := range []struct {
			err    error
			status int
		}{
			{common.ErrUnauthorized, 401}, {common.ErrForbidden, 403}, {fiber.ErrUnsupportedMediaType, 415},
			{dictionary.ErrSessionResultInvalid, 500}, {dictionary.ErrCSRFGenerationFailed, 500},
		} {
			mapped := response.NewErrorMapper().Map(tc.err, lang)
			if mapped.Status != tc.status {
				t.Fatalf("status=%d want %d", mapped.Status, tc.status)
			}
			b, _ := json.Marshal(mapped.Body)
			if strings.Contains(string(b), "session dependency") || strings.Contains(string(b), "browser proof") {
				t.Fatal("internal guard leaked")
			}
		}
	}
}
