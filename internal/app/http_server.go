package app

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/healthcheck"
	recovermw "github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/siti-nabila/rest-orc/internal/config"
)

const internalErrorMessage = "internal server error"

type errorResponse struct {
	Error string `json:"error"`
}

func newHTTPServer(cfg *config.Config) *fiber.App {
	httpServer := fiber.New(fiber.Config{
		AppName:      cfg.App.Name,
		ReadTimeout:  cfg.Server.ReadTimeout.Duration,
		WriteTimeout: cfg.Server.WriteTimeout.Duration,
		IdleTimeout:  cfg.Server.IdleTimeout.Duration,
		ErrorHandler: handleHTTPError,
	})

	httpServer.Use(recovermw.New(recovermw.Config{
		EnableStackTrace: false,
		PanicHandler: func(fiber.Ctx, any) error {
			return fiber.ErrInternalServerError
		},
	}))
	httpServer.Get(
		healthcheck.LivenessEndpoint,
		healthcheck.New(healthcheck.Config{ResponseFormat: healthcheck.FormatJSON}),
	)

	return httpServer
}

func handleHTTPError(ctx fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError
	message := internalErrorMessage

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		status = fiberErr.Code
		if status < fiber.StatusInternalServerError {
			message = fiberErr.Message
		}
	}

	return ctx.Status(status).JSON(errorResponse{Error: message})
}
