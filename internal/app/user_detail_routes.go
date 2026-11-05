package app

import (
	"github.com/siti-nabila/rest-orc/internal/client/authservice"
	gateway "github.com/siti-nabila/rest-orc/internal/features/users/gateway/authservice"
	"github.com/siti-nabila/rest-orc/internal/features/users/handler"
	"github.com/siti-nabila/rest-orc/internal/features/users/usecase"
	"github.com/siti-nabila/rest-orc/internal/response"
)

func registerUserDetailRoute(routes routeGroups, client *authservice.Client, writer *response.Writer) error {
	detailGateway, err := gateway.NewUserDetail(client)
	if err != nil {
		return err
	}
	getUser, err := usecase.NewGetUser(detailGateway)
	if err != nil {
		return err
	}
	detailHandler, err := handler.NewUserDetail(getUser, writer)
	if err != nil {
		return err
	}
	detailHandler.RegisterRoutes(routes.router, routes.authentication)
	return nil
}
