package response_test

import (
	"testing"

	"github.com/siti-nabila/rest-orc/tests/response/test_scenarios"
	"github.com/siti-nabila/rest-orc/tests/shared/testutils"
)

func TestResponse(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.All())
}
