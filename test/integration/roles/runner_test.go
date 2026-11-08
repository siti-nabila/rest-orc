package roles_test

import (
	"testing"

	"github.com/siti-nabila/rest-orc/test/integration/roles/test_scenarios"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func TestRoleCatalog(t *testing.T) { testutils.RunScenarios(t, test_scenarios.All()) }
