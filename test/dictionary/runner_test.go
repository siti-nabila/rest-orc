package dictionary_test

import (
	"testing"

	"github.com/siti-nabila/rest-orc/test/dictionary/test_scenarios"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func TestDictionary(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.All())
}
