package test_scenarios

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/rest-orc/test/app/fixtures"
	"google.golang.org/grpc/connectivity"
)

func passwordRouteRegistration(t *testing.T) {
	application := newApplication(t, fixtures.ValidConfig())
	found := 0
	for _, route := range application.HTTPServer.GetRoutes(true) {
		if route.Method != http.MethodPost {
			continue
		}
		switch route.Path {
		case "/api/v1/me/password":
			found++
		case "/me/password", "/api/v1/users/password", "/api/v1/users/me/password", "/api/v1/auth/password":
			t.Error("unexpected password alias")
		}
	}
	if found != 1 {
		t.Fatal("password route is missing or duplicated")
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/me/password", strings.NewReader(`{"current_password":"legacy password","new_password":"new-password"}`))
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	response, err := application.HTTPServer.Test(request)
	readResponseBody(t, response, err)
	if response.StatusCode != 401 || len(response.Header.Values(fiber.HeaderSetCookie)) != 0 {
		t.Fatal("missing bearer changed auth or cookie policy")
	}
	state := application.AuthConnection.Connection().(interface{ GetState() connectivity.State })
	if state.GetState() != connectivity.Idle {
		t.Fatal("missing bearer dispatched an RPC")
	}
}
