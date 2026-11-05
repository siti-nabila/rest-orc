package auth_test

import (
	"testing"

	"github.com/siti-nabila/rest-orc/test/auth/test_scenarios"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func TestAuthenticator(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.All())
	testutils.RunScenarios(t, test_scenarios.MeSnapshots())
}
