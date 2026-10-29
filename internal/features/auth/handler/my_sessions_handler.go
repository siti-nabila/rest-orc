package handler

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	requestauth "github.com/siti-nabila/rest-orc/internal/auth"
	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	"github.com/siti-nabila/rest-orc/internal/response"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

type ListMySessionsUseCase interface {
	Execute(context.Context, domain.SessionActor, domain.MySessionsQuery) (domain.MySessionsPage, error)
}
type RevokeMySessionUseCase interface {
	Execute(context.Context, domain.SessionActor, string) (domain.RevokeMySessionResponse, error)
}
type RevokeAllMySessionsUseCase interface {
	Execute(context.Context, domain.SessionActor) (domain.RevokeAllMySessionsResponse, error)
}
type MySessionsHandler struct {
	list      ListMySessionsUseCase
	revoke    RevokeMySessionUseCase
	revokeAll RevokeAllMySessionsUseCase
	writer    *response.Writer
	browser   *BrowserPolicy
}

func NewMySessions(list ListMySessionsUseCase, revoke RevokeMySessionUseCase, all RevokeAllMySessionsUseCase, writer *response.Writer, browser *BrowserPolicy) (*MySessionsHandler, error) {
	if list == nil || revoke == nil || all == nil {
		return nil, dictionary.ErrAuthHandlerUseCaseRequired
	}
	if writer == nil {
		return nil, dictionary.ErrAuthHandlerResponseWriterRequired
	}
	if browser == nil {
		return nil, dictionary.ErrBrowserPolicyRequired
	}
	return &MySessionsHandler{list: list, revoke: revoke, revokeAll: all, writer: writer, browser: browser}, nil
}

func (h *MySessionsHandler) RegisterRoutes(meRoutes fiber.Router) {
	sessions := meRoutes.Group("/sessions")
	sessions.Get("", h.List)
	sessions.Delete("/:session_id", h.Revoke)
	sessions.Post("/revoke-all", h.RevokeAll)
}

func sessionActor(ctx fiber.Ctx) (domain.SessionActor, error) {
	ctx.Set(fiber.HeaderCacheControl, "no-store")
	principal, ok := requestauth.PrincipalFromContext(ctx.Context())
	if !ok || principal.ID == 0 {
		return domain.SessionActor{}, common.ErrUnauthorized
	}
	return domain.SessionActor{UserID: principal.ID, MustChangePassword: principal.MustChangePassword}, nil
}

func (h *MySessionsHandler) List(ctx fiber.Ctx) error {
	actor, err := sessionActor(ctx)
	if err != nil {
		return err
	}
	values := make(map[string][]string)
	for key, value := range ctx.Request().URI().QueryArgs().All() {
		name := string(key)
		values[name] = append(values[name], string(value))
	}
	query, err := domain.ParseMySessionsQuery(values)
	if err != nil {
		return err
	}
	result, err := h.list.Execute(ctx.Context(), actor, query)
	if err != nil {
		return err
	}
	return h.writer.Write(ctx, response.Result{Success: response.Success{Message: "Sessions retrieved successfully."}, Data: result})
}

func (h *MySessionsHandler) Revoke(ctx fiber.Ctx) error {
	actor, err := sessionActor(ctx)
	if err != nil {
		return err
	}
	result, err := h.revoke.Execute(ctx.Context(), actor, ctx.Params("session_id"))
	if err != nil {
		return err
	}
	return h.writer.Write(ctx, response.Result{Success: response.Success{Message: "Session revoked successfully."}, Data: result})
}

func (h *MySessionsHandler) RevokeAll(ctx fiber.Ctx) error {
	actor, err := sessionActor(ctx)
	if err != nil {
		return err
	}
	result, err := h.revokeAll.Execute(ctx.Context(), actor)
	if err != nil {
		return err
	}
	if result.LoginRequired {
		h.browser.Clear(ctx)
	}
	return h.writer.Write(ctx, response.Result{Success: response.Success{Message: "All sessions revoked successfully."}, Data: result})
}
