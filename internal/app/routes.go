package app

import (
	"github.com/gofiber/fiber/v3"
	requestauth "github.com/siti-nabila/rest-orc/internal/auth"
	"github.com/siti-nabila/rest-orc/internal/client/authservice"
	authhandler "github.com/siti-nabila/rest-orc/internal/features/auth/handler"
	"github.com/siti-nabila/rest-orc/internal/middleware/authorization"
	"github.com/siti-nabila/rest-orc/internal/response"
)

type routeGroups struct {
	router         fiber.Router
	authentication fiber.Handler
}

type routeDependencies struct {
	authClient     *authservice.Client
	responseWriter *response.Writer
	browser        *authhandler.BrowserPolicy
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

func (routes routeGroups) authenticated(prefix string) fiber.Router {
	return routes.router.Group(prefix, routes.authentication)
}

func registerFeatureRoutes(
	routes routeGroups,
	dependencies routeDependencies,
) error {
	meRoutes := routes.authenticated(authhandler.MeRoute)
	if err := registerAuthRoutes(
		routes,
		meRoutes,
		dependencies.authClient,
		dependencies.responseWriter,
		dependencies.browser,
	); err != nil {
		return err
	}

	return registerUsersRoutes(routes, meRoutes, dependencies.authClient, dependencies.responseWriter)
}
