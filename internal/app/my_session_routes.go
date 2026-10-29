package app

import (
	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/rest-orc/internal/client/authservice"
	authgateway "github.com/siti-nabila/rest-orc/internal/features/auth/gateway/authservice"
	authhandler "github.com/siti-nabila/rest-orc/internal/features/auth/handler"
	authusecase "github.com/siti-nabila/rest-orc/internal/features/auth/usecase"
	"github.com/siti-nabila/rest-orc/internal/response"
)

func registerMySessionRoutes(meRoutes fiber.Router, client *authservice.Client, writer *response.Writer, browser *authhandler.BrowserPolicy) error {
	gateway, err := authgateway.NewMySessions(client)
	if err != nil {
		return err
	}
	list, err := authusecase.NewListMySessions(gateway)
	if err != nil {
		return err
	}
	revoke, err := authusecase.NewRevokeMySession(gateway)
	if err != nil {
		return err
	}
	all, err := authusecase.NewRevokeAllMySessions(gateway)
	if err != nil {
		return err
	}
	handler, err := authhandler.NewMySessions(list, revoke, all, writer, browser)
	if err != nil {
		return err
	}
	handler.RegisterRoutes(meRoutes)
	return nil
}
