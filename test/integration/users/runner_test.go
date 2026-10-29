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
