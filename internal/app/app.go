package app

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/rest-orc/internal/client/grpcclient"
	"github.com/siti-nabila/rest-orc/internal/client/httpclient"
	"github.com/siti-nabila/rest-orc/internal/config"
)

type App struct {
	HTTPServer     *fiber.App
	AuthConnection *grpcclient.Client
	BackendClient  *httpclient.Client

	listenAddress   string
	shutdownTimeout config.Duration
}

func New(cfg *config.Config) (*App, error) {
	if cfg == nil {
		return nil, fmt.Errorf("create application: configuration must not be nil")
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("create application: %w", err)
	}

	authConnection, err := grpcclient.New(cfg.Clients.AuthGRPC)
	if err != nil {
		return nil, fmt.Errorf("create application: %w", err)
	}

	backendClient, err := httpclient.New(cfg.Clients.BackendHTTP)
	if err != nil {
		closeErr := authConnection.Close()
		return nil, errors.Join(
			fmt.Errorf("create application: %w", err),
			closeErr,
		)
	}

	return &App{
		HTTPServer:      newHTTPServer(cfg),
		AuthConnection:  authConnection,
		BackendClient:   backendClient,
		listenAddress:   ":" + strconv.Itoa(cfg.App.Port),
		shutdownTimeout: cfg.Server.ShutdownTimeout,
	}, nil
}

func (application *App) Listen(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("listen HTTP server: context must not be nil")
	}

	if err := application.HTTPServer.Listen(
		application.listenAddress,
		fiber.ListenConfig{
			GracefulContext:       ctx,
			ShutdownTimeout:       application.shutdownTimeout.Duration,
			DisableStartupMessage: true,
		},
	); err != nil {
		return fmt.Errorf("listen HTTP server on %s: %w", application.listenAddress, err)
	}
	return nil
}

func (application *App) Close() error {
	var closeErrors []error

	if err := application.HTTPServer.ShutdownWithTimeout(
		application.shutdownTimeout.Duration,
	); err != nil && !errors.Is(err, fiber.ErrNotRunning) {
		closeErrors = append(closeErrors, fmt.Errorf("shutdown HTTP server: %w", err))
	}

	application.BackendClient.Close()
	if err := application.AuthConnection.Close(); err != nil {
		closeErrors = append(closeErrors, fmt.Errorf("close auth gRPC connection: %w", err))
	}

	if err := errors.Join(closeErrors...); err != nil {
		return fmt.Errorf("close application: %w", err)
	}
	return nil
}
