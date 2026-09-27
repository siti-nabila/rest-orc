package grpcauth_test

import (
	"testing"

	"github.com/siti-nabila/rest-orc/tests/client/grpcauth/test_scenarios"
	"github.com/siti-nabila/rest-orc/tests/shared/testutils"
)

func TestGRPCAuthClient(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.All())
}
