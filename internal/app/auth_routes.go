package app

import (
	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/rest-orc/internal/client/authservice"
	authgateway "github.com/siti-nabila/rest-orc/internal/features/auth/gateway/authservice"
	authhandler "github.com/siti-nabila/rest-orc/internal/features/auth/handler"
	authusecase "github.com/siti-nabila/rest-orc/internal/features/auth/usecase"
	"github.com/siti-nabila/rest-orc/internal/response"
)

func registerAuthRoutes(routes routeGroups, meRoutes fiber.Router, authClient *authservice.Client, writer *response.Writer, browser *authhandler.BrowserPolicy) error {
	gateway, err := authgateway.New(authClient)
	if err != nil {
		return err
	}
	register, err := authusecase.NewRegister(gateway)
	if err != nil {
		return err
	}
	login, err := authusecase.NewLogin(gateway)
	if err != nil {
		return err
	}
	passwordGateway, err := authgateway.NewPassword(authClient)
	if err != nil {
		return err
	}
	changePassword, err := authusecase.NewChangePassword(passwordGateway)
	if err != nil {
		return err
	}
	handler, err := authhandler.New(register, login, changePassword, writer, browser)
	if err != nil {
		return err
	}
	// Register public auth before the protected users group on the same prefix.
	handler.RegisterPublicRoutes(routes.public(authhandler.Route))
	handler.RegisterAuthenticatedRoutes(meRoutes)
	sessionGateway, err := authgateway.NewSession(authClient)
	if err != nil {
		return err
	}
	refresh, err := authusecase.NewRefresh(sessionGateway)
	if err != nil {
		return err
	}
	logout, err := authusecase.NewLogout(sessionGateway)
	if err != nil {
		return err
	}
	sessionHandler, err := authhandler.NewSession(refresh, logout, writer, browser)
	if err != nil {
		return err
	}
	sessionHandler.RegisterRoutes(routes.public(authhandler.SessionRoute))
	return registerMySessionRoutes(meRoutes, authClient, writer, browser)
}
