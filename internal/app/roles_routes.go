package app

import (
	"github.com/siti-nabila/rest-orc/internal/client/authservice"
	gateway "github.com/siti-nabila/rest-orc/internal/features/roles/gateway/authservice"
	"github.com/siti-nabila/rest-orc/internal/features/roles/handler"
	"github.com/siti-nabila/rest-orc/internal/features/roles/usecase"
	"github.com/siti-nabila/rest-orc/internal/response"
)

func registerRolesRoutes(routes routeGroups, client *authservice.Client, writer *response.Writer) error {
	g, err := gateway.New(client)
	if err != nil {
		return err
	}
	list, err := usecase.NewList(g)
	if err != nil {
		return err
	}
	get, err := usecase.NewGet(g)
	if err != nil {
		return err
	}
	h, err := handler.New(list, get, writer)
	if err != nil {
		return err
	}
	h.RegisterRoutes(routes.router, routes.authentication)
	return nil
}
