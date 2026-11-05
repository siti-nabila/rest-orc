package app_test

import (
	"testing"

	scenarios "github.com/siti-nabila/rest-orc/test/app/test_scenarios"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func TestApp(t *testing.T) {
	testutils.RunScenarios(t, scenarios.All())
}
