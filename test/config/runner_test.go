package config_test

import (
	"testing"

	scenarios "github.com/siti-nabila/rest-orc/test/config/test_scenarios"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func TestConfig(t *testing.T) {
	testutils.RunScenarios(t, scenarios.All())
}
