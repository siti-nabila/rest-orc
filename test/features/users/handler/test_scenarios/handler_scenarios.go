package test_scenarios

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	usershandler "github.com/siti-nabila/rest-orc/internal/features/users/handler"
	"github.com/siti-nabila/rest-orc/internal/response"
	"github.com/siti-nabila/rest-orc/pkg/pagination"
	"github.com/siti-nabila/rest-orc/test/features/users/handler/fixtures"
	"github.com/siti-nabila/rest-orc/test/features/users/handler/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

type contextKey string

const requestIDKey contextKey = "request-id"

func All() []testutils.Scenario {
	return []testutils.Scenario{
		{
			Name: "List maps query and writes complete paginator response",
			Run:  listMapsQueryAndResponse,
		},
		{
			Name: "List leaves omitted pagination query at zero",
			Run:  listLeavesOmittedQueryAtZero,
		},
		{
			Name: "List rejects malformed page before use case call",
			Run:  listRejectsMalformedPage,
		},
		{
			Name: "List rejects non-positive limit before use case call",
			Run:  listRejectsNonPositiveLimit,
		},
		{
			Name: "List rejects malformed created_from before use case call",
			Run:  listRejectsMalformedCreatedFrom,
		},
		{
			Name: "List rejects created_from after created_to before use case call",
			Run:  listRejectsInvertedCreatedRange,
		},
		{
			Name: "List rejects malformed role before use case call",
			Run:  listRejectsMalformedRole,
		},
		{
			Name: "List maps use case authorization error",
			Run:  listMapsUseCaseError,
		},
		{
			Name: "handler constructor rejects nil dependencies",
			Run:  constructorRejectsNilDependencies,
		},
	}
}

func listMapsQueryAndResponse(t *testing.T) {
	// Arrange
	jakarta := time.FixedZone("Asia/Jakarta", 7*60*60)
	createdFrom := time.Date(2026, time.August, 1, 0, 0, 0, 0, jakarta)
	createdTo := time.Date(2026, time.September, 1, 0, 0, 0, 0, jakarta)
	expectedQuery := domain.ListQuery{
		Page: pagination.Query{
			Page:   2,
			Limit:  20,
			Search: &pagination.Search{Keyword: "blek"},
			LastID: "cursor-20",
		},
		Filter: domain.ListFilter{
			CreatedFrom: &createdFrom,
			CreatedTo:   &createdTo,
			RoleCodes:   []uint64{1, 2},
		},
	}
	usecase := &mocks.ListUseCase{
		ExecuteFunc: func(
			ctx context.Context,
			query domain.ListQuery,
		) (pagination.Page[domain.ListItem], error) {
			if ctx.Value(requestIDKey) != "request-17" {
				t.Errorf("request ID = %v, want request-17", ctx.Value(requestIDKey))
			}
			if !reflect.DeepEqual(query, expectedQuery) {
				t.Errorf("query = %#v, want %#v", query, expectedQuery)
			}
			return fixtures.Page(), nil
		},
	}
	app, writer := testutils.NewHTTPApp()
	handler := newHandler(t, usecase, writer)
	app.Use(func(ctx fiber.Ctx) error {
		ctx.SetContext(context.WithValue(ctx.Context(), requestIDKey, "request-17"))
		return ctx.Next()
	})
	handler.RegisterAdminRoutes(app.Group(usershandler.Route))
	request := httptest.NewRequest(
		http.MethodGet,
		usershandler.Route+"?page=2&limit=20&last_id=cursor-20"+
			"&created_from=2026-08-01&created_to=2026-08-31"+
			"&role=1,2&keyword=%20blek%20",
		nil,
	)

	// Act
	response, err := app.Test(request)

	// Assert
	body := testutils.ResponseBody(t, response, err)
	if response.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if body != fixtures.SuccessBody {
		t.Errorf("body = %q, want %q", body, fixtures.SuccessBody)
	}
	if usecase.Calls != 1 {
		t.Errorf("Execute calls = %d, want 1", usecase.Calls)
	}
}

