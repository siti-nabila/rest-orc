package testutils

import (
	"io"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	authdictionary "github.com/siti-nabila/api-contracts/pkg/dictionary/auth"
	commondictionary "github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	appresponse "github.com/siti-nabila/rest-orc/internal/response"
)

func NewHTTPApp(configs ...fiber.Config) (*fiber.App, *appresponse.Writer) {
	writer := appresponse.NewWriter(appresponse.NewErrorMapper(
		authdictionary.Registry(),
		commondictionary.Registry(),
	))
	var cfg fiber.Config
	if len(configs) > 0 {
		cfg = configs[0]
	}
	cfg.ErrorHandler = func(ctx fiber.Ctx, err error) error {
		return writer.Write(ctx, appresponse.Result{Err: err})
	}
	app := fiber.New(cfg)
	return app, writer
}

func ResponseBody(t *testing.T, response *http.Response, err error) string {
	t.Helper()

	if err != nil {
		t.Fatalf("HTTP request error = %v", err)
	}
	if response == nil {
		t.Fatal("HTTP response = nil")
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close HTTP response body: %v", err)
		}
	}()

	body, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		t.Fatalf("read HTTP response body: %v", readErr)
	}
	return string(body)
}
