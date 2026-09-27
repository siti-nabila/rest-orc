package app

import (
	"github.com/gofiber/fiber/v3"
	requestauth "github.com/siti-nabila/rest-orc/internal/auth"
	"github.com/siti-nabila/rest-orc/internal/client/grpcauth"
	"github.com/siti-nabila/rest-orc/internal/middleware/authorization"
	"github.com/siti-nabila/rest-orc/internal/response"
)

type routeGroups struct {
	router         fiber.Router
	authentication fiber.Handler
}

type routeDependencies struct {
	authClient     *grpcauth.Client
	responseWriter *response.Writer
}

func newRouteGroups(
	router fiber.Router,
	authentication fiber.Handler,
) routeGroups {
	return routeGroups{
		router:         router,
		authentication: authentication,
	}
}

func (routes routeGroups) admin(prefix string) fiber.Router {
	return routes.router.Group(
		prefix,
		routes.authentication,
		authorization.RequireRole(requestauth.RoleAdmin),
	)
}

func (routes routeGroups) public(prefix string) fiber.Router {
	return routes.router.Group(prefix)
}

func registerFeatureRoutes(
	routes routeGroups,
	dependencies routeDependencies,
) error {
	if err := registerUsersRoutes(
		routes,
		dependencies.authClient,
		dependencies.responseWriter,
	); err != nil {
		return err
	}

	return nil
}
