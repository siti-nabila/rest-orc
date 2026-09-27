package app

import (
	"github.com/siti-nabila/rest-orc/internal/client/grpcauth"
	usershttp "github.com/siti-nabila/rest-orc/internal/features/users/handler/http"
	usersrepo "github.com/siti-nabila/rest-orc/internal/features/users/repo/grpcauth"
	usersusecase "github.com/siti-nabila/rest-orc/internal/features/users/usecase"
	"github.com/siti-nabila/rest-orc/internal/response"
)

func registerUsersRoutes(
	routes routeGroups,
	authClient *grpcauth.Client,
	responseWriter *response.Writer,
) error {
	repository, err := usersrepo.New(authClient)
	if err != nil {
		return err
	}
	listUsers, err := usersusecase.NewList(repository)
	if err != nil {
		return err
	}
	register, err := usersusecase.NewRegister(repository)
	if err != nil {
		return err
	}
	login, err := usersusecase.NewLogin(repository)
	if err != nil {
		return err
	}
	handler, err := usershttp.New(
		listUsers,
		register,
		login,
		responseWriter,
	)
	if err != nil {
		return err
	}

	// Register public endpoints before the admin middleware group with the same
	// prefix, otherwise Fiber would run the admin middleware first.
	handler.RegisterPublicRoutes(routes.public(usershttp.Route))
	handler.RegisterAdminRoutes(routes.admin(usershttp.Route))
	return nil
}
