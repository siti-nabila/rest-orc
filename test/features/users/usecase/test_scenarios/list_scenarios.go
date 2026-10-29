package test_scenarios

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	usersusecase "github.com/siti-nabila/rest-orc/internal/features/users/usecase"
	"github.com/siti-nabila/rest-orc/pkg/pagination"
	"github.com/siti-nabila/rest-orc/test/features/users/usecase/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

type contextKey string

const requestIDKey contextKey = "request-id"

var errGateway = errors.New("users gateway failed")

func All() []testutils.Scenario {
	return []testutils.Scenario{
		{
			Name: "Execute forwards context and query then returns gateway page",
			Run:  executeReturnsGatewayPage,
		},
		{
			Name: "Execute preserves gateway error identity",
			Run:  executePreservesGatewayError,
		},
		{
			Name: "Execute rejects nil context before gateway call",
			Run:  executeRejectsNilContext,
		},
		{
			Name: "use case constructor rejects nil gateway",
			Run:  constructorRejectsNilGateway,
		},
	}
}

func executeReturnsGatewayPage(t *testing.T) {
	// Arrange
	query := domain.ListQuery{Page: pagination.Query{Page: 2, Limit: 20}}
	expected := pagination.Page[domain.ListItem]{
		Items: []domain.ListItem{{ID: 17, Email: "admin@example.com"}},
		Total: 1,
		Page:  2,
		Limit: 20,
	}
	gateway := &mocks.Gateway{
		ListFunc: func(
			ctx context.Context,
			actualQuery domain.ListQuery,
		) (pagination.Page[domain.ListItem], error) {
			if ctx.Value(requestIDKey) != "request-17" {
				t.Errorf("request ID = %v, want request-17", ctx.Value(requestIDKey))
			}
			if !reflect.DeepEqual(actualQuery, query) {
				t.Errorf("query = %#v, want %#v", actualQuery, query)
			}
			return expected, nil
		},
	}
	usecase := newUseCase(t, gateway)
	ctx := context.WithValue(context.Background(), requestIDKey, "request-17")

	// Act
	actual, err := usecase.Execute(ctx, query)

	// Assert
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Execute() = %#v, want %#v", actual, expected)
	}
	if gateway.Calls != 1 {
		t.Errorf("gateway calls = %d, want 1", gateway.Calls)
	}
}

func executePreservesGatewayError(t *testing.T) {
	// Arrange
	gateway := &mocks.Gateway{
		ListFunc: func(
			context.Context,
			domain.ListQuery,
		) (pagination.Page[domain.ListItem], error) {
			return pagination.Page[domain.ListItem]{}, errGateway
		},
	}
	usecase := newUseCase(t, gateway)

	// Act
	_, err := usecase.Execute(context.Background(), domain.ListQuery{})

	// Assert
	if !errors.Is(err, errGateway) {
		t.Errorf("Execute() error = %v, want gateway error", err)
	}
}

func executeRejectsNilContext(t *testing.T) {
	// Arrange
	gateway := &mocks.Gateway{
		ListFunc: func(
			context.Context,
			domain.ListQuery,
		) (pagination.Page[domain.ListItem], error) {
			t.Fatal("unexpected gateway call")
			return pagination.Page[domain.ListItem]{}, nil
		},
	}
	usecase := newUseCase(t, gateway)

	// Act
	_, err := usecase.Execute(nil, domain.ListQuery{})

	// Assert
	if err == nil {
		t.Fatal("Execute(nil) error = nil, want error")
	}
	if gateway.Calls != 0 {
		t.Errorf("gateway calls = %d, want 0", gateway.Calls)
	}
}

func constructorRejectsNilGateway(t *testing.T) {
	// Act
	_, err := usersusecase.NewList(nil)

	// Assert
	if err == nil {
		t.Fatal("NewList(nil) error = nil, want error")
	}
}

func newUseCase(
	t *testing.T,
	gateway domain.Gateway,
) *usersusecase.List {
	t.Helper()

	usecase, err := usersusecase.NewList(gateway)
	if err != nil {
		t.Fatalf("NewList() error = %v", err)
	}
	return usecase
}
