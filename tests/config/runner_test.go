package config_test

import (
	"testing"

	scenarios "github.com/siti-nabila/rest-orc/tests/config/test_scenarios"
	"github.com/siti-nabila/rest-orc/tests/shared/testutils"
)

func TestConfig(t *testing.T) {
	testutils.RunScenarios(t, scenarios.All())
}
