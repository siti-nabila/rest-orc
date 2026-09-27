package app

import (
	"context"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/rest-orc/internal/client/grpcclient"
	"github.com/siti-nabila/rest-orc/internal/config"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

type App struct {
	HTTPServer     *fiber.App
	AuthConnection *grpcclient.Client
	// BackendClient  *httpclient.Client

	listenAddress   string
	shutdownTimeout config.Duration
}

func New(cfg *config.Config) (*App, error) {
	if cfg == nil {
		return nil, dictionary.ErrApplicationConfigurationRequired
	}
	if err := cfg.Validate(); err != nil {
		return nil, dictionary.CreateApplication(err)
	}

	authConnection, err := grpcclient.New(cfg.Clients.AuthGRPC)
	if err != nil {
		return nil, dictionary.CreateApplication(err)
	}

	// backendClient, err := httpclient.New(cfg.Clients.BackendHTTP)
	// if err != nil {
	// 	closeErr := authConnection.Close()
	// 	return nil, errors.Join(
	// 		dictionary.CreateApplication(err),
	// 		closeErr,
	// 	)
	// }

	httpServer, err := newHTTPServer(cfg, authConnection)
	if err != nil {
		// backendClient.Close()
		closeErr := authConnection.Close()
		return nil, errors.Join(
			dictionary.CreateApplication(err),
			closeErr,
		)
	}

	return &App{
		HTTPServer:     httpServer,
		AuthConnection: authConnection,
		// BackendClient:   backendClient,
		listenAddress:   ":" + strconv.Itoa(cfg.App.Port),
		shutdownTimeout: cfg.Server.ShutdownTimeout,
	}, nil
}

func (application *App) Listen(ctx context.Context) error {
	if ctx == nil {
		return dictionary.ErrListenContextRequired
	}

	if err := application.HTTPServer.Listen(
		application.listenAddress,
		fiber.ListenConfig{
			GracefulContext:       ctx,
			ShutdownTimeout:       application.shutdownTimeout.Duration,
			DisableStartupMessage: true,
		},
	); err != nil {
		return dictionary.ListenHTTPServer(application.listenAddress, err)
	}
	return nil
}

func (application *App) Close() error {
	var closeErrors []error

	if err := application.HTTPServer.ShutdownWithTimeout(
		application.shutdownTimeout.Duration,
	); err != nil && !errors.Is(err, fiber.ErrNotRunning) {
		closeErrors = append(closeErrors, dictionary.ShutdownHTTPServer(err))
	}

	// application.BackendClient.Close()
	if err := application.AuthConnection.Close(); err != nil {
		closeErrors = append(closeErrors, dictionary.CloseAuthGRPCConnection(err))
	}

	if err := errors.Join(closeErrors...); err != nil {
		return dictionary.CloseApplication(err)
	}
	return nil
}
