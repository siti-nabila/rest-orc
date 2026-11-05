package app

import (
	"github.com/siti-nabila/rest-orc/internal/client/authservice"
	gateway "github.com/siti-nabila/rest-orc/internal/features/users/gateway/authservice"
	"github.com/siti-nabila/rest-orc/internal/features/users/handler"
	"github.com/siti-nabila/rest-orc/internal/features/users/usecase"
	"github.com/siti-nabila/rest-orc/internal/response"
)

func registerUserProfileRoute(routes routeGroups, client *authservice.Client, writer *response.Writer) error {
	profileGateway, err := gateway.NewUserProfile(client)
	if err != nil {
		return err
	}
	updateProfile, err := usecase.NewUpdateUserProfile(profileGateway)
	if err != nil {
		return err
	}
	profileHandler, err := handler.NewUserProfile(updateProfile, writer)
	if err != nil {
		return err
	}
	profileHandler.RegisterRoutes(routes.router, routes.authentication)
	return nil
}
