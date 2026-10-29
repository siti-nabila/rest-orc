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

const (
	MeRoute                = "/me"
	Route                  = "/users" // Legacy public auth prefix is part of the HTTP contract.
	RegisterRoute          = "/create"
	LoginRoute             = "/login"
	PasswordRoute          = "/password"
	registerSuccessMessage = "Register Successfully"
	loginSuccessMessage    = "Login Successfully"
)

type RegisterUseCase interface {
	Execute(ctx context.Context, request domain.AuthRequest) (domain.RegisterResponse, error)
}

type LoginUseCase interface {
	Execute(ctx context.Context, request domain.AuthRequest) (domain.SessionResult, error)
}

type ChangePasswordUseCase interface {
	Execute(context.Context, uint64, domain.ChangePasswordRequest) (domain.ChangePasswordResponse, error)
}

type Handler struct {
	registerUseCase       RegisterUseCase
	loginUseCase          LoginUseCase
	changePasswordUseCase ChangePasswordUseCase
	responseWriter        *response.Writer
	browser               *BrowserPolicy
}

func New(register RegisterUseCase, login LoginUseCase, changePassword ChangePasswordUseCase, writer *response.Writer, browser *BrowserPolicy) (*Handler, error) {
	if register == nil || login == nil || changePassword == nil {
		return nil, dictionary.ErrAuthHandlerUseCaseRequired
	}
	if writer == nil {
		return nil, dictionary.ErrAuthHandlerResponseWriterRequired
	}
	if browser == nil {
		return nil, dictionary.ErrBrowserPolicyRequired
	}
	return &Handler{
		registerUseCase:       register,
		loginUseCase:          login,
		changePasswordUseCase: changePassword,
		responseWriter:        writer,
		browser:               browser,
	}, nil
}

func (handler *Handler) RegisterAuthenticatedRoutes(router fiber.Router) {
	router.Get("", handler.Me)
	router.Post(PasswordRoute, handler.ChangePassword)
}

func (handler *Handler) Me(ctx fiber.Ctx) error {
	principal, ok := requestauth.PrincipalFromContext(ctx.Context())
	if !ok {
		return common.ErrUnauthorized
	}
	return handler.responseWriter.Write(ctx, response.Result{
		Success: response.Success{Message: "Profile retrieved successfully."},
		Data:    meResponse(principal),
	})
}

func (handler *Handler) RegisterPublicRoutes(router fiber.Router) {
	router.Post(RegisterRoute, handler.Register)
	router.Post(LoginRoute, handler.Login)
}

func (h *Handler) Register(ctx fiber.Ctx) error {
	ctx.Set(fiber.HeaderCacheControl, "no-store")
	// Parse request body
	var request domain.AuthRequest
	if err := ctx.Bind().Body(&request); err != nil {
		return fiber.ErrBadRequest
	}

	// Execute use case
	result, err := h.registerUseCase.Execute(ctx.Context(), request)
	if err != nil {
		return err
	}

	// Write response
	return h.responseWriter.Write(ctx, response.Result{
		Success: response.Success{Message: registerSuccessMessage},
		Data:    result,
	})
}

func (h *Handler) Login(ctx fiber.Ctx) error {
	ctx.Set(fiber.HeaderCacheControl, "no-store")
	if err := h.browser.ValidateLogin(ctx); err != nil {
		return err
	}
	// Parse request body
	var request domain.AuthRequest
	if err := ctx.Bind().Body(&request); err != nil {
		return fiber.ErrBadRequest
	}

	nonce, err := h.browser.PrepareNonce()
	if err != nil {
		return err
	}
	// Execute use case
	result, err := h.loginUseCase.Execute(ctx.Context(), request)
	if err != nil {
		return err
	}

	data, err := h.browser.Issue(ctx, result, nonce)
	if err != nil {
		return err
	}
	// Write response
	return h.responseWriter.Write(ctx, response.Result{
		Success: response.Success{Message: loginSuccessMessage},
		Data:    data,
	})
}
