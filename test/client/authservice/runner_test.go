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

func TestAdminSessions(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.AdminSessionsScenarios())
}

func TestUserDetail(t *testing.T) { testutils.RunScenarios(t, test_scenarios.UserDetailScenarios()) }

func TestUserStatus(t *testing.T) { testutils.RunScenarios(t, test_scenarios.UserStatusScenarios()) }

func TestUserProfile(t *testing.T) { testutils.RunScenarios(t, test_scenarios.UserProfileScenarios()) }

func TestUserCreation(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.UserCreationScenarios())
}

func TestRoleCatalog(t *testing.T) { testutils.RunScenarios(t, test_scenarios.RoleCatalogScenarios()) }

func TestUserRoles(t *testing.T) { testutils.RunScenarios(t, test_scenarios.UserRoleScenarios()) }

func TestApplicationCatalog(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.ApplicationCatalogScenarios())
}

func TestUserApplicationAccess(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.UserApplicationAccessScenarios())
}
