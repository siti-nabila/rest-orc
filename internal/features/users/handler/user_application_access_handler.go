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

type UserApplicationAccessListUseCase interface {
	Execute(context.Context, domain.UserApplicationAccessActor, domain.ListUserApplicationAccessRequest) (domain.ListUserApplicationAccessResponse, error)
}
type ApplicationAccessGrantUseCase interface {
	Execute(context.Context, domain.UserApplicationAccessActor, domain.GrantApplicationAccessRequest) (domain.ApplicationAccessCommandResponse, error)
}
type ApplicationAccessRevokeUseCase interface {
	Execute(context.Context, domain.UserApplicationAccessActor, domain.RevokeApplicationAccessRequest) (domain.ApplicationAccessCommandResponse, error)
}
type UserApplicationAccessHandler struct {
	list   UserApplicationAccessListUseCase
	grant  ApplicationAccessGrantUseCase
	revoke ApplicationAccessRevokeUseCase
	writer *response.Writer
}

func NewUserApplicationAccess(list UserApplicationAccessListUseCase, grant ApplicationAccessGrantUseCase, revoke ApplicationAccessRevokeUseCase, writer *response.Writer) (*UserApplicationAccessHandler, error) {
	if list == nil || grant == nil || revoke == nil {
		return nil, dictionary.ErrUsersHandlerUseCaseRequired
	}
	if writer == nil {
		return nil, dictionary.ErrUsersHandlerResponseWriterRequired
	}
	return &UserApplicationAccessHandler{list: list, grant: grant, revoke: revoke, writer: writer}, nil
}
func (h *UserApplicationAccessHandler) RegisterRoutes(router fiber.Router, authentication fiber.Handler) {
	router.Get("/users/:user_id/applications", authentication, h.ListUserApplicationAccess)
	router.Post("/users/:user_id/applications/:application_code/grant", authentication, h.GrantApplicationAccess)
	router.Post("/users/:user_id/applications/:application_code/revoke", authentication, h.RevokeApplicationAccess)
}
func userApplicationAccessActor(c fiber.Ctx) (domain.UserApplicationAccessActor, error) {
	c.Set(fiber.HeaderCacheControl, "no-store")
	p, ok := requestauth.PrincipalFromContext(c.Context())
	if !ok || p.ID == 0 {
		return domain.UserApplicationAccessActor{}, common.ErrUnauthorized
	}
	return domain.UserApplicationAccessActor{UserID: p.ID, MustChangePassword: p.MustChangePassword, Permissions: p.Permissions}, nil
}
func (h *UserApplicationAccessHandler) ListUserApplicationAccess(c fiber.Ctx) error {
	a, err := userApplicationAccessActor(c)
	if err != nil {
		return err
	}
	input := domain.UserApplicationAccessListInput{}
	c.Request().URI().QueryArgs().VisitAll(func(k, v []byte) { key := string(k); input[key] = append(input[key], string(v)) })
	out, err := h.list.Execute(c.Context(), a, domain.ListUserApplicationAccessRequest{UserID: c.Params("user_id"), Input: input})
	if err != nil {
		return err
	}
	return h.writer.Write(c, response.Result{Success: response.Success{Message: "User application access retrieved successfully."}, Data: out})
}
func parseApplicationAccessCommand(c fiber.Ctx) (domain.ApplicationAccessCommandRequest, error) {
	keyHeaders := c.Request().Header.PeekAll("Idempotency-Key")
	keys := make([]string, len(keyHeaders))
	for i, v := range keyHeaders {
		keys[i] = string(v)
	}
	versionHeaders := c.Request().Header.PeekAll("Version")
	versions := make([]string, len(versionHeaders))
	for i, v := range versionHeaders {
		versions[i] = string(v)
	}
	key, version, err := domain.ParseUserRoleHeaders(keys, versions)
	if err != nil {
		return domain.ApplicationAccessCommandRequest{}, err
	}
	raw := bytes.TrimSpace(c.Body())
	if len(raw) == 0 || raw[0] != '{' {
		return domain.ApplicationAccessCommandRequest{}, fiber.ErrBadRequest
	}
	return domain.ApplicationAccessCommandRequest{UserID: c.Params("user_id"), ApplicationCode: c.Params("application_code"), Version: version, IdempotencyKey: key}, nil
}
func (h *UserApplicationAccessHandler) GrantApplicationAccess(c fiber.Ctx) error {
	a, err := userApplicationAccessActor(c)
	if err != nil {
		return err
	}
	r, err := parseApplicationAccessCommand(c)
	if err != nil {
		return err
	}
	var body domain.GrantApplicationAccessBody
	if json.Unmarshal(bytes.TrimSpace(c.Body()), &body) != nil {
		return fiber.ErrBadRequest
	}
	out, err := h.grant.Execute(c.Context(), a, domain.GrantApplicationAccessRequest{ApplicationAccessCommandRequest: r, GrantApplicationAccessBody: body})
	if err != nil {
		return err
	}
	return h.writer.Write(c, response.Result{Success: response.Success{Message: "Application access granted successfully."}, Data: out})
}
func (h *UserApplicationAccessHandler) RevokeApplicationAccess(c fiber.Ctx) error {
	a, err := userApplicationAccessActor(c)
	if err != nil {
		return err
	}
	r, err := parseApplicationAccessCommand(c)
	if err != nil {
		return err
	}
	var body domain.RevokeApplicationAccessBody
	if json.Unmarshal(bytes.TrimSpace(c.Body()), &body) != nil {
		return fiber.ErrBadRequest
	}
	out, err := h.revoke.Execute(c.Context(), a, domain.RevokeApplicationAccessRequest{ApplicationAccessCommandRequest: r, RevokeApplicationAccessBody: body})
	if err != nil {
		return err
	}
	return h.writer.Write(c, response.Result{Success: response.Success{Message: "Application access revoked successfully."}, Data: out})
}
