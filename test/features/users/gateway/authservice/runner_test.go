package authservice_test

import (
	"testing"

	"github.com/siti-nabila/rest-orc/test/features/users/gateway/authservice/test_scenarios"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func TestUsersGRPCAuthGateway(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.All())
}

func TestProfileGateway(t *testing.T) { testutils.RunScenarios(t, test_scenarios.ProfileScenarios()) }
