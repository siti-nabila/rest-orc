package list_flow_test

import (
	"testing"

	"github.com/siti-nabila/rest-orc/test/integration/users/test_scenarios"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func TestListUsersFlow(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.All())
}

func TestUpdateProfileFlow(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.ProfileScenarios())
}

func TestAdminSessions(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.AdminSessionScenarios())
}

func TestUserDetail(t *testing.T) { testutils.RunScenarios(t, test_scenarios.UserDetailScenarios()) }

func TestUserStatus(t *testing.T) { testutils.RunScenarios(t, test_scenarios.UserStatusScenarios()) }

func TestUserProfile(t *testing.T) { testutils.RunScenarios(t, test_scenarios.UserProfileScenarios()) }

func TestUserCreation(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.UserCreationScenarios())
}
