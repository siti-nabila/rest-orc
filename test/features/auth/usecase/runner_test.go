package usecase_test

import (
	"testing"

	"github.com/siti-nabila/rest-orc/test/features/auth/usecase/test_scenarios"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func TestAuthUseCases(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.All())
}

func TestChangePasswordUseCase(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.PasswordScenarios())
}

func TestSessionContracts(t *testing.T) { testutils.RunScenarios(t, test_scenarios.SessionScenarios()) }

func TestMySessions(t *testing.T) { testutils.RunScenarios(t, test_scenarios.MySessionsScenarios()) }
