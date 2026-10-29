package domain_test

import (
	"testing"

	"github.com/siti-nabila/rest-orc/test/features/auth/domain/test_scenarios"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func TestAuthValidation(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.All())
}

func TestChangePasswordValidation(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.PasswordScenarios())
}

func TestSessionContracts(t *testing.T) { testutils.RunScenarios(t, test_scenarios.SessionScenarios()) }

func TestMySessions(t *testing.T) { testutils.RunScenarios(t, test_scenarios.MySessionsScenarios()) }