func listLeavesOmittedQueryAtZero(t *testing.T) {
	// Arrange
	usecase := &mocks.ListUseCase{
		ExecuteFunc: func(
			_ context.Context,
			query domain.ListQuery,
		) (pagination.Page[domain.ListItem], error) {
			if query.Page.Page != 0 || query.Page.Limit != 0 || query.Page.LastID != "" {
				t.Errorf("omitted query = %#v, want zero values", query)
			}
			return fixtures.Page(), nil
		},
	}
	app, writer := testutils.NewHTTPApp()
	handler := newHandler(t, usecase, writer)
	handler.RegisterAdminRoutes(app.Group(usershandler.Route))

	// Act
	response, err := app.Test(httptest.NewRequest(http.MethodGet, usershandler.Route, nil))

	// Assert
	testutils.ResponseBody(t, response, err)
	if response.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
}

func listRejectsMalformedPage(t *testing.T) {
	assertInvalidQuery(t, "?page=invalid", http.StatusBadRequest)
}

func listRejectsNonPositiveLimit(t *testing.T) {
	assertInvalidQuery(t, "?limit=0", http.StatusBadRequest)
}

func listRejectsMalformedCreatedFrom(t *testing.T) {
	assertInvalidQuery(t, "?created_from=01-08-2026", http.StatusBadRequest)
}

func listRejectsInvertedCreatedRange(t *testing.T) {
	assertInvalidQuery(
		t,
		"?created_from=2026-08-02&created_to=2026-08-01",
		http.StatusBadRequest,
	)
}

func listRejectsMalformedRole(t *testing.T) {
	assertInvalidQuery(t, "?role=1,invalid", http.StatusBadRequest)
}

func listMapsUseCaseError(t *testing.T) {
	// Arrange
	usecase := &mocks.ListUseCase{
		ExecuteFunc: func(
			context.Context,
			domain.ListQuery,
		) (pagination.Page[domain.ListItem], error) {
			return pagination.Page[domain.ListItem]{}, common.ErrForbidden
		},
	}
	app, writer := testutils.NewHTTPApp()
	handler := newHandler(t, usecase, writer)
	handler.RegisterAdminRoutes(app.Group(usershandler.Route))

	// Act
	response, err := app.Test(httptest.NewRequest(http.MethodGet, usershandler.Route, nil))

	// Assert
	testutils.ResponseBody(t, response, err)
	if response.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want %d", response.StatusCode, http.StatusForbidden)
	}
}

func constructorRejectsNilDependencies(t *testing.T) {
	_, writer := testutils.NewHTTPApp()
	if _, err := usershandler.New(&mocks.ListUseCase{}, nil, writer); err == nil {
		t.Error("nil profile usecase accepted")
	}
	if _, err := usershandler.New(nil, &mocks.ProfileUseCase{}, writer); err == nil {
		t.Error("nil list usecase accepted")
	}
	if _, err := usershandler.New(&mocks.ListUseCase{}, &mocks.ProfileUseCase{}, nil); err == nil {
		t.Error("nil response writer accepted")
	}
}

func assertInvalidQuery(t *testing.T, query string, expectedStatus int) {
	t.Helper()

	usecase := &mocks.ListUseCase{
		ExecuteFunc: func(
			context.Context,
			domain.ListQuery,
		) (pagination.Page[domain.ListItem], error) {
			t.Fatal("unexpected use case call")
			return pagination.Page[domain.ListItem]{}, nil
		},
	}
	app, writer := testutils.NewHTTPApp()
	handler := newHandler(t, usecase, writer)
	handler.RegisterAdminRoutes(app.Group(usershandler.Route))
	response, err := app.Test(httptest.NewRequest(
		http.MethodGet,
		usershandler.Route+query,
		nil,
	))
	testutils.ResponseBody(t, response, err)
	if response.StatusCode != expectedStatus {
		t.Errorf("status = %d, want %d", response.StatusCode, expectedStatus)
	}
	if usecase.Calls != 0 {
		t.Errorf("Execute calls = %d, want 0", usecase.Calls)
	}
}

func newHandler(t *testing.T, usecase usershandler.ListUseCase, writer *response.Writer) *usershandler.Handler {
	t.Helper()
	handler, err := usershandler.New(usecase, &mocks.ProfileUseCase{ExecuteFunc: func(context.Context, domain.ProfileActor, domain.UpdateProfileRequest) (domain.UpdateProfileResponse, error) {
		t.Fatal("unexpected profile usecase")
		return domain.UpdateProfileResponse{}, nil
	}}, writer)
	if err != nil {
		t.Fatal(err)
	}
	return handler
}
