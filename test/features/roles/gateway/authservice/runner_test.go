package authservice_test

import (
	"testing"

	"github.com/siti-nabila/rest-orc/test/features/roles/gateway/authservice/test_scenarios"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func TestRoleCatalog(t *testing.T) { testutils.RunScenarios(t, test_scenarios.All()) }
