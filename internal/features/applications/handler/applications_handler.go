package handler

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	requestauth "github.com/siti-nabila/rest-orc/internal/auth"
	"github.com/siti-nabila/rest-orc/internal/features/applications/domain"
	"github.com/siti-nabila/rest-orc/internal/response"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

type ApplicationListUseCase interface {
	Execute(context.Context, domain.ApplicationActor, domain.ApplicationListInput) (domain.ApplicationPage[domain.Application], error)
}
type ApplicationCatalogHandler struct {
	list   ApplicationListUseCase
	writer *response.Writer
}

func New(list ApplicationListUseCase, writer *response.Writer) (*ApplicationCatalogHandler, error) {
	if list == nil {
		return nil, dictionary.ErrApplicationsHandlerUseCaseRequired
	}
	if writer == nil {
		return nil, dictionary.ErrApplicationsHandlerWriterRequired
	}
	return &ApplicationCatalogHandler{list: list, writer: writer}, nil
}
func (h *ApplicationCatalogHandler) RegisterRoutes(router fiber.Router, authentication fiber.Handler) {
	router.Get("/applications/options", authentication, h.Options)
	router.Get("/applications", authentication, h.List)
}
func applicationReadInput(c fiber.Ctx) (domain.ApplicationActor, domain.ApplicationListInput, error) {
	c.Set(fiber.HeaderCacheControl, "no-store")
	p, ok := requestauth.PrincipalFromContext(c.Context())
	if !ok || p.ID == 0 {
		return domain.ApplicationActor{}, nil, common.ErrUnauthorized
	}
	input := domain.ApplicationListInput{}
	c.Request().URI().QueryArgs().VisitAll(func(k, v []byte) { key := string(k); input[key] = append(input[key], string(v)) })
	return domain.ApplicationActor{UserID: p.ID, MustChangePassword: p.MustChangePassword, Permissions: p.Permissions}, input, nil
}
func (h *ApplicationCatalogHandler) List(c fiber.Ctx) error {
	a, input, err := applicationReadInput(c)
	if err != nil {
		return err
	}
	out, err := h.list.Execute(c.Context(), a, input)
	if err != nil {
		return err
	}
	return h.writer.Write(c, response.Result{Success: response.Success{Message: "Applications retrieved successfully."}, Data: out})
}
func (h *ApplicationCatalogHandler) Options(c fiber.Ctx) error {
	a, input, err := applicationReadInput(c)
	if err != nil {
		return err
	}
	page, err := h.list.Execute(c.Context(), a, input)
	if err != nil {
		return err
	}
	items := make([]domain.ApplicationOption, len(page.Items))
	for i, app := range page.Items {
		items[i] = domain.ApplicationOption{Value: app.Code, Label: app.Code + " - " + app.Name}
	}
	out := domain.ApplicationPage[domain.ApplicationOption]{
		Items: items, Total: page.Total, Page: page.Page, Limit: page.Limit, TotalPages: page.TotalPages,
		HasNext: page.HasNext, HasPrev: page.HasPrev, NextCursor: page.NextCursor,
	}
	return h.writer.Write(c, response.Result{Success: response.Success{Message: "Application options retrieved successfully."}, Data: out})
}
