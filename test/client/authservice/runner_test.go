package authservice_test

import (
	"testing"

	"github.com/siti-nabila/rest-orc/test/client/authservice/test_scenarios"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func TestChangePasswordClient(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.PasswordScenarios())
}

func TestGRPCAuthClient(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.All())
}

func TestProfileClient(t *testing.T) { testutils.RunScenarios(t, test_scenarios.ProfileScenarios()) }

func TestSessionClient(t *testing.T) { testutils.RunScenarios(t, test_scenarios.SessionScenarios()) }

func TestMySessions(t *testing.T) { testutils.RunScenarios(t, test_scenarios.MySessionsScenarios()) }
