package pagination_test

import (
	"testing"

	"github.com/siti-nabila/rest-orc/test/pagination/test_scenarios"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func TestPagination(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.All())
}
