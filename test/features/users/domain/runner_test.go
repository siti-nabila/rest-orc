package domain_test

import (
	"github.com/siti-nabila/rest-orc/test/features/users/domain/test_scenarios"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"testing"
)

func TestProfileValidation(t *testing.T) { testutils.RunScenarios(t, test_scenarios.All()) }

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
