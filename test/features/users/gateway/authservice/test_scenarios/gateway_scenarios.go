package test_scenarios

import (
	"context"
	"errors"
	"reflect"
	"testing"

	paginatorv1 "github.com/siti-nabila/api-contracts/pb/paginator/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	usersgateway "github.com/siti-nabila/rest-orc/internal/features/users/gateway/authservice"
	"github.com/siti-nabila/rest-orc/pkg/pagination"
	"github.com/siti-nabila/rest-orc/test/features/users/gateway/authservice/fixtures"
	"github.com/siti-nabila/rest-orc/test/features/users/gateway/authservice/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/protobuf/proto"
)

type contextKey string

const requestIDKey contextKey = "request-id"

var errDependency = errors.New("ListUsers dependency failed")

func All() []testutils.Scenario {
	return []testutils.Scenario{
		{
			Name: "List maps query, propagates context, and maps complete paginator",
			Run:  listMapsRequestAndResponse,
		},
		{
			Name: "List preserves empty items as a JSON array",
			Run:  listPreservesEmptyItems,
		},
		{
			Name: "List maps unknown search mode to unspecified",
			Run:  listMapsUnknownSearchModeToUnspecified,
		},
		{
			Name: "List preserves dependency error identity",
			Run:  listPreservesDependencyError,
		},
		{
			Name: "List rejects nil dependency response",
			Run:  listRejectsNilResponse,
		},
		{
			Name: "List rejects malformed nil response item",
			Run:  listRejectsNilItem,
		},
		{
			Name: "List rejects nil context before dependency call",
			Run:  listRejectsNilContext,
		},
		{
			Name: "gateway constructor rejects nil client",
			Run:  constructorRejectsNilClient,
		},
	}
}

func listMapsRequestAndResponse(t *testing.T) {
	// Arrange
	requestContext := context.WithValue(
		context.Background(),
		requestIDKey,
		"request-17",
	)
	client := &mocks.Client{
		ListUsersFunc: func(
			ctx context.Context,
			request *userv1.ListUsersRequest,
		) (*userv1.ListUsersResponse, error) {
			if ctx.Value(requestIDKey) != "request-17" {
				t.Errorf("request context value = %v, want request-17", ctx.Value(requestIDKey))
			}
			if !proto.Equal(request, fixtures.ExpectedRequest()) {
				t.Errorf("ListUsers request = %v, want %v", request, fixtures.ExpectedRequest())
			}
			return fixtures.Response(), nil
		},
	}
	gateway := newGateway(t, client)

	// Act
	actual, err := gateway.List(requestContext, fixtures.Query())

	// Assert
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if !reflect.DeepEqual(actual, fixtures.ExpectedPage()) {
		t.Errorf("List() = %#v, want %#v", actual, fixtures.ExpectedPage())
	}
	if client.Calls != 1 {
		t.Errorf("ListUsers calls = %d, want 1", client.Calls)
	}
}

func listPreservesEmptyItems(t *testing.T) {
	// Arrange
	client := &mocks.Client{
		ListUsersFunc: func(
			context.Context,
			*userv1.ListUsersRequest,
		) (*userv1.ListUsersResponse, error) {
			return &userv1.ListUsersResponse{
				Items:      nil,
				Page:       1,
				Limit:      10,
				NextCursor: "",
			}, nil
		},
	}
	gateway := newGateway(t, client)

	// Act
	result, err := gateway.List(context.Background(), fixtures.Query())

	// Assert
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if result.Items == nil {
		t.Fatal("List() items = nil, want non-nil empty slice")
	}
	if len(result.Items) != 0 {
		t.Errorf("List() items length = %d, want 0", len(result.Items))
	}
}

func listMapsUnknownSearchModeToUnspecified(t *testing.T) {
	// Arrange
	query := fixtures.Query()
	query.Page.Search.Mode = pagination.SearchMode(99)
	client := &mocks.Client{
		ListUsersFunc: func(
			_ context.Context,
			request *userv1.ListUsersRequest,
		) (*userv1.ListUsersResponse, error) {
			if actual := request.GetQuery().GetSearch().GetMode(); actual != paginatorv1.SearchMode_SEARCH_MODE_UNSPECIFIED {
				t.Errorf("search mode = %s, want unspecified", actual)
			}
			return &userv1.ListUsersResponse{}, nil
		},
	}
	gateway := newGateway(t, client)

	// Act
	_, err := gateway.List(context.Background(), query)

	// Assert
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
}

func listPreservesDependencyError(t *testing.T) {
	// Arrange
	client := &mocks.Client{
		ListUsersFunc: func(
			context.Context,
			*userv1.ListUsersRequest,
		) (*userv1.ListUsersResponse, error) {
			return nil, errDependency
		},
	}
	gateway := newGateway(t, client)

	// Act
	_, err := gateway.List(context.Background(), fixtures.Query())

	// Assert
	if !errors.Is(err, errDependency) {
		t.Errorf("List() error = %v, want dependency error", err)
	}
}

func listRejectsNilResponse(t *testing.T) {
	// Arrange
	client := &mocks.Client{
		ListUsersFunc: func(
			context.Context,
			*userv1.ListUsersRequest,
		) (*userv1.ListUsersResponse, error) {
			return nil, nil
		},
	}
	gateway := newGateway(t, client)

	// Act
	_, err := gateway.List(context.Background(), fixtures.Query())

	// Assert
	if err == nil {
		t.Fatal("List() error = nil, want nil response error")
	}
}

func listRejectsNilItem(t *testing.T) {
	// Arrange
	client := &mocks.Client{
		ListUsersFunc: func(
			context.Context,
			*userv1.ListUsersRequest,
		) (*userv1.ListUsersResponse, error) {
			return &userv1.ListUsersResponse{
				Items: []*userv1.UserListItem{nil},
			}, nil
		},
	}
	gateway := newGateway(t, client)

	// Act
	_, err := gateway.List(context.Background(), fixtures.Query())

	// Assert
	if err == nil {
		t.Fatal("List() error = nil, want malformed item error")
	}
}

func listRejectsNilContext(t *testing.T) {
	// Arrange
	client := &mocks.Client{
		ListUsersFunc: func(
			context.Context,
			*userv1.ListUsersRequest,
		) (*userv1.ListUsersResponse, error) {
			t.Fatal("unexpected ListUsers dependency call")
			return nil, nil
		},
	}
	gateway := newGateway(t, client)

	// Act
	_, err := gateway.List(nil, fixtures.Query())

	// Assert
	if err == nil {
		t.Fatal("List(nil, query) error = nil, want error")
	}
	if client.Calls != 0 {
		t.Errorf("ListUsers calls = %d, want 0", client.Calls)
	}
}

func constructorRejectsNilClient(t *testing.T) {
	// Act
	_, err := usersgateway.New(nil)

	// Assert
	if err == nil {
		t.Fatal("New(nil) error = nil, want error")
	}
}

func newGateway(t *testing.T, client usersgateway.Client) *usersgateway.Gateway {
	t.Helper()

	gateway, err := usersgateway.New(client)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return gateway
}
