package response_test

import (
	"testing"

	"github.com/siti-nabila/rest-orc/test/response/test_scenarios"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func TestResponse(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.All())
}
