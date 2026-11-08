package app

import (
	"github.com/siti-nabila/rest-orc/internal/client/authservice"
	gateway "github.com/siti-nabila/rest-orc/internal/features/applications/gateway/authservice"
	"github.com/siti-nabila/rest-orc/internal/features/applications/handler"
	"github.com/siti-nabila/rest-orc/internal/features/applications/usecase"
	"github.com/siti-nabila/rest-orc/internal/response"
)

func registerApplicationsRoutes(routes routeGroups, client *authservice.Client, writer *response.Writer) error {
	g, err := gateway.New(client)
	if err != nil {
		return err
	}
	list, err := usecase.NewList(g)
	if err != nil {
		return err
	}
	h, err := handler.New(list, writer)
	if err != nil {
		return err
	}
	h.RegisterRoutes(routes.router, routes.authentication)
	return nil
}
