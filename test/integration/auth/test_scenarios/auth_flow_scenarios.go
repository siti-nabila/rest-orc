package test_scenarios

import (
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/rest-orc/internal/client/authservice"
	"github.com/siti-nabila/rest-orc/internal/client/grpcclient"
	grpcfixtures "github.com/siti-nabila/rest-orc/test/client/grpcclient/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

const apiV1Prefix = "/api/v1"

func All() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "browser session login bootstrap rotation and logout modes", Run: sessionBrowserFlow},
		{Name: "session local denials dispatch no RPCs", Run: sessionLocalDenials},
		{Name: "session dependency failures do not mutate cookies or retry", Run: sessionDependencyFailures},
		{Name: "registration and login validate complete authoritative results", Run: sessionRegistrationAndLoginResults},
		{Name: "public auth forwards raw credentials and preserves success contract", Run: publicAuthSuccess},
		{Name: "public login preserves downstream 401 without retry", Run: loginServiceFailure},
		{Name: "public auth validation aggregates localized fields before RPC", Run: localAuthErrors},
		{Name: "public auth parsing failures retain 400 before RPC", Run: authParsingErrors},
		{Name: "register rejects forbidden password characters before RPC", Run: registerCharacterErrors},
		{Name: "preserves register service conflict and retry guidance", Run: registerErrorResponse},
	}
}
func newAuthApp(t *testing.T, service *mocks.Service) *fiber.App {
	t.Helper()
	transport, err := grpcclient.New(grpcfixtures.ValidConfig())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := transport.Close(); err != nil {
			t.Error(err)
		}
	})
	client, err := authservice.NewWithService(transport, service)
	if err != nil {
		t.Fatal(err)
	}
	return fixtures.NewApp(t, client)
}
