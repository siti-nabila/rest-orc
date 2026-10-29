package handler_test

import (
	"testing"

	"github.com/siti-nabila/rest-orc/test/features/auth/handler/test_scenarios"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func TestAuthHTTPHandler(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.All())
}

func TestChangePasswordHandler(t *testing.T) {
	testutils.RunScenarios(t, test_scenarios.PasswordScenarios())
}

func TestBrowserPolicy(t *testing.T) { testutils.RunScenarios(t, test_scenarios.BrowserScenarios()) }

func TestSessionHandler(t *testing.T) { testutils.RunScenarios(t, test_scenarios.SessionScenarios()) }

func TestMySessions(t *testing.T) { testutils.RunScenarios(t, test_scenarios.MySessionsScenarios()) }
