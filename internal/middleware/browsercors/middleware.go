package browsercors

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/api-contracts/pkg/locale"
)

// New uses the same validated exact origins as the auth browser policy.
func New(origins []string) fiber.Handler {
	allowed := make(map[string]bool, len(origins))
	for _, origin := range origins {
		allowed[origin] = true
	}
	handle := cors.New(cors.Config{AllowOrigins: append([]string(nil), origins...), AllowCredentials: true, MaxAge: -1,
		AllowMethods: []string{fiber.MethodGet, fiber.MethodPost, fiber.MethodPatch, fiber.MethodPut, fiber.MethodDelete, fiber.MethodOptions},
		AllowHeaders: []string{fiber.HeaderAuthorization, fiber.HeaderContentType, "X-CSRF-Token", "Version", "Idempotency-Key", locale.HTTPHeader, locale.MetadataKey, "X-Request-ID"},
	})
	return func(ctx fiber.Ctx) error {
		ctx.Vary(fiber.HeaderOrigin)
		values := ctx.Request().Header.PeekAll(fiber.HeaderOrigin)
		if len(values) > 1 || len(values) == 1 && !allowed[string(values[0])] {
			return common.ErrForbidden
		}
		return handle(ctx)
	}
}
