package httpclient_test

import (
	"testing"

	scenarios "github.com/siti-nabila/rest-orc/test/client/httpclient/test_scenarios"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func TestHTTPClient(t *testing.T) {
	testutils.RunScenarios(t, scenarios.All())
}
