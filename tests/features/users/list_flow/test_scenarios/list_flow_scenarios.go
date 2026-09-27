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
	requestauth "github.com/siti-nabila/rest-orc/internal/auth"
	"github.com/siti-nabila/rest-orc/internal/client/grpcauth"
	"github.com/siti-nabila/rest-orc/internal/client/grpcclient"
	usershttp "github.com/siti-nabila/rest-orc/internal/features/users/handler/http"
	usersrepo "github.com/siti-nabila/rest-orc/internal/features/users/repo/grpcauth"
	usersusecase "github.com/siti-nabila/rest-orc/internal/features/users/usecase"
	authmiddleware "github.com/siti-nabila/rest-orc/internal/middleware/auth"
	"github.com/siti-nabila/rest-orc/internal/middleware/authorization"
	grpcclientfixtures "github.com/siti-nabila/rest-orc/tests/client/grpcclient/fixtures"
	"github.com/siti-nabila/rest-orc/tests/features/users/list_flow/fixtures"
	"github.com/siti-nabila/rest-orc/tests/features/users/list_flow/mocks"
	"github.com/siti-nabila/rest-orc/tests/shared/testutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

const authorizationValue = "Bearer example-token"
const apiV1Prefix = "/api/v1"

func All() []testutils.Scenario {
	return []testutils.Scenario{
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
	transport := newTransport(t)
	authClient, err := grpcauth.NewWithService(transport, service)
	if err != nil {
		t.Fatalf("grpcauth.NewWithService() error = %v", err)
	}
	authenticator, err := requestauth.NewAuthenticator(authClient)
	if err != nil {
		t.Fatalf("NewAuthenticator() error = %v", err)
	}
	authentication, err := authmiddleware.New(authenticator)
	if err != nil {
		t.Fatalf("auth middleware New() error = %v", err)
	}
	repository, err := usersrepo.New(authClient)
	if err != nil {
		t.Fatalf("users repo New() error = %v", err)
	}
	listUsers, err := usersusecase.NewList(repository)
	if err != nil {
		t.Fatalf("NewList() error = %v", err)
	}
	register, err := usersusecase.NewRegister(repository)
	if err != nil {
		t.Fatalf("NewRegister() error = %v", err)
	}
	login, err := usersusecase.NewLogin(repository)
	if err != nil {
		t.Fatalf("NewLogin() error = %v", err)
	}
	app, writer := testutils.NewHTTPApp()
	handler, err := usershttp.New(listUsers, register, login, writer)
	if err != nil {
		t.Fatalf("users HTTP New() error = %v", err)
	}
	v1 := app.Group(apiV1Prefix)
	handler.RegisterPublicRoutes(v1.Group(usershttp.Route))
	admin := v1.Group(
		usershttp.Route,
		authentication.Handle,
		authorization.RequireRole(requestauth.RoleAdmin),
	)
	handler.RegisterAdminRoutes(admin)
	request := httptest.NewRequest(
		http.MethodGet,
		apiV1Prefix+usershttp.Route+"?page=2&limit=20&last_id=cursor-20"+
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
