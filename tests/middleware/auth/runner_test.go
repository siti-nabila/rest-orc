package auth_test

import (
	"testing"

	"github.com/siti-nabila/rest-orc/tests/middleware/auth/test_scenarios"
	"github.com/siti-nabila/rest-orc/tests/shared/testutils"
)

func TestAuthenticationMiddleware(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.All())
}
