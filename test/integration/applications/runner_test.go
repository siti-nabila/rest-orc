package applications_test

import (
	"testing"

	"github.com/siti-nabila/rest-orc/test/integration/applications/test_scenarios"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func TestApplicationCatalog(t *testing.T) { testutils.RunScenarios(t, test_scenarios.All()) }
