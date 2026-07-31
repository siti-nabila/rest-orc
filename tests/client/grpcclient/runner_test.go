package grpcclient_test

import (
	"testing"

	scenarios "github.com/siti-nabila/rest-orc/tests/client/grpcclient/test_scenarios"
	"github.com/siti-nabila/rest-orc/tests/shared/testutils"
)

func TestGRPCClient(t *testing.T) {
	testutils.RunScenarios(t, scenarios.All())
}
