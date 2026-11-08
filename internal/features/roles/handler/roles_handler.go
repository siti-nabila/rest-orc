package handler

import (
	"context"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	requestauth "github.com/siti-nabila/rest-orc/internal/auth"
	"github.com/siti-nabila/rest-orc/internal/features/roles/domain"
	"github.com/siti-nabila/rest-orc/internal/response"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

type RoleListUseCase interface {
	Execute(context.Context, domain.Actor, domain.ListInput) (domain.Page[domain.Role], error)
}
type RoleGetUseCase interface {
	Execute(context.Context, domain.Actor, string) (domain.Detail, error)
}
type RoleCatalogHandler struct {
	list   RoleListUseCase
	get    RoleGetUseCase
	writer *response.Writer
}

func New(list RoleListUseCase, get RoleGetUseCase, writer *response.Writer) (*RoleCatalogHandler, error) {
	if list == nil || get == nil {
		return nil, dictionary.ErrRolesHandlerUseCaseRequired
	}
	if writer == nil {
		return nil, dictionary.ErrRolesHandlerWriterRequired
	}
	return &RoleCatalogHandler{list: list, get: get, writer: writer}, nil
}
func (h *RoleCatalogHandler) RegisterRoutes(router fiber.Router, authentication fiber.Handler) {
	router.Get("/roles/options", authentication, h.Options)
	router.Get("/roles/:role_id", authentication, h.Get)
	router.Get("/roles", authentication, h.List)
}
func actorFromRequest(c fiber.Ctx) (domain.Actor, error) {
	c.Set(fiber.HeaderCacheControl, "no-store")
	p, ok := requestauth.PrincipalFromContext(c.Context())
	if !ok || p.ID == 0 {
		return domain.Actor{}, common.ErrUnauthorized
	}
	return domain.Actor{UserID: p.ID, MustChangePassword: p.MustChangePassword, Permissions: p.Permissions}, nil
}
func listInput(c fiber.Ctx) domain.ListInput {
	input := domain.ListInput{}
	c.Request().URI().QueryArgs().VisitAll(func(k, v []byte) { key := string(k); input[key] = append(input[key], string(v)) })
	return input
}
func (h *RoleCatalogHandler) List(c fiber.Ctx) error {
	a, err := actorFromRequest(c)
	if err != nil {
		return err
	}
	out, err := h.list.Execute(c.Context(), a, listInput(c))
	if err != nil {
		return err
	}
	return h.writer.Write(c, response.Result{Success: response.Success{Message: "Roles retrieved successfully."}, Data: out})
}
func (h *RoleCatalogHandler) Get(c fiber.Ctx) error {
	a, err := actorFromRequest(c)
	if err != nil {
		return err
	}
	out, err := h.get.Execute(c.Context(), a, c.Params("role_id"))
	if err != nil {
		return err
	}
	return h.writer.Write(c, response.Result{Success: response.Success{Message: "Role retrieved successfully."}, Data: out})
}
func (h *RoleCatalogHandler) Options(c fiber.Ctx) error {
	a, err := actorFromRequest(c)
	if err != nil {
		return err
	}
	page, err := h.list.Execute(c.Context(), a, listInput(c))
	if err != nil {
		return err
	}
	items := make([]domain.Option, len(page.Items))
	for i, r := range page.Items {
		label := r.Name
		if r.Description != nil && *r.Description != "" {
			label = *r.Description
		}
		items[i] = domain.Option{Value: strconv.FormatUint(r.ID, 10), Label: r.Code + " - " + label, Code: r.Code, RoleCode: strconv.FormatInt(int64(r.RoleCode), 10)}
	}
	out := domain.Page[domain.Option]{Items: items, Total: page.Total, Page: page.Page, Limit: page.Limit, TotalPages: page.TotalPages, HasNext: page.HasNext, HasPrev: page.HasPrev, NextCursor: page.NextCursor}
	return h.writer.Write(c, response.Result{Success: response.Success{Message: "Role options retrieved successfully."}, Data: out})
}
