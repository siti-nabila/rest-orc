package app

import (
	"github.com/siti-nabila/rest-orc/internal/client/authservice"
	gateway "github.com/siti-nabila/rest-orc/internal/features/users/gateway/authservice"
	"github.com/siti-nabila/rest-orc/internal/features/users/handler"
	"github.com/siti-nabila/rest-orc/internal/features/users/usecase"
	"github.com/siti-nabila/rest-orc/internal/response"
)

func registerAdminSessionRoutes(routes routeGroups, client *authservice.Client, writer *response.Writer) error {
	g, err := gateway.NewAdminSessions(client)
	if err != nil {
		return err
	}
	list, err := usecase.NewListUserSessions(g)
	if err != nil {
		return err
	}
	revoke, err := usecase.NewRevokeUserSession(g)
	if err != nil {
		return err
	}
	all, err := usecase.NewRevokeAllUserSessions(g)
	if err != nil {
		return err
	}
	h, err := handler.NewAdminSessions(list, revoke, all, writer)
	if err != nil {
		return err
	}
	h.RegisterRoutes(routes.router, routes.authentication)
	return nil
}
