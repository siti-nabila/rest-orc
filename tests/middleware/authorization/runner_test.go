package authorization_test

import (
	"testing"

	"github.com/siti-nabila/rest-orc/tests/middleware/authorization/test_scenarios"
	"github.com/siti-nabila/rest-orc/tests/shared/testutils"
)

func TestAuthorizationMiddleware(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.All())
}
