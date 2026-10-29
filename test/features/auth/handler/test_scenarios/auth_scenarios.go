package test_scenarios

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	authhandler "github.com/siti-nabila/rest-orc/internal/features/auth/handler"
	"github.com/siti-nabila/rest-orc/internal/response"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/test/features/auth/handler/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

type contextKey string

const requestIDKey contextKey = "request-id"

func registerPublicRouteMapsRequestAndResponse(t *testing.T) {
	// Arrange
	requestContext := context.WithValue(context.Background(), requestIDKey, "auth-request")
	expectedRequest := domain.AuthRequest{
		Email:    "new-user@example.com",
		Password: "secret-password",
	}
	registerUseCase := &mocks.AuthUseCase{
		ExecuteFunc: func(
			ctx context.Context,
			request domain.AuthRequest,
		) (domain.RegisterResponse, error) {
			if ctx != requestContext {
				t.Error("request context was not forwarded unchanged")
			}
			if request != expectedRequest {
				t.Errorf("Register request = %#v, want %#v", request, expectedRequest)
			}
			return domain.RegisterResponse{RegistrationID: 17, RegistrationStatus: "PENDING"}, nil
		},
	}
	app, writer := testutils.NewHTTPApp()
	handler := newAuthHandler(t, registerUseCase, unexpectedLoginUseCase(t), writer)
	app.Use(func(ctx fiber.Ctx) error { ctx.SetContext(requestContext); return ctx.Next() })
	handler.RegisterPublicRoutes(app.Group(authhandler.Route))
	request := httptest.NewRequest(
		http.MethodPost,
		authhandler.Route+authhandler.RegisterRoute,
		strings.NewReader(`{"email":"new-user@example.com","password":"secret-password"}`),
	)
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	request.Header.Set("Origin", fixtures.BrowserOrigin)

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
	requestContext := context.WithValue(context.Background(), requestIDKey, "auth-request")
	expectedRequest := domain.AuthRequest{
		Email:    "user@example.com",
		Password: "secret-password",
	}
	loginUseCase := &mocks.LoginUseCase{
		ExecuteFunc: func(
			ctx context.Context,
			request domain.AuthRequest,
		) (domain.SessionResult, error) {
			if ctx != requestContext {
				t.Error("request context was not forwarded unchanged")
			}
			if request != expectedRequest {
				t.Errorf("Login request = %#v, want %#v", request, expectedRequest)
			}
			return fixtures.LoginResult(), nil
		},
	}
	app, writer := testutils.NewHTTPApp()
	handler := newAuthHandler(t, unexpectedAuthUseCase(t), loginUseCase, writer)
	app.Use(func(ctx fiber.Ctx) error { ctx.SetContext(requestContext); return ctx.Next() })
	handler.RegisterPublicRoutes(app.Group(authhandler.Route))
	request := httptest.NewRequest(
		http.MethodPost,
		authhandler.Route+authhandler.LoginRoute,
		strings.NewReader(`{"email":"user@example.com","password":"secret-password"}`),
	)
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	request.Header.Set("Origin", fixtures.BrowserOrigin)

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
	assertMalformedAuthJSON(t, authhandler.RegisterRoute)
}

func loginRejectsMalformedJSON(t *testing.T) {
	assertMalformedAuthJSON(t, authhandler.LoginRoute)
}

func assertMalformedAuthJSON(t *testing.T, route string) {
	t.Helper()
	registerUseCase := unexpectedAuthUseCase(t)
	loginUseCase := unexpectedLoginUseCase(t)
	app, writer := testutils.NewHTTPApp()
	handler := newAuthHandler(t, registerUseCase, loginUseCase, writer)
	handler.RegisterPublicRoutes(app.Group(authhandler.Route))
	request := httptest.NewRequest(
		http.MethodPost,
		authhandler.Route+route,
		strings.NewReader("{"),
	)
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	request.Header.Set("Origin", fixtures.BrowserOrigin)

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

func constructorRejectsNilAuthUseCases(t *testing.T) {
	_, writer := testutils.NewHTTPApp()
	uc := unexpectedAuthUseCase(t)
	loginUC := unexpectedLoginUseCase(t)
	for _, deps := range []struct {
		register authhandler.RegisterUseCase
		login    authhandler.LoginUseCase
		password authhandler.ChangePasswordUseCase
	}{{nil, loginUC, unexpectedPasswordUseCase(t)}, {uc, nil, unexpectedPasswordUseCase(t)}, {uc, loginUC, nil}} {
		handler, err := authhandler.New(deps.register, deps.login, deps.password, writer, fixtures.NewBrowserPolicy(t))
		if handler != nil || !errors.Is(err, dictionary.ErrAuthHandlerUseCaseRequired) {
			t.Error("nil usecase did not preserve constructor guard identity")
		}
	}
}

func newAuthHandler(t *testing.T, register authhandler.RegisterUseCase, login authhandler.LoginUseCase, writer *response.Writer) *authhandler.Handler {
	t.Helper()
	handler, err := authhandler.New(register, login, unexpectedPasswordUseCase(t), writer, fixtures.NewBrowserPolicy(t))
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func unexpectedAuthUseCase(t *testing.T) *mocks.AuthUseCase {
	t.Helper()
	return &mocks.AuthUseCase{ExecuteFunc: func(context.Context, domain.AuthRequest) (domain.RegisterResponse, error) {
		t.Fatal("unexpected auth usecase call")
		return domain.RegisterResponse{}, nil
	}}
}

func unexpectedPasswordUseCase(t *testing.T) *mocks.PasswordUseCase {
	t.Helper()
	return &mocks.PasswordUseCase{ExecuteFunc: func(context.Context, uint64, domain.ChangePasswordRequest) (domain.ChangePasswordResponse, error) {
		t.Fatal("unexpected password usecase call")
		return domain.ChangePasswordResponse{}, nil
	}}
}

func unexpectedLoginUseCase(t *testing.T) *mocks.LoginUseCase {
	t.Helper()
	return &mocks.LoginUseCase{ExecuteFunc: func(context.Context, domain.AuthRequest) (domain.SessionResult, error) {
		t.Fatal("unexpected Login call")
		return domain.SessionResult{}, nil
	}}
}
