package handler

import (
	"context"

	"github.com/gofiber/fiber/v3"
	contractdictionary "github.com/siti-nabila/api-contracts/pkg/dictionary"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	requestauth "github.com/siti-nabila/rest-orc/internal/auth"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/internal/response"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

type ListUserSessionsUseCase interface {
	Execute(context.Context, domain.AdminSessionActor, uint64, domain.AdminSessionsQuery) (domain.AdminSessionsPage, error)
}
type RevokeUserSessionUseCase interface {
	Execute(context.Context, domain.AdminSessionActor, domain.RevokeUserSessionRequest) (domain.RevokeUserSessionResponse, error)
}
type RevokeAllUserSessionsUseCase interface {
	Execute(context.Context, domain.AdminSessionActor, domain.RevokeAllUserSessionsRequest) (domain.RevokeAllUserSessionsResponse, error)
}
type AdminSessionsHandler struct {
	list   ListUserSessionsUseCase
	revoke RevokeUserSessionUseCase
	all    RevokeAllUserSessionsUseCase
	writer *response.Writer
}

func NewAdminSessions(list ListUserSessionsUseCase, revoke RevokeUserSessionUseCase, all RevokeAllUserSessionsUseCase, writer *response.Writer) (*AdminSessionsHandler, error) {
	if list == nil || revoke == nil || all == nil {
		return nil, dictionary.ErrUsersHandlerUseCaseRequired
	}
	if writer == nil {
		return nil, dictionary.ErrUsersHandlerResponseWriterRequired
	}
	return &AdminSessionsHandler{list: list, revoke: revoke, all: all, writer: writer}, nil
}

// RegisterRoutes must precede the broader legacy role-admin /users group.
func (h *AdminSessionsHandler) RegisterRoutes(router fiber.Router, authentication fiber.Handler) {
	router.Get("/users/:user_id/sessions", authentication, h.List)
	router.Delete("/users/:user_id/sessions/:session_id", authentication, h.Revoke)
	router.Post("/users/:user_id/sessions/revoke-all", authentication, h.RevokeAll)
}

func adminSessionActor(ctx fiber.Ctx) (domain.AdminSessionActor, error) {
	ctx.Set(fiber.HeaderCacheControl, "no-store")
	principal, ok := requestauth.PrincipalFromContext(ctx.Context())
	if !ok || principal.ID == 0 {
		return domain.AdminSessionActor{}, common.ErrUnauthorized
	}
	return domain.AdminSessionActor{UserID: principal.ID, MustChangePassword: principal.MustChangePassword, Permissions: principal.Permissions}, nil
}

func (h *AdminSessionsHandler) List(ctx fiber.Ctx) error {
	actor, err := adminSessionActor(ctx)
	if err != nil {
		return err
	}
	target, err := domain.ParseAdminSessionUserID(ctx.Params("user_id"))
	if err != nil {
		return err
	}
	values := make(map[string][]string)
	for key, value := range ctx.Request().URI().QueryArgs().All() {
		name := string(key)
		values[name] = append(values[name], string(value))
	}
	query, err := domain.ParseAdminSessionsQuery(values)
	if err != nil {
		return err
	}
	result, err := h.list.Execute(ctx.Context(), actor, target, query)
	if err != nil {
		return err
	}
	return h.writer.Write(ctx, response.Result{Success: response.Success{Message: "User sessions retrieved successfully."}, Data: result})
}

func adminSessionKey(ctx fiber.Ctx) (string, error) {
	values := ctx.Request().Header.PeekAll("Idempotency-Key")
	if len(values) != 1 {
		return "", contractdictionary.FieldErrors{"idempotency_key": {dictionary.ErrInvalidSessionIdempotencyKey}}
	}
	key := string(values[0])
	if err := domain.ValidateSessionIdempotencyKey(key); err != nil {
		return "", err
	}
	return key, nil
}

func (h *AdminSessionsHandler) Revoke(ctx fiber.Ctx) error {
	actor, err := adminSessionActor(ctx)
	if err != nil {
		return err
	}
	target, err := domain.ParseAdminSessionUserID(ctx.Params("user_id"))
	if err != nil {
		return err
	}
	key, err := adminSessionKey(ctx)
	if err != nil {
		return err
	}
	result, err := h.revoke.Execute(ctx.Context(), actor, domain.RevokeUserSessionRequest{UserID: target, SessionID: ctx.Params("session_id"), IdempotencyKey: key})
	if err != nil {
		return err
	}
	return h.writer.Write(ctx, response.Result{Success: response.Success{Message: "User session revoked successfully."}, Data: result})
}

func (h *AdminSessionsHandler) RevokeAll(ctx fiber.Ctx) error {
	actor, err := adminSessionActor(ctx)
	if err != nil {
		return err
	}
	target, err := domain.ParseAdminSessionUserID(ctx.Params("user_id"))
	if err != nil {
		return err
	}
	key, err := adminSessionKey(ctx)
	if err != nil {
		return err
	}
	versions := ctx.Request().Header.PeekAll("Version")
	if len(versions) != 1 {
		return contractdictionary.FieldErrors{"version": {dictionary.ErrInvalidSessionVersion}}
	}
	version, err := domain.ParseAdminSessionVersion(string(versions[0]))
	if err != nil {
		return err
	}
	result, err := h.all.Execute(ctx.Context(), actor, domain.RevokeAllUserSessionsRequest{UserID: target, IdempotencyKey: key, Version: version})
	if err != nil {
		return err
	}
	return h.writer.Write(ctx, response.Result{Success: response.Success{Message: "All user sessions revoked successfully."}, Data: result})
}
