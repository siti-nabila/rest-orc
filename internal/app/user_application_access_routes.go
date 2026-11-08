package app

import (
	"github.com/siti-nabila/rest-orc/internal/client/authservice"
	gateway "github.com/siti-nabila/rest-orc/internal/features/users/gateway/authservice"
	"github.com/siti-nabila/rest-orc/internal/features/users/handler"
	"github.com/siti-nabila/rest-orc/internal/features/users/usecase"
	"github.com/siti-nabila/rest-orc/internal/response"
)

func registerUserApplicationAccessRoutes(routes routeGroups, client *authservice.Client, writer *response.Writer) error {
	g, err := gateway.NewUserApplicationAccess(client)
	if err != nil {
		return err
	}
	list, err := usecase.NewListUserApplicationAccess(g)
	if err != nil {
		return err
	}
	grant, err := usecase.NewGrantApplicationAccess(g)
	if err != nil {
		return err
	}
	revoke, err := usecase.NewRevokeApplicationAccess(g)
	if err != nil {
		return err
	}
	h, err := handler.NewUserApplicationAccess(list, grant, revoke, writer)
	if err != nil {
		return err
	}
	h.RegisterRoutes(routes.router, routes.authentication)
	return nil
}
