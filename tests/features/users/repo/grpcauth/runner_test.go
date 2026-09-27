package grpcauth_test

import (
	"testing"

	"github.com/siti-nabila/rest-orc/tests/features/users/repo/grpcauth/test_scenarios"
	"github.com/siti-nabila/rest-orc/tests/shared/testutils"
)

func TestUsersGRPCAuthRepository(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.All())
}
