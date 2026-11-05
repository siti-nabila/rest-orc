package app

import (
	"github.com/siti-nabila/rest-orc/internal/client/authservice"
	gateway "github.com/siti-nabila/rest-orc/internal/features/users/gateway/authservice"
	"github.com/siti-nabila/rest-orc/internal/features/users/handler"
	"github.com/siti-nabila/rest-orc/internal/features/users/usecase"
	"github.com/siti-nabila/rest-orc/internal/response"
)

func registerUserCreationRoute(routes routeGroups, client *authservice.Client, writer *response.Writer) error {
	creationGateway, err := gateway.NewUserCreation(client)
	if err != nil {
		return err
	}
	createUser, err := usecase.NewCreateUser(creationGateway)
	if err != nil {
		return err
	}
	creationHandler, err := handler.NewUserCreation(createUser, writer)
	if err != nil {
		return err
	}
	creationHandler.RegisterRoutes(routes.router, routes.authentication)
	return nil
}
