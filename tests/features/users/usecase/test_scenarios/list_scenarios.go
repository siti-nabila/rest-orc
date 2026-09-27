package test_scenarios

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	usersusecase "github.com/siti-nabila/rest-orc/internal/features/users/usecase"
	"github.com/siti-nabila/rest-orc/pkg/pagination"
	"github.com/siti-nabila/rest-orc/tests/features/users/usecase/mocks"
	"github.com/siti-nabila/rest-orc/tests/shared/testutils"
)

type contextKey string

const requestIDKey contextKey = "request-id"

var errRepository = errors.New("users repository failed")

func All() []testutils.Scenario {
	return []testutils.Scenario{
		{
			Name: "Execute forwards context and query then returns repository page",
			Run:  executeReturnsRepositoryPage,
		},
		{
			Name: "Execute preserves repository error identity",
			Run:  executePreservesRepositoryError,
		},
		{
			Name: "Execute rejects nil context before repository call",
			Run:  executeRejectsNilContext,
		},
		{
			Name: "use case constructor rejects nil repository",
			Run:  constructorRejectsNilRepository,
		},
	}
}

func executeReturnsRepositoryPage(t *testing.T) {
	// Arrange
	query := domain.ListQuery{Page: pagination.Query{Page: 2, Limit: 20}}
	expected := pagination.Page[domain.ListItem]{
		Items: []domain.ListItem{{ID: 17, Email: "admin@example.com"}},
		Total: 1,
		Page:  2,
		Limit: 20,
	}
	repository := &mocks.Repository{
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
	usecase := newUseCase(t, repository)
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
	if repository.Calls != 1 {
		t.Errorf("repository calls = %d, want 1", repository.Calls)
	}
}

func executePreservesRepositoryError(t *testing.T) {
	// Arrange
	repository := &mocks.Repository{
		ListFunc: func(
			context.Context,
			domain.ListQuery,
		) (pagination.Page[domain.ListItem], error) {
			return pagination.Page[domain.ListItem]{}, errRepository
		},
	}
	usecase := newUseCase(t, repository)

	// Act
	_, err := usecase.Execute(context.Background(), domain.ListQuery{})

	// Assert
	if !errors.Is(err, errRepository) {
		t.Errorf("Execute() error = %v, want repository error", err)
	}
}

func executeRejectsNilContext(t *testing.T) {
	// Arrange
	repository := &mocks.Repository{
		ListFunc: func(
			context.Context,
			domain.ListQuery,
		) (pagination.Page[domain.ListItem], error) {
			t.Fatal("unexpected repository call")
			return pagination.Page[domain.ListItem]{}, nil
		},
	}
	usecase := newUseCase(t, repository)

	// Act
	_, err := usecase.Execute(nil, domain.ListQuery{})

	// Assert
	if err == nil {
		t.Fatal("Execute(nil) error = nil, want error")
	}
	if repository.Calls != 0 {
		t.Errorf("repository calls = %d, want 0", repository.Calls)
	}
}

func constructorRejectsNilRepository(t *testing.T) {
	// Act
	_, err := usersusecase.NewList(nil)

	// Assert
	if err == nil {
		t.Fatal("NewList(nil) error = nil, want error")
	}
}

func newUseCase(
	t *testing.T,
	repository domain.Repository,
) *usersusecase.List {
	t.Helper()

	usecase, err := usersusecase.NewList(repository)
	if err != nil {
		t.Fatalf("NewList() error = %v", err)
	}
	return usecase
}
