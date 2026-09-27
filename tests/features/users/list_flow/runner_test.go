package list_flow_test

import (
	"testing"

	"github.com/siti-nabila/rest-orc/tests/features/users/list_flow/test_scenarios"
	"github.com/siti-nabila/rest-orc/tests/shared/testutils"
)

func TestListUsersFlow(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.All())
}
