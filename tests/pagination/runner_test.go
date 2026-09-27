package pagination_test

import (
	"testing"

	"github.com/siti-nabila/rest-orc/tests/pagination/test_scenarios"
	"github.com/siti-nabila/rest-orc/tests/shared/testutils"
)

func TestPagination(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.All())
}
