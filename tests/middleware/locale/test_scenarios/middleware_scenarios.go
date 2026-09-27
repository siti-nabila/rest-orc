package test_scenarios

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	contractlocale "github.com/siti-nabila/api-contracts/pkg/locale"
	localemiddleware "github.com/siti-nabila/rest-orc/internal/middleware/locale"
	"github.com/siti-nabila/rest-orc/tests/shared/testutils"
	"google.golang.org/grpc/metadata"
)

func All() []testutils.Scenario {
	return []testutils.Scenario{
		{
			Name: "forwards arbitrary request language and preserves metadata",
			Run:  forwardArbitraryLanguage,
		},
		{
			Name: "forwards default language for malformed input",
			Run:  forwardDefaultLanguage,
		},
	}
}

func forwardArbitraryLanguage(t *testing.T) {
	app, _ := testutils.NewHTTPApp()
	app.Use(func(ctx fiber.Ctx) error {
		ctx.SetContext(metadata.AppendToOutgoingContext(
			ctx.Context(),
			"x-request-id",
			"request-17",
			contractlocale.MetadataKey,
			"en",
		))
		return ctx.Next()
	})
	app.Get("/localized", localemiddleware.Handle, func(ctx fiber.Ctx) error {
		outgoing, exists := metadata.FromOutgoingContext(ctx.Context())
		if !exists {
			t.Fatal("outgoing metadata is missing")
		}
		assertMetadata(t, outgoing, contractlocale.MetadataKey, "zh-CN")
		assertMetadata(t, outgoing, "x-request-id", "request-17")
		return ctx.SendStatus(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodGet, "/localized", nil)
	request.Header.Set(contractlocale.HTTPHeader, "zh-CN,zh;q=0.9")

	response, err := app.Test(request)
	testutils.ResponseBody(t, response, err)
	if response.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want %d", response.StatusCode, http.StatusNoContent)
	}
}

func forwardDefaultLanguage(t *testing.T) {
	app, _ := testutils.NewHTTPApp()
	app.Get("/localized", localemiddleware.Handle, func(ctx fiber.Ctx) error {
		outgoing, exists := metadata.FromOutgoingContext(ctx.Context())
		if !exists {
			t.Fatal("outgoing metadata is missing")
		}
		assertMetadata(
			t,
			outgoing,
			contractlocale.MetadataKey,
			string(contractlocale.DefaultLanguage),
		)
		return ctx.SendStatus(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodGet, "/localized", nil)
	request.Header.Set(contractlocale.HTTPHeader, "invalid@locale")

	response, err := app.Test(request)
	testutils.ResponseBody(t, response, err)
	if response.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want %d", response.StatusCode, http.StatusNoContent)
	}
}

func assertMetadata(t *testing.T, values metadata.MD, key, expected string) {
	t.Helper()
	actual := values.Get(key)
	if len(actual) != 1 || actual[0] != expected {
		t.Errorf("metadata %s = %v, want %q", key, actual, expected)
	}
}
