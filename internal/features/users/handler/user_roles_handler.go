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

type UserRoleListUseCase interface {
	Execute(context.Context, domain.UserRoleActor, domain.ListUserRolesRequest) (domain.ListUserRolesResponse, error)
}
type UserRoleGrantUseCase interface {
	Execute(context.Context, domain.UserRoleActor, domain.UserRoleCommandRequest) (domain.UserRoleCommandResponse, error)
}
type UserRoleRevokeUseCase interface {
	Execute(context.Context, domain.UserRoleActor, domain.UserRoleCommandRequest) (domain.UserRoleCommandResponse, error)
}
type UserRoleManagementHandler struct {
	list   UserRoleListUseCase
	grant  UserRoleGrantUseCase
	revoke UserRoleRevokeUseCase
	writer *response.Writer
}

func NewUserRoleManagement(list UserRoleListUseCase, grant UserRoleGrantUseCase, revoke UserRoleRevokeUseCase, writer *response.Writer) (*UserRoleManagementHandler, error) {
	if list == nil || grant == nil || revoke == nil {
		return nil, dictionary.ErrUsersHandlerUseCaseRequired
	}
	if writer == nil {
		return nil, dictionary.ErrUsersHandlerResponseWriterRequired
	}
	return &UserRoleManagementHandler{list: list, grant: grant, revoke: revoke, writer: writer}, nil
}
func (h *UserRoleManagementHandler) RegisterRoutes(router fiber.Router, authentication fiber.Handler) {
	router.Get("/users/:user_id/roles", authentication, h.ListUserRoles)
	router.Post("/users/:user_id/roles/grant", authentication, h.GrantUserRoles)
	router.Post("/users/:user_id/roles/revoke", authentication, h.RevokeUserRoles)
}
func userRoleActor(c fiber.Ctx) (domain.UserRoleActor, error) {
	c.Set(fiber.HeaderCacheControl, "no-store")
	p, ok := requestauth.PrincipalFromContext(c.Context())
	if !ok || p.ID == 0 {
		return domain.UserRoleActor{}, common.ErrUnauthorized
	}
	return domain.UserRoleActor{UserID: p.ID, MustChangePassword: p.MustChangePassword, Permissions: p.Permissions}, nil
}
func (h *UserRoleManagementHandler) ListUserRoles(c fiber.Ctx) error {
	a, err := userRoleActor(c)
	if err != nil {
		return err
	}
	input := domain.UserRoleListInput{}
	c.Request().URI().QueryArgs().VisitAll(func(k, v []byte) { key := string(k); input[key] = append(input[key], string(v)) })
	out, err := h.list.Execute(c.Context(), a, domain.ListUserRolesRequest{UserID: c.Params("user_id"), Input: input})
	if err != nil {
		return err
	}
	return h.writer.Write(c, response.Result{Success: response.Success{Message: "User roles retrieved successfully."}, Data: out})
}
func parseUserRoleCommand(c fiber.Ctx) (domain.UserRoleCommandRequest, error) {
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
		return domain.UserRoleCommandRequest{}, err
	}
	raw := bytes.TrimSpace(c.Body())
	if len(raw) == 0 || raw[0] != '{' {
		return domain.UserRoleCommandRequest{}, fiber.ErrBadRequest
	}
	var body domain.UserRoleCommandBody
	if json.Unmarshal(raw, &body) != nil {
		return domain.UserRoleCommandRequest{}, fiber.ErrBadRequest
	}
	return domain.UserRoleCommandRequest{UserID: c.Params("user_id"), UserRoleCommandBody: body, Version: version, IdempotencyKey: key}, nil
}
func (h *UserRoleManagementHandler) GrantUserRoles(c fiber.Ctx) error {
	a, err := userRoleActor(c)
	if err != nil {
		return err
	}
	r, err := parseUserRoleCommand(c)
	if err != nil {
		return err
	}
	out, err := h.grant.Execute(c.Context(), a, r)
	if err != nil {
		return err
	}
	return h.writer.Write(c, response.Result{Success: response.Success{Message: "User roles granted successfully."}, Data: out})
}
func (h *UserRoleManagementHandler) RevokeUserRoles(c fiber.Ctx) error {
	a, err := userRoleActor(c)
	if err != nil {
		return err
	}
	r, err := parseUserRoleCommand(c)
	if err != nil {
		return err
	}
	out, err := h.revoke.Execute(c.Context(), a, r)
	if err != nil {
		return err
	}
	return h.writer.Write(c, response.Result{Success: response.Success{Message: "User roles revoked successfully."}, Data: out})
}
