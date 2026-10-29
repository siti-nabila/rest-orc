package locale_test

import (
	"testing"

	"github.com/siti-nabila/rest-orc/test/middleware/locale/test_scenarios"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func TestLocaleMiddleware(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.All())
}
