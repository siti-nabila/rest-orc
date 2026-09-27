package app

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/healthcheck"
	recovermw "github.com/gofiber/fiber/v3/middleware/recover"
	authdictionary "github.com/siti-nabila/api-contracts/pkg/dictionary/auth"
	commondictionary "github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	requestauth "github.com/siti-nabila/rest-orc/internal/auth"
	"github.com/siti-nabila/rest-orc/internal/client/grpcauth"
	"github.com/siti-nabila/rest-orc/internal/client/grpcclient"
	"github.com/siti-nabila/rest-orc/internal/config"
	authmiddleware "github.com/siti-nabila/rest-orc/internal/middleware/auth"
	localemiddleware "github.com/siti-nabila/rest-orc/internal/middleware/locale"
	"github.com/siti-nabila/rest-orc/internal/response"
)

const authGRPCClientName = "auth_grpc"

func newHTTPServer(
	cfg *config.Config,
	authConnection *grpcclient.Client,
) (*fiber.App, error) {
	responseWriter := response.NewWriter(response.NewErrorMapper(
		authdictionary.Registry(),
		commondictionary.Registry(),
	))
	authClient, err := grpcauth.New(authConnection)
	if err != nil {
		return nil, err
	}
	authenticator, err := requestauth.NewAuthenticator(authClient)
	if err != nil {
		return nil, err
	}
	authentication, err := authmiddleware.New(authenticator)
	if err != nil {
		return nil, err
	}
	httpServer := fiber.New(fiber.Config{
		AppName:      cfg.App.Name,
		ReadTimeout:  cfg.Server.ReadTimeout.Duration,
		WriteTimeout: cfg.Server.WriteTimeout.Duration,
		IdleTimeout:  cfg.Server.IdleTimeout.Duration,
		ErrorHandler: func(ctx fiber.Ctx, err error) error {
			return responseWriter.Write(ctx, response.Result{Err: err})
		},
	})
	httpServer.Hooks().OnListen(func(data fiber.ListenData) error {
		fmt.Println("🟢 " + FormatListenMessage(data))
		fmt.Println("\t🌐 " + FormatClientMessage(
			authGRPCClientName,
			cfg.Clients.AuthGRPC.Target,
		))
		return nil
	})

	httpServer.Use(recovermw.New(recovermw.Config{
		EnableStackTrace: false,
		PanicHandler: func(fiber.Ctx, any) error {
			return commondictionary.ErrInternalServerError
		},
	}))
	httpServer.Get(
		healthcheck.LivenessEndpoint,
		healthcheck.New(healthcheck.Config{ResponseFormat: healthcheck.FormatJSON}),
	)

	v1 := httpServer.Group("/api/v1", localemiddleware.Handle)
	routes := newRouteGroups(v1, authentication.Handle)
	if err := registerFeatureRoutes(routes, routeDependencies{
		authClient:     authClient,
		responseWriter: responseWriter,
	}); err != nil {
		return nil, err
	}

	return httpServer, nil
}

func FormatClientMessage(name, target string) string {
	message := fmt.Sprintf("[CLIENT]: %s target: %s", name, target)
	if port := targetPort(target); port != "" {
		return fmt.Sprintf("%s (port %s)", message, port)
	}
	return message
}

func targetPort(target string) string {
	endpoint := target
	if parsed, err := url.Parse(target); err == nil {
		switch {
		case parsed.Host != "":
			endpoint = parsed.Host
		case parsed.Path != "":
			endpoint = strings.TrimPrefix(parsed.Path, "/")
		}
	}

	_, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		return ""
	}
	return port
}

func FormatListenMessage(data fiber.ListenData) string {
	scheme := "http"
	if data.TLS {
		scheme = "https"
	}
	applicationURL := fmt.Sprintf(
		"%s://%s:%s",
		scheme,
		data.Host,
		data.Port,
	)
	return fmt.Sprintf(
		"[%s] is running at %s (port %s)",
		strings.ToUpper(data.AppName),
		applicationURL,
		data.Port,
	)
}
