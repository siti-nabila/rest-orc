package auth_test

import (
	"testing"

	"github.com/siti-nabila/rest-orc/tests/auth/test_scenarios"
	"github.com/siti-nabila/rest-orc/tests/shared/testutils"
)

func TestAuthenticator(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.All())
}
