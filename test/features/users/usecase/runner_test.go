package usecase_test

import (
	"testing"

	"github.com/siti-nabila/rest-orc/test/features/users/usecase/test_scenarios"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func TestListUsersUseCase(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.All())
}

func TestUpdateProfileUseCase(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.ProfileScenarios())
}
