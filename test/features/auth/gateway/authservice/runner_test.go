package authservice_test

import (
	"testing"

	"github.com/siti-nabila/rest-orc/test/features/auth/gateway/authservice/test_scenarios"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func TestChangePasswordGateway(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.PasswordScenarios())
}

func TestAuthGateway(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.All())
}

func TestSessionGateway(t *testing.T) { testutils.RunScenarios(t, test_scenarios.SessionScenarios()) }

func TestMySessions(t *testing.T) { testutils.RunScenarios(t, test_scenarios.MySessionsScenarios()) }
