package app

import (
	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/rest-orc/internal/client/authservice"
	usersgateway "github.com/siti-nabila/rest-orc/internal/features/users/gateway/authservice"
	usershandler "github.com/siti-nabila/rest-orc/internal/features/users/handler"
	usersusecase "github.com/siti-nabila/rest-orc/internal/features/users/usecase"
	"github.com/siti-nabila/rest-orc/internal/response"
)

func registerUsersRoutes(
	routes routeGroups,
	meRoutes fiber.Router,
	authClient *authservice.Client,
	responseWriter *response.Writer,
) error {
	gateway, err := usersgateway.New(authClient)
	if err != nil {
		return err
	}
	listUsers, err := usersusecase.NewList(gateway)
	if err != nil {
		return err
	}
	profileGateway, err := usersgateway.NewProfile(authClient)
	if err != nil {
		return err
	}
	updateProfile, err := usersusecase.NewUpdateProfile(profileGateway)
	if err != nil {
		return err
	}
	handler, err := usershandler.New(
		listUsers,
		updateProfile,
		responseWriter,
	)
	if err != nil {
		return err
	}

	handler.RegisterAuthenticatedRoutes(meRoutes)
	handler.RegisterAdminRoutes(routes.admin(usershandler.Route))
	return nil
}
