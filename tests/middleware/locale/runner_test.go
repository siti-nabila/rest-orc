package locale_test

import (
	"testing"

	"github.com/siti-nabila/rest-orc/tests/middleware/locale/test_scenarios"
	"github.com/siti-nabila/rest-orc/tests/shared/testutils"
)

func TestLocaleMiddleware(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.All())
}
