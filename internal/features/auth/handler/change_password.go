package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	requestauth "github.com/siti-nabila/rest-orc/internal/auth"
	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	"github.com/siti-nabila/rest-orc/internal/response"
)

func (handler *Handler) ChangePassword(ctx fiber.Ctx) error {
	ctx.Set(fiber.HeaderCacheControl, "no-store")
	principal, ok := requestauth.PrincipalFromContext(ctx.Context())
	if !ok || principal.ID == 0 {
		return common.ErrUnauthorized
	}
	var request domain.ChangePasswordRequest
	if err := ctx.Bind().Body(&request); err != nil {
		return fiber.ErrBadRequest
	}
	result, err := handler.changePasswordUseCase.Execute(ctx.Context(), principal.ID, request)
	if err != nil {
		return err
	}
	if result.LoginRequired {
		handler.browser.Clear(ctx)
	}
	return handler.responseWriter.Write(ctx, response.Result{
		Success: response.Success{Message: "Password changed successfully."},
		Data:    result,
	})
}
