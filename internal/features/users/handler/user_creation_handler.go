package handler

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	requestauth "github.com/siti-nabila/rest-orc/internal/auth"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/internal/response"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

type UserCreationUseCase interface {
	Execute(context.Context, domain.UserCreationActor, domain.CreateUserRequest) (domain.CreateUserResponse, error)
}
type UserCreationHandler struct {
	usecase UserCreationUseCase
	writer  *response.Writer
}

func NewUserCreation(usecase UserCreationUseCase, writer *response.Writer) (*UserCreationHandler, error) {
	if usecase == nil {
		return nil, dictionary.ErrUsersHandlerUseCaseRequired
	}
	if writer == nil {
		return nil, dictionary.ErrUsersHandlerResponseWriterRequired
	}
	return &UserCreationHandler{usecase: usecase, writer: writer}, nil
}

// RegisterRoutes precedes the broader legacy role-admin users group.
func (h *UserCreationHandler) RegisterRoutes(router fiber.Router, authentication fiber.Handler) {
	router.Post("/users", authentication, h.Create)
}
func (h *UserCreationHandler) Create(ctx fiber.Ctx) error {
	ctx.Set(fiber.HeaderCacheControl, "no-store")
	principal, ok := requestauth.PrincipalFromContext(ctx.Context())
	if !ok || principal.ID == 0 {
		return common.ErrUnauthorized
	}
	keys := ctx.Request().Header.PeekAll("Idempotency-Key")
	keyValues := make([]string, len(keys))
	for i, k := range keys {
		keyValues[i] = string(k)
	}
	key, err := domain.ParseUserCreationKey(keyValues)
	if err != nil {
		return err
	}
	raw := bytes.TrimSpace(ctx.Body())
	if len(raw) == 0 || raw[0] != '{' {
		return fiber.ErrBadRequest
	}
	var body domain.CreateUserBody
	if err := json.Unmarshal(ctx.Body(), &body); err != nil {
		return fiber.ErrBadRequest
	}
	actor := domain.UserCreationActor{UserID: principal.ID, MustChangePassword: principal.MustChangePassword, Permissions: principal.Permissions}
	result, err := h.usecase.Execute(ctx.Context(), actor, domain.CreateUserRequest{Body: body, IdempotencyKey: key})
	if err != nil {
		return err
	}
	return h.writer.Write(ctx, response.Result{Success: response.Success{Message: "User created successfully."}, Data: result})
}
