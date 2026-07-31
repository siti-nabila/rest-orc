package test_scenarios

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/api-contracts/pkg/dictionary"
	authdictionary "github.com/siti-nabila/api-contracts/pkg/dictionary/auth"
	commondictionary "github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/api-contracts/pkg/grpcerror"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	appresponse "github.com/siti-nabila/rest-orc/internal/response"
	"github.com/siti-nabila/rest-orc/tests/response/fixtures"
	"github.com/siti-nabila/rest-orc/tests/shared/testutils"
)

func All() []testutils.Scenario {
	return []testutils.Scenario{
		{
			Name: "writes success response from service result",
			Run:  writeSuccess,
		},
		{
			Name: "maps registered grpc error using request language",
			Run:  mapRegisteredGRPCError,
		},
		{
			Name: "maps grpc bad request field details",
			Run:  mapBadRequest,
		},
		{
			Name: "hides unknown error as internal response",
			Run:  hideUnknownError,
		},
	}
}

func writeSuccess(t *testing.T) {
	writer := newWriter()
	app := fiber.New()
	app.Get("/profile", func(ctx fiber.Ctx) error {
		return writer.Write(ctx, appresponse.Result{
			Success: appresponse.Success{
				Message: "User profile retrieved successfully.",
			},
			Data: fixtures.UserProfile,
		})
	})

	response := performRequest(t, app, "/profile", "")
	assertResponse(t, response, http.StatusOK, fixtures.SuccessBody)
}

func mapRegisteredGRPCError(t *testing.T) {
	writer := newWriter()
	app := fiber.New()
	app.Get("/profile", func(ctx fiber.Ctx) error {
		return writer.Write(ctx, appresponse.Result{
			Err: grpcerror.Encode(authdictionary.ErrNotFound, locale.English),
		})
	})

	response := performRequest(t, app, "/profile", "id-ID")
	assertResponse(
		t,
		response,
		http.StatusNotFound,
		fixtures.NotFoundIndonesianBody,
	)
}

func mapBadRequest(t *testing.T) {
	writer := newWriter()
	fieldErrors := dictionary.FieldErrors{}
	fieldErrors.Add("email", authdictionary.ErrRequired)

	app := fiber.New()
	app.Get("/profile", func(ctx fiber.Ctx) error {
		return writer.Write(ctx, appresponse.Result{
			Err: grpcerror.Encode(fieldErrors, locale.Indonesian),
		})
	})

	response := performRequest(t, app, "/profile", "id")
	assertResponse(
		t,
		response,
		http.StatusBadRequest,
		fixtures.BadRequestIndonesianBody,
	)
}

func hideUnknownError(t *testing.T) {
	writer := newWriter()
	app := fiber.New()
	app.Get("/profile", func(ctx fiber.Ctx) error {
		return writer.Write(ctx, appresponse.Result{
			Err: errors.New("database password leaked"),
		})
	})

	response := performRequest(t, app, "/profile", "en")
	assertResponse(
		t,
		response,
		http.StatusInternalServerError,
		fixtures.InternalErrorBody,
	)
}

func newWriter() *appresponse.Writer {
	return appresponse.NewWriter(appresponse.NewErrorMapper(
		authdictionary.Registry(),
		commondictionary.Registry(),
	))
}

func performRequest(
	t *testing.T,
	app *fiber.App,
	path string,
	language string,
) *http.Response {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	if language != "" {
		request.Header.Set(locale.HTTPHeader, language)
	}
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	return response
}

func assertResponse(
	t *testing.T,
	response *http.Response,
	status int,
	body string,
) {
	t.Helper()
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close response body: %v", err)
		}
	}()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if response.StatusCode != status {
		t.Errorf("status = %d, want %d", response.StatusCode, status)
	}
	if string(responseBody) != body {
		t.Errorf("body = %q, want %q", responseBody, body)
	}
}
