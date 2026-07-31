package app

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/healthcheck"
	recovermw "github.com/gofiber/fiber/v3/middleware/recover"
	authdictionary "github.com/siti-nabila/api-contracts/pkg/dictionary/auth"
	commondictionary "github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/config"
	"github.com/siti-nabila/rest-orc/internal/response"
)

func newHTTPServer(cfg *config.Config) *fiber.App {
	responseWriter := response.NewWriter(response.NewErrorMapper(
		authdictionary.Registry(),
		commondictionary.Registry(),
	))
	httpServer := fiber.New(fiber.Config{
		AppName:      cfg.App.Name,
		ReadTimeout:  cfg.Server.ReadTimeout.Duration,
		WriteTimeout: cfg.Server.WriteTimeout.Duration,
		IdleTimeout:  cfg.Server.IdleTimeout.Duration,
		ErrorHandler: func(ctx fiber.Ctx, err error) error {
			return responseWriter.Write(ctx, response.Result{Err: err})
		},
	})

	httpServer.Use(recovermw.New(recovermw.Config{
		EnableStackTrace: false,
		PanicHandler: func(fiber.Ctx, any) error {
			return commondictionary.ErrInternalServerError
		},
	}))
	httpServer.Get(
		healthcheck.LivenessEndpoint,
		healthcheck.New(healthcheck.Config{ResponseFormat: healthcheck.FormatJSON}),
	)

	return httpServer
}
