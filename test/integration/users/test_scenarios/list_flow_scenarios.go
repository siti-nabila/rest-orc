package test_scenarios

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	authservice "github.com/siti-nabila/rest-orc/internal/client/authservice"
	"github.com/siti-nabila/rest-orc/internal/client/grpcclient"
	usershandler "github.com/siti-nabila/rest-orc/internal/features/users/handler"
	grpcclientfixtures "github.com/siti-nabila/rest-orc/test/client/grpcclient/fixtures"
	"github.com/siti-nabila/rest-orc/test/integration/users/fixtures"
	sharedfixtures "github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

const authorizationValue = "Bearer example-token"
const apiV1Prefix = "/api/v1"

func All() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "preserves protected service errors and skips business RPC on invalid input", Run: protectedErrorResponses},
		{
			Name: "admin request calls Me then ListUsers and returns paginator",
			Run:  adminRequestReturnsListUsersResponse,
		},
	}
}

func adminRequestReturnsListUsersResponse(t *testing.T) {
	// Arrange
	calls := make([]string, 0, 2)
	service := &mocks.Service{
		MeFunc: func(
			ctx context.Context,
			_ *emptypb.Empty,
			_ ...grpc.CallOption,
		) (*userv1.UserData, error) {
			assertAuthorizationMetadata(t, ctx)
			calls = append(calls, "Me")
			return fixtures.MeResponse(), nil
		},
		ListUsersFunc: func(
			ctx context.Context,
			request *userv1.ListUsersRequest,
			_ ...grpc.CallOption,
		) (*userv1.ListUsersResponse, error) {
			assertAuthorizationMetadata(t, ctx)
			calls = append(calls, "ListUsers")
			query := request.GetQuery()
			if query.GetPage() != 2 || query.GetLimit() != 20 || query.GetLastId() != "cursor-20" {
				t.Errorf("ListUsers query = %v, want page=2 limit=20 last_id=cursor-20", query)
			}
			search := query.GetSearch()
			if search.GetKeyword() != "blek" || len(search.GetFields()) != 0 || search.GetMode() != 0 {
				t.Errorf("ListUsers search = %v, want keyword-only search", search)
			}
			filter := request.GetFilter()
			createdFrom := time.Date(2026, time.July, 31, 17, 0, 0, 0, time.UTC)
			createdTo := time.Date(2026, time.August, 31, 17, 0, 0, 0, time.UTC)
			if filter.GetCreatedFrom() == nil || !filter.GetCreatedFrom().AsTime().Equal(createdFrom) {
				t.Errorf("created_from = %v, want %v", filter.GetCreatedFrom(), createdFrom)
			}
			if filter.GetCreatedTo() == nil || !filter.GetCreatedTo().AsTime().Equal(createdTo) {
				t.Errorf("created_to = %v, want %v", filter.GetCreatedTo(), createdTo)
			}
			if !reflect.DeepEqual(filter.GetRoleCodes(), []uint64{1, 2}) {
				t.Errorf("role codes = %v, want [1 2]", filter.GetRoleCodes())
			}
			return fixtures.ListUsersResponse(), nil
		},
	}
	app := newUsersApp(t, service)
	request := httptest.NewRequest(
		http.MethodGet,
		apiV1Prefix+usershandler.Route+"?page=2&limit=20&last_id=cursor-20"+
			"&created_from=2026-08-01&created_to=2026-08-31"+
			"&role=1,2&keyword=blek",
		nil,
	)
	request.Header.Set(fiber.HeaderAuthorization, authorizationValue)

	// Act
	response, requestErr := app.Test(request)

	// Assert
	body := testutils.ResponseBody(t, response, requestErr)
	if response.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if body != fixtures.SuccessBody {
		t.Errorf("body = %q, want %q", body, fixtures.SuccessBody)
	}
	if !reflect.DeepEqual(calls, []string{"Me", "ListUsers"}) {
		t.Errorf("dependency call order = %v, want [Me ListUsers]", calls)
	}
}

func newTransport(t *testing.T) *grpcclient.Client {
	t.Helper()

	transport, err := grpcclient.New(grpcclientfixtures.ValidConfig())
	if err != nil {
		t.Fatalf("grpcclient.New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := transport.Close(); err != nil {
			t.Errorf("close gRPC transport: %v", err)
		}
	})
	return transport
}

func assertAuthorizationMetadata(t *testing.T, ctx context.Context) {
	t.Helper()

	outgoing, exists := metadata.FromOutgoingContext(ctx)
	if !exists {
		t.Fatal("outgoing metadata is missing")
	}
	actual := outgoing.Get("authorization")
	if len(actual) != 1 || actual[0] != authorizationValue {
		t.Errorf("authorization metadata = %v, want bearer token", actual)
	}
}

func newUsersApp(t *testing.T, service *mocks.Service) *fiber.App {
	t.Helper()
	transport := newTransport(t)
	authClient, err := authservice.NewWithService(transport, service)
	if err != nil {
		t.Fatalf("authservice.NewWithService() error = %v", err)
	}
	return sharedfixtures.NewApp(t, authClient)
}
