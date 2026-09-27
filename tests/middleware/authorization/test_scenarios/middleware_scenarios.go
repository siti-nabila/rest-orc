package test_scenarios

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	requestauth "github.com/siti-nabila/rest-orc/internal/auth"
	"github.com/siti-nabila/rest-orc/internal/middleware/authorization"
	"github.com/siti-nabila/rest-orc/tests/shared/testutils"
)

func All() []testutils.Scenario {
	return []testutils.Scenario{
		{
			Name: "admin principal is authorized",
			Run:  adminPrincipalIsAuthorized,
		},
		{
			Name: "non-admin principal is forbidden",
			Run:  nonAdminPrincipalIsForbidden,
		},
		{
			Name: "missing principal is unauthorized",
			Run:  missingPrincipalIsUnauthorized,
		},
	}
}

func adminPrincipalIsAuthorized(t *testing.T) {
	response := performRequest(t, &requestauth.Principal{RoleNames: []string{"Admin"}})
	if response.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want %d", response.StatusCode, http.StatusNoContent)
	}
}

func nonAdminPrincipalIsForbidden(t *testing.T) {
	response := performRequest(t, &requestauth.Principal{RoleNames: []string{"member"}})
	if response.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want %d", response.StatusCode, http.StatusForbidden)
	}
}

func missingPrincipalIsUnauthorized(t *testing.T) {
	response := performRequest(t, nil)
	if response.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", response.StatusCode, http.StatusUnauthorized)
	}
}

func performRequest(t *testing.T, principal *requestauth.Principal) *http.Response {
	t.Helper()

	app, _ := testutils.NewHTTPApp()
	if principal != nil {
		app.Use(func(ctx fiber.Ctx) error {
			ctx.SetContext(requestauth.WithPrincipal(ctx.Context(), *principal))
			return ctx.Next()
		})
	}
	app.Get(
		"/admin",
		authorization.RequireRole(requestauth.RoleAdmin),
		func(ctx fiber.Ctx) error {
			return ctx.SendStatus(http.StatusNoContent)
		},
	)
	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/admin", nil))
	testutils.ResponseBody(t, response, err)
	return response
}
