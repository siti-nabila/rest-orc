package auth_test

import (
	"testing"

	"github.com/siti-nabila/rest-orc/test/middleware/auth/test_scenarios"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func TestAuthenticationMiddleware(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.All())
}
