package authservice_test

import (
	"testing"

	"github.com/siti-nabila/rest-orc/test/integration/authservice/test_scenarios"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func TestAuthServiceTransport(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.All())
}
