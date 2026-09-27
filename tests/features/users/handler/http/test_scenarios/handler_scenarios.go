package test_scenarios

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	usershttp "github.com/siti-nabila/rest-orc/internal/features/users/handler/http"
	"github.com/siti-nabila/rest-orc/internal/response"
	"github.com/siti-nabila/rest-orc/pkg/pagination"
	"github.com/siti-nabila/rest-orc/tests/features/users/handler/http/fixtures"
	"github.com/siti-nabila/rest-orc/tests/features/users/handler/http/mocks"
	"github.com/siti-nabila/rest-orc/tests/shared/testutils"
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
			Name: "Register public route maps request and response",
			Run:  registerPublicRouteMapsRequestAndResponse,
		},
		{
			Name: "Login public route maps request and response",
			Run:  loginPublicRouteMapsRequestAndResponse,
		},
		{
			Name: "Register rejects malformed JSON before use case call",
			Run:  registerRejectsMalformedJSON,
		},
		{
			Name: "Login rejects malformed JSON before use case call",
			Run:  loginRejectsMalformedJSON,
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
	handler.RegisterAdminRoutes(app.Group(usershttp.Route))
	request := httptest.NewRequest(
		http.MethodGet,
		usershttp.Route+"?page=2&limit=20&last_id=cursor-20"+
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
	handler.RegisterAdminRoutes(app.Group(usershttp.Route))

	// Act
	response, err := app.Test(httptest.NewRequest(http.MethodGet, usershttp.Route, nil))

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
	handler.RegisterAdminRoutes(app.Group(usershttp.Route))

	// Act
	response, err := app.Test(httptest.NewRequest(http.MethodGet, usershttp.Route, nil))

	// Assert
	testutils.ResponseBody(t, response, err)
	if response.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want %d", response.StatusCode, http.StatusForbidden)
	}
}

func registerPublicRouteMapsRequestAndResponse(t *testing.T) {
	// Arrange
	expectedRequest := domain.AuthRequest{
		Email:    "new-user@example.com",
		Password: "secret-password",
	}
	registerUseCase := &mocks.AuthUseCase{
		ExecuteFunc: func(
			_ context.Context,
			request domain.AuthRequest,
		) (domain.AuthResponse, error) {
			if request != expectedRequest {
				t.Errorf("Register request = %#v, want %#v", request, expectedRequest)
			}
			return domain.AuthResponse{Token: "register-token"}, nil
		},
	}
	app, writer := testutils.NewHTTPApp()
	handler := newAuthHandler(t, registerUseCase, unexpectedAuthUseCase(t), writer)
	handler.RegisterPublicRoutes(app.Group(usershttp.Route))
	request := httptest.NewRequest(
		http.MethodPost,
		usershttp.Route+usershttp.RegisterRoute,
		strings.NewReader(`{"email":"new-user@example.com","password":"secret-password"}`),
	)
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

	// Act
	response, err := app.Test(request)

	// Assert
	body := testutils.ResponseBody(t, response, err)
	if response.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if body != fixtures.RegisterSuccessBody {
		t.Errorf("body = %q, want %q", body, fixtures.RegisterSuccessBody)
	}
	if registerUseCase.Calls != 1 {
		t.Errorf("Register Execute calls = %d, want 1", registerUseCase.Calls)
	}
}

func loginPublicRouteMapsRequestAndResponse(t *testing.T) {
	// Arrange
	expectedRequest := domain.AuthRequest{
		Email:    "user@example.com",
		Password: "secret-password",
	}
	loginUseCase := &mocks.AuthUseCase{
		ExecuteFunc: func(
			_ context.Context,
			request domain.AuthRequest,
		) (domain.AuthResponse, error) {
			if request != expectedRequest {
				t.Errorf("Login request = %#v, want %#v", request, expectedRequest)
			}
			return domain.AuthResponse{Token: "login-token"}, nil
		},
	}
	app, writer := testutils.NewHTTPApp()
	handler := newAuthHandler(t, unexpectedAuthUseCase(t), loginUseCase, writer)
	handler.RegisterPublicRoutes(app.Group(usershttp.Route))
	request := httptest.NewRequest(
		http.MethodPost,
		usershttp.Route+usershttp.LoginRoute,
		strings.NewReader(`{"email":"user@example.com","password":"secret-password"}`),
	)
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

	// Act
	response, err := app.Test(request)

	// Assert
	body := testutils.ResponseBody(t, response, err)
	if response.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if body != fixtures.LoginSuccessBody {
		t.Errorf("body = %q, want %q", body, fixtures.LoginSuccessBody)
	}
	if loginUseCase.Calls != 1 {
		t.Errorf("Login Execute calls = %d, want 1", loginUseCase.Calls)
	}
}

