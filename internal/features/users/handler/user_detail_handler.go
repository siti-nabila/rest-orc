package handler

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	requestauth "github.com/siti-nabila/rest-orc/internal/auth"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/internal/response"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

type UserDetailUseCase interface {
	Execute(context.Context, domain.UserDetailActor, uint64) (domain.UserDetail, error)
}

type UserDetailHandler struct {
	usecase UserDetailUseCase
	writer  *response.Writer
}

func NewUserDetail(usecase UserDetailUseCase, writer *response.Writer) (*UserDetailHandler, error) {
	if usecase == nil {
		return nil, dictionary.ErrUsersHandlerUseCaseRequired
	}
	if writer == nil {
		return nil, dictionary.ErrUsersHandlerResponseWriterRequired
	}
	return &UserDetailHandler{usecase: usecase, writer: writer}, nil
}

// RegisterRoutes precedes the broader legacy role-admin /users group.
func (h *UserDetailHandler) RegisterRoutes(router fiber.Router, authentication fiber.Handler) {
	router.Get("/users/:user_id", authentication, h.Get)
}

func (h *UserDetailHandler) Get(ctx fiber.Ctx) error {
	ctx.Set(fiber.HeaderCacheControl, "no-store")
	principal, ok := requestauth.PrincipalFromContext(ctx.Context())
	if !ok || principal.ID == 0 {
		return common.ErrUnauthorized
	}
	target, err := domain.ParseUserDetailID(ctx.Params("user_id"))
	if err != nil {
		return err
	}
	actor := domain.UserDetailActor{UserID: principal.ID, MustChangePassword: principal.MustChangePassword, Permissions: principal.Permissions}
	result, err := h.usecase.Execute(ctx.Context(), actor, target)
	if err != nil {
		return err
	}
	return h.writer.Write(ctx, response.Result{Success: response.Success{Message: "User retrieved successfully."}, Data: result})
}
