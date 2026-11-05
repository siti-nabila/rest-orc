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

type UserProfileUseCase interface {
	Execute(context.Context, domain.UserProfileActor, domain.UpdateUserProfileRequest) (domain.UpdateUserProfileResponse, error)
}
type UserProfileHandler struct {
	usecase UserProfileUseCase
	writer  *response.Writer
}

func NewUserProfile(usecase UserProfileUseCase, writer *response.Writer) (*UserProfileHandler, error) {
	if usecase == nil {
		return nil, dictionary.ErrUsersHandlerUseCaseRequired
	}
	if writer == nil {
		return nil, dictionary.ErrUsersHandlerResponseWriterRequired
	}
	return &UserProfileHandler{usecase: usecase, writer: writer}, nil
}

// RegisterRoutes precedes the broader legacy role-admin users group.
func (h *UserProfileHandler) RegisterRoutes(router fiber.Router, authentication fiber.Handler) {
	router.Patch("/users/:user_id/profile", authentication, h.Update)
}
func (h *UserProfileHandler) Update(ctx fiber.Ctx) error {
	ctx.Set(fiber.HeaderCacheControl, "no-store")
	principal, ok := requestauth.PrincipalFromContext(ctx.Context())
	if !ok || principal.ID == 0 {
		return common.ErrUnauthorized
	}
	target, err := domain.ParseUserDetailID(ctx.Params("user_id"))
	if err != nil {
		return err
	}
	keys := ctx.Request().Header.PeekAll("Idempotency-Key")
	keyValues := make([]string, len(keys))
	for i, k := range keys {
		keyValues[i] = string(k)
	}
	versions := ctx.Request().Header.PeekAll("Version")
	versionValues := make([]string, len(versions))
	for i, v := range versions {
		versionValues[i] = string(v)
	}
	key, version, err := domain.ParseUserProfileHeaders(keyValues, versionValues)
	if err != nil {
		return err
	}
	raw := bytes.TrimSpace(ctx.Body())
	if len(raw) == 0 || (raw[0] != '{' && !bytes.Equal(raw, []byte("null"))) {
		return fiber.ErrBadRequest
	}
	var body domain.UpdateProfileRequest
	if err := json.Unmarshal(ctx.Body(), &body); err != nil {
		return fiber.ErrBadRequest
	}
	actor := domain.UserProfileActor{UserID: principal.ID, MustChangePassword: principal.MustChangePassword, Permissions: principal.Permissions}
	result, err := h.usecase.Execute(ctx.Context(), actor, domain.UpdateUserProfileRequest{UserID: target, Profile: body, Version: version, IdempotencyKey: key})
	if err != nil {
		return err
	}
	return h.writer.Write(ctx, response.Result{Success: response.Success{Message: "User profile updated successfully."}, Data: result})
}
