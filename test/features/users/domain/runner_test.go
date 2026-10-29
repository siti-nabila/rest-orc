package domain_test

import (
	"github.com/siti-nabila/rest-orc/test/features/users/domain/test_scenarios"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"testing"
)

func TestProfileValidation(t *testing.T) { testutils.RunScenarios(t, test_scenarios.All()) }
