package handler

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	"github.com/siti-nabila/rest-orc/internal/response"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

const SessionRoute = "/auth"

type RefreshUseCase interface {
	Execute(context.Context, string) (domain.SessionResult, error)
}
type LogoutUseCase interface {
	Execute(context.Context, domain.LogoutCredential) (domain.LogoutResponse, error)
}
type SessionHandler struct {
	refresh RefreshUseCase
	logout  LogoutUseCase
	writer  *response.Writer
	browser *BrowserPolicy
}

func NewSession(refresh RefreshUseCase, logout LogoutUseCase, writer *response.Writer, browser *BrowserPolicy) (*SessionHandler, error) {
	if refresh == nil || logout == nil {
		return nil, dictionary.ErrAuthHandlerUseCaseRequired
	}
	if writer == nil {
		return nil, dictionary.ErrAuthHandlerResponseWriterRequired
	}
	if browser == nil {
		return nil, dictionary.ErrBrowserPolicyRequired
	}
	return &SessionHandler{refresh: refresh, logout: logout, writer: writer, browser: browser}, nil
}
func (h *SessionHandler) RegisterRoutes(router fiber.Router) {
	router.Post("/refresh", h.Refresh)
	router.Post("/logout", h.Logout)
	router.Get("/csrf", h.CSRF)
}

func (h *SessionHandler) Refresh(ctx fiber.Ctx) error {
	ctx.Set(fiber.HeaderCacheControl, "no-store")
	refresh, err := h.browser.RefreshCredential(ctx)
	if err != nil {
		return err
	}
	nonce, err := h.browser.PrepareNonce()
	if err != nil {
		return err
	}
	result, err := h.refresh.Execute(ctx.Context(), refresh)
	if err != nil {
		return err
	}
	data, err := h.browser.Issue(ctx, result, nonce)
	if err != nil {
		return err
	}
	return h.writer.Write(ctx, response.Result{Success: response.Success{Message: "Session refreshed successfully."}, Data: data})
}
func (h *SessionHandler) Logout(ctx fiber.Ctx) error {
	ctx.Set(fiber.HeaderCacheControl, "no-store")
	proof, err := h.browser.LogoutCredential(ctx)
	if err != nil {
		return err
	}
	result, err := h.logout.Execute(ctx.Context(), proof)
	if err != nil {
		return err
	}
	h.browser.Clear(ctx)
	return h.writer.Write(ctx, response.Result{Success: response.Success{Message: "Logout successfully."}, Data: result})
}
func (h *SessionHandler) CSRF(ctx fiber.Ctx) error {
	ctx.Set(fiber.HeaderCacheControl, "no-store")
	data, err := h.browser.Bootstrap(ctx)
	if err != nil {
		return err
	}
	return h.writer.Write(ctx, response.Result{Success: response.Success{Message: "CSRF token issued successfully."}, Data: data})
}
