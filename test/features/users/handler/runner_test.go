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