func registerRejectsMalformedJSON(t *testing.T) {
	assertMalformedAuthJSON(t, usershttp.RegisterRoute)
}

func loginRejectsMalformedJSON(t *testing.T) {
	assertMalformedAuthJSON(t, usershttp.LoginRoute)
}

func assertMalformedAuthJSON(t *testing.T, route string) {
	t.Helper()
	registerUseCase := unexpectedAuthUseCase(t)
	loginUseCase := unexpectedAuthUseCase(t)
	app, writer := testutils.NewHTTPApp()
	handler := newAuthHandler(t, registerUseCase, loginUseCase, writer)
	handler.RegisterPublicRoutes(app.Group(usershttp.Route))
	request := httptest.NewRequest(
		http.MethodPost,
		usershttp.Route+route,
		strings.NewReader("{"),
	)
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

	response, err := app.Test(request)
	testutils.ResponseBody(t, response, err)
	if response.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", response.StatusCode, http.StatusBadRequest)
	}
	if registerUseCase.Calls != 0 || loginUseCase.Calls != 0 {
		t.Errorf(
			"auth use case calls = register:%d login:%d, want zero",
			registerUseCase.Calls,
			loginUseCase.Calls,
		)
	}
}

func constructorRejectsNilDependencies(t *testing.T) {
	// Arrange
	listUseCase := &mocks.ListUseCase{}
	registerUseCase := &mocks.AuthUseCase{}
	loginUseCase := &mocks.AuthUseCase{}
	_, writer := testutils.NewHTTPApp()

	// Act
	_, listErr := usershttp.New(nil, registerUseCase, loginUseCase, writer)
	_, registerErr := usershttp.New(listUseCase, nil, loginUseCase, writer)
	_, loginErr := usershttp.New(listUseCase, registerUseCase, nil, writer)
	_, writerErr := usershttp.New(
		listUseCase,
		registerUseCase,
		loginUseCase,
		nil,
	)

	// Assert
	if listErr == nil {
		t.Error("New(nil, register, login, writer) error = nil, want error")
	}
	if registerErr == nil {
		t.Error("New(list, nil, login, writer) error = nil, want error")
	}
	if loginErr == nil {
		t.Error("New(list, register, nil, writer) error = nil, want error")
	}
	if writerErr == nil {
		t.Error("New(list, register, login, nil) error = nil, want error")
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
	handler.RegisterAdminRoutes(app.Group(usershttp.Route))
	response, err := app.Test(httptest.NewRequest(
		http.MethodGet,
		usershttp.Route+query,
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

func newHandler(
	t *testing.T,
	usecase usershttp.ListUseCase,
	writer *response.Writer,
) *usershttp.Handler {
	t.Helper()

	unexpectedAuthCall := func(
		context.Context,
		domain.AuthRequest,
	) (domain.AuthResponse, error) {
		t.Fatal("unexpected auth use case call")
		return domain.AuthResponse{}, nil
	}
	handler, err := usershttp.New(
		usecase,
		&mocks.AuthUseCase{ExecuteFunc: unexpectedAuthCall},
		&mocks.AuthUseCase{ExecuteFunc: unexpectedAuthCall},
		writer,
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return handler
}

func newAuthHandler(
	t *testing.T,
	registerUseCase usershttp.RegisterUseCase,
	loginUseCase usershttp.LoginUseCase,
	writer *response.Writer,
) *usershttp.Handler {
	t.Helper()

	listUseCase := &mocks.ListUseCase{
		ExecuteFunc: func(
			context.Context,
			domain.ListQuery,
		) (pagination.Page[domain.ListItem], error) {
			t.Fatal("unexpected List use case call")
			return pagination.Page[domain.ListItem]{}, nil
		},
	}
	handler, err := usershttp.New(
		listUseCase,
		registerUseCase,
		loginUseCase,
		writer,
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return handler
}

func unexpectedAuthUseCase(t *testing.T) *mocks.AuthUseCase {
	t.Helper()
	return &mocks.AuthUseCase{
		ExecuteFunc: func(
			context.Context,
			domain.AuthRequest,
		) (domain.AuthResponse, error) {
			t.Fatal("unexpected auth use case call")
			return domain.AuthResponse{}, nil
		},
	}
}
