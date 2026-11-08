package handler_test

import (
	"testing"

	"github.com/siti-nabila/rest-orc/test/features/users/handler/test_scenarios"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func TestListUsersHTTPHandler(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.All())
}

func TestProfileHandler(t *testing.T) { testutils.RunScenarios(t, test_scenarios.ProfileScenarios()) }

func TestAdminSessions(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.AdminSessionScenarios())
}

func TestUserDetail(t *testing.T) { testutils.RunScenarios(t, test_scenarios.UserDetailScenarios()) }

func TestUserStatus(t *testing.T) { testutils.RunScenarios(t, test_scenarios.UserStatusScenarios()) }

func TestUserProfile(t *testing.T) { testutils.RunScenarios(t, test_scenarios.UserProfileScenarios()) }

func TestUserCreation(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.UserCreationScenarios())
}

func TestUserRoles(t *testing.T) { testutils.RunScenarios(t, test_scenarios.UserRoleScenarios()) }

func TestUserApplicationAccess(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.UserApplicationAccessScenarios())
}
