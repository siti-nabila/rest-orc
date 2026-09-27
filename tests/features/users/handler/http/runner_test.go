package http_test

import (
	"testing"

	"github.com/siti-nabila/rest-orc/tests/features/users/handler/http/test_scenarios"
	"github.com/siti-nabila/rest-orc/tests/shared/testutils"
)

func TestListUsersHTTPHandler(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.All())
}
