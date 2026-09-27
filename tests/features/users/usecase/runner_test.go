package usecase_test

import (
	"testing"

	"github.com/siti-nabila/rest-orc/tests/features/users/usecase/test_scenarios"
	"github.com/siti-nabila/rest-orc/tests/shared/testutils"
)

func TestListUsersUseCase(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.All())
}
