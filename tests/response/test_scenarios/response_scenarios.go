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
	grpcmapping "github.com/siti-nabila/api-contracts/pkg/grpcerror/mapping"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	errorpackage "github.com/siti-nabila/error-package"
	appresponse "github.com/siti-nabila/rest-orc/internal/response"
	"github.com/siti-nabila/rest-orc/tests/response/fixtures"
	"github.com/siti-nabila/rest-orc/tests/shared/testutils"
	"google.golang.org/grpc/codes"
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
			Name: "maps registered grpc error using arbitrary request language",
			Run:  mapArbitraryLanguageGRPCError,
		},
		{
			Name: "maps registered error by HTTP contract independently of grpc code",
			Run:  mapRegisteredErrorIndependentlyOfGRPCCode,
		},
		{
			Name: "maps common deadline exceeded to HTTP gateway timeout",
			Run:  mapCommonDeadlineExceeded,
		},
		{
			Name: "maps missing HTTP endpoint separately from missing data",
			Run:  mapMissingHTTPEndpoint,
		},
		{
			Name: "maps grpc bad request field details",
			Run:  mapBadRequest,
		},
		{
			Name: "preserves multiple localized grpc field errors",
			Run:  mapArbitraryLanguageFieldErrors,
		},
		{
			Name: "maps error-package errors to HTTP bad request",
			Run:  mapErrorPackageErrors,
		},
		{
			Name: "maps encoded unknown grpc error to HTTP internal error",
			Run:  mapEncodedUnknownError,
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
			Err: newErrorEncoder().Encode(authdictionary.ErrNotFound, locale.English),
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

func mapArbitraryLanguageGRPCError(t *testing.T) {
	writer := newWriter()
	app := fiber.New()
	app.Get("/profile", func(ctx fiber.Ctx) error {
		return writer.Write(ctx, appresponse.Result{
			Err: newErrorEncoder().Encode(authdictionary.ErrDataExists, locale.Chinese),
		})
	})

	response := performRequest(t, app, "/profile", "zh-CN")
	assertResponse(
		t,
		response,
		http.StatusConflict,
		fixtures.AlreadyExistsChineseBody,
	)
}

func mapCommonDeadlineExceeded(t *testing.T) {
	writer := newWriter()
	app := fiber.New()
	app.Get("/profile", func(ctx fiber.Ctx) error {
		return writer.Write(ctx, appresponse.Result{
			Err: newErrorEncoder().Encode(
				commondictionary.ErrDeadlineExceeded,
				locale.English,
			),
		})
	})

	response := performRequest(t, app, "/profile", "en")
	assertResponse(
		t,
		response,
		http.StatusGatewayTimeout,
		fixtures.DeadlineExceededBody,
	)
}

func mapRegisteredErrorIndependentlyOfGRPCCode(t *testing.T) {
	writer := newWriter()
	encoder := grpcerror.NewEncoder(grpcerror.CodeMapping{
		Code:   codes.PermissionDenied,
		Errors: []error{authdictionary.ErrDataExists},
	})
	app := fiber.New()
	app.Get("/profile", func(ctx fiber.Ctx) error {
		return writer.Write(ctx, appresponse.Result{
			Err: encoder.Encode(authdictionary.ErrDataExists, locale.English),
		})
	})

	response := performRequest(t, app, "/profile", "en")
	assertResponse(
		t,
		response,
		http.StatusConflict,
		fixtures.AlreadyExistsEnglishBody,
	)
}

func mapMissingHTTPEndpoint(t *testing.T) {
	writer := newWriter()
	app := fiber.New(fiber.Config{
		ErrorHandler: func(ctx fiber.Ctx, err error) error {
			return writer.Write(ctx, appresponse.Result{Err: err})
		},
	})

	response := performRequest(t, app, "/missing-endpoint", "id-ID")
	assertResponse(
		t,
		response,
		http.StatusNotFound,
		fixtures.EndpointNotFoundBody,
	)
}

func mapBadRequest(t *testing.T) {
	writer := newWriter()
	fieldErrors := dictionary.FieldErrors{}
	fieldErrors.Add("email", authdictionary.ErrRequired)

	app := fiber.New()
	app.Get("/profile", func(ctx fiber.Ctx) error {
		return writer.Write(ctx, appresponse.Result{
			Err: newErrorEncoder().Encode(fieldErrors, locale.Indonesian),
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

func mapArbitraryLanguageFieldErrors(t *testing.T) {
	writer := newWriter()
	fieldErrors := dictionary.FieldErrors{}
	fieldErrors.Add("email", authdictionary.ErrRequired)
	fieldErrors.Add("email", authdictionary.ErrMinLength(6))

	app := fiber.New()
	app.Get("/profile", func(ctx fiber.Ctx) error {
		return writer.Write(ctx, appresponse.Result{
			Err: newErrorEncoder().Encode(fieldErrors, locale.Language("zh-CN")),
		})
	})

	response := performRequest(t, app, "/profile", "zh-CN")
	assertResponse(
		t,
		response,
		http.StatusBadRequest,
		fixtures.BadRequestChineseBody,
	)
}

func mapErrorPackageErrors(t *testing.T) {
	writer := newWriter()
	fieldErrors := errorpackage.NewErrors()
	fieldErrors.Add("email", authdictionary.ErrRequired)

	app := fiber.New()
	app.Get("/profile", func(ctx fiber.Ctx) error {
		return writer.Write(ctx, appresponse.Result{
			Err: newErrorEncoder().Encode(fieldErrors, locale.Indonesian),
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

func mapEncodedUnknownError(t *testing.T) {
	writer := newWriter()
	app := fiber.New()
	app.Get("/profile", func(ctx fiber.Ctx) error {
		return writer.Write(ctx, appresponse.Result{
			Err: newErrorEncoder().Encode(
				errors.New("database password leaked"),
				locale.English,
			),
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

func newErrorEncoder() *grpcerror.Encoder {
	mappings := append(
		grpcmapping.CommonCodeMappings(),
		grpcmapping.AuthCodeMappings()...,
	)
	return grpcerror.NewEncoder(mappings...)
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
