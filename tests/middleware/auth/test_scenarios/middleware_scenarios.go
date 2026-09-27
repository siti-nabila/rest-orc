package test_scenarios

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	authmiddleware "github.com/siti-nabila/rest-orc/internal/middleware/auth"
	"github.com/siti-nabila/rest-orc/tests/middleware/auth/mocks"
	"github.com/siti-nabila/rest-orc/tests/shared/testutils"
)

type contextKey string

const authenticatedKey contextKey = "authenticated"

func All() []testutils.Scenario {
	return []testutils.Scenario{
		{
			Name: "middleware forwards authorization and authenticated context",
			Run:  middlewareForwardsAuthenticatedContext,
		},
		{
			Name: "middleware stops pipeline when authentication fails",
			Run:  middlewareStopsOnAuthenticationError,
		},
		{
			Name: "middleware constructor rejects nil authenticator",
			Run:  constructorRejectsNilAuthenticator,
		},
	}
}

func middlewareForwardsAuthenticatedContext(t *testing.T) {
	// Arrange
	authenticator := &mocks.Authenticator{
		AuthenticateFunc: func(
			ctx context.Context,
			authorization string,
		) (context.Context, error) {
			if authorization != "Bearer example-token" {
				t.Errorf("authorization = %q, want bearer token", authorization)
			}
			return context.WithValue(ctx, authenticatedKey, true), nil
		},
	}
	middleware := newMiddleware(t, authenticator)
	app, _ := testutils.NewHTTPApp()
	app.Get("/protected", middleware.Handle, func(ctx fiber.Ctx) error {
		if ctx.Context().Value(authenticatedKey) != true {
			t.Error("authenticated context was not forwarded")
		}
		return ctx.SendStatus(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set(fiber.HeaderAuthorization, "Bearer example-token")

	// Act
	response, err := app.Test(request)

	// Assert
	testutils.ResponseBody(t, response, err)
	if response.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want %d", response.StatusCode, http.StatusNoContent)
	}
	if authenticator.Calls != 1 {
		t.Errorf("Authenticate calls = %d, want 1", authenticator.Calls)
	}
}

func middlewareStopsOnAuthenticationError(t *testing.T) {
	// Arrange
	authenticator := &mocks.Authenticator{
		AuthenticateFunc: func(
			context.Context,
			string,
		) (context.Context, error) {
			return nil, common.ErrUnauthorized
		},
	}
	middleware := newMiddleware(t, authenticator)
	app, _ := testutils.NewHTTPApp()
	nextCalled := false
	app.Get("/protected", middleware.Handle, func(fiber.Ctx) error {
		nextCalled = true
		return nil
	})

	// Act
	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/protected", nil))

	// Assert
	testutils.ResponseBody(t, response, err)
	if response.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", response.StatusCode, http.StatusUnauthorized)
	}
	if nextCalled {
		t.Error("next handler was called after authentication error")
	}
}

func constructorRejectsNilAuthenticator(t *testing.T) {
	// Act
	_, err := authmiddleware.New(nil)

	// Assert
	if err == nil {
		t.Fatal("New(nil) error = nil, want error")
	}
}

func newMiddleware(
	t *testing.T,
	authenticator authmiddleware.Authenticator,
) *authmiddleware.Middleware {
	t.Helper()

	middleware, err := authmiddleware.New(authenticator)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return middleware
}
