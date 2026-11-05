package app

import (
	"github.com/siti-nabila/rest-orc/internal/client/authservice"
	gateway "github.com/siti-nabila/rest-orc/internal/features/users/gateway/authservice"
	"github.com/siti-nabila/rest-orc/internal/features/users/handler"
	"github.com/siti-nabila/rest-orc/internal/features/users/usecase"
	"github.com/siti-nabila/rest-orc/internal/response"
)

func registerUserStatusRoute(routes routeGroups, client *authservice.Client, writer *response.Writer) error {
	statusGateway, err := gateway.NewUserStatus(client)
	if err != nil {
		return err
	}
	changeStatus, err := usecase.NewChangeUserStatus(statusGateway)
	if err != nil {
		return err
	}
	statusHandler, err := handler.NewUserStatus(changeStatus, writer)
	if err != nil {
		return err
	}
	statusHandler.RegisterRoutes(routes.router, routes.authentication)
	return nil
}
