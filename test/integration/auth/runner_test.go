package auth_flow_test

import (
	"testing"

	"github.com/siti-nabila/rest-orc/test/integration/auth/test_scenarios"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func TestAuthFlow(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.All())
}

func TestChangePasswordFlow(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.PasswordScenarios())
}

func TestMySessions(t *testing.T) { testutils.RunScenarios(t, test_scenarios.MySessionsScenarios()) }
