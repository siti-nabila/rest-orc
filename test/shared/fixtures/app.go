package fixtures

import (
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	requestauth "github.com/siti-nabila/rest-orc/internal/auth"
	"github.com/siti-nabila/rest-orc/internal/client/authservice"
	authgateway "github.com/siti-nabila/rest-orc/internal/features/auth/gateway/authservice"
	authhandler "github.com/siti-nabila/rest-orc/internal/features/auth/handler"
	authusecase "github.com/siti-nabila/rest-orc/internal/features/auth/usecase"
	usersgateway "github.com/siti-nabila/rest-orc/internal/features/users/gateway/authservice"
	usershandler "github.com/siti-nabila/rest-orc/internal/features/users/handler"
	usersusecase "github.com/siti-nabila/rest-orc/internal/features/users/usecase"
	authmiddleware "github.com/siti-nabila/rest-orc/internal/middleware/auth"
	"github.com/siti-nabila/rest-orc/internal/middleware/authorization"
	"github.com/siti-nabila/rest-orc/internal/middleware/browsercors"
	localemiddleware "github.com/siti-nabila/rest-orc/internal/middleware/locale"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

// NewApp shares auth and users HTTP pipelines between fake-service and TLS wire tests.
func NewApp(t *testing.T, authClient *authservice.Client, before ...fiber.Handler) *fiber.App {
	t.Helper()
	authenticator, err := requestauth.NewAuthenticator(authClient)
	if err != nil {
		t.Fatalf("NewAuthenticator() error = %v", err)
	}
	authentication, err := authmiddleware.New(authenticator)
	if err != nil {
		t.Fatalf("auth middleware New() error = %v", err)
	}
	gateway, err := usersgateway.New(authClient)
	if err != nil {
		t.Fatalf("users gateway New() error = %v", err)
	}
	listUsers, err := usersusecase.NewList(gateway)
	if err != nil {
		t.Fatalf("NewList() error = %v", err)
	}
	authGateway, err := authgateway.New(authClient)
	if err != nil {
		t.Fatal(err)
	}
	register, err := authusecase.NewRegister(authGateway)
	if err != nil {
		t.Fatal(err)
	}
	login, err := authusecase.NewLogin(authGateway)
	if err != nil {
		t.Fatal(err)
	}
	// Avoid fasthttp's process-wide date ticker inside a synctest bubble.
	app, writer := testutils.NewHTTPApp(fiber.Config{DisableDefaultDate: true})
	for _, middleware := range before {
		app.Use(middleware)
	}
	profileGateway, err := usersgateway.NewProfile(authClient)
	if err != nil {
		t.Fatal(err)
	}
	updateProfile, err := usersusecase.NewUpdateProfile(profileGateway)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := usershandler.New(listUsers, updateProfile, writer)
	if err != nil {
		t.Fatalf("users HTTP New() error = %v", err)
	}
	browser := NewBrowserPolicy(t)
	v1 := app.Group("/api/v1", authhandler.LifecycleNoStore("/api/v1"), browsercors.New(BrowserConfig().AllowedOrigins), localemiddleware.Handle)
	passwordGateway, err := authgateway.NewPassword(authClient)
	if err != nil {
		t.Fatal(err)
	}
	changePassword, err := authusecase.NewChangePassword(passwordGateway)
	if err != nil {
		t.Fatal(err)
	}
	authHandler, err := authhandler.New(register, login, changePassword, writer, browser)
	if err != nil {
		t.Fatalf("auth HTTP New() error = %v", err)
	}
	sessionGateway, err := authgateway.NewSession(authClient)
	if err != nil {
		t.Fatal(err)
	}
	refresh, err := authusecase.NewRefresh(sessionGateway)
	if err != nil {
		t.Fatal(err)
	}
	logout, err := authusecase.NewLogout(sessionGateway)
	if err != nil {
		t.Fatal(err)
	}
	sessionHandler, err := authhandler.NewSession(refresh, logout, writer, browser)
	if err != nil {
		t.Fatal(err)
	}
	sessionHandler.RegisterRoutes(v1.Group(authhandler.SessionRoute))
	authHandler.RegisterPublicRoutes(v1.Group(authhandler.Route))
	meRoutes := v1.Group(authhandler.MeRoute, authentication.Handle)
	authHandler.RegisterAuthenticatedRoutes(meRoutes)
	myGateway, err := authgateway.NewMySessions(authClient)
	if err != nil {
		t.Fatal(err)
	}
	myList, err := authusecase.NewListMySessions(myGateway)
	if err != nil {
		t.Fatal(err)
	}
	myRevoke, err := authusecase.NewRevokeMySession(myGateway)
	if err != nil {
		t.Fatal(err)
	}
	myAll, err := authusecase.NewRevokeAllMySessions(myGateway)
	if err != nil {
		t.Fatal(err)
	}
	myHandler, err := authhandler.NewMySessions(myList, myRevoke, myAll, writer, browser)
	if err != nil {
		t.Fatal(err)
	}
	myHandler.RegisterRoutes(meRoutes)
	handler.RegisterAuthenticatedRoutes(meRoutes)
	admin := v1.Group(
		usershandler.Route,
		authentication.Handle,
		authorization.RequireRole(requestauth.RoleAdmin),
	)
	handler.RegisterAdminRoutes(admin)
	return app
}

func NewBrowserPolicy(t *testing.T) *authhandler.BrowserPolicy {
	t.Helper()
	p, err := authhandler.NewBrowserPolicy(BrowserConfig(), "/api/v1/auth", SessionNow, strings.NewReader(strings.Repeat("n", 1<<20)))
	if err != nil {
		t.Fatal(err)
	}
	return p
}
