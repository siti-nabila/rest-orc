package authorization

import (
	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/auth"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	requestauth "github.com/siti-nabila/rest-orc/internal/auth"
)

func RequireRole(role requestauth.RoleCode) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		principal, exists := requestauth.PrincipalFromContext(ctx.Context())
		if !exists {
			return common.ErrUnauthorized
		}
		if !principal.HasRole(requestauth.RoleAdmin.Int32()) {
			return auth.ErrAuthNotAllowed
		}
		return ctx.Next()
	}
}
