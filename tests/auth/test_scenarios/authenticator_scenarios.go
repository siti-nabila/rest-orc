package test_scenarios

import (
	"context"
	"errors"
	"reflect"
	"testing"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	requestauth "github.com/siti-nabila/rest-orc/internal/auth"
	"github.com/siti-nabila/rest-orc/tests/auth/fixtures"
	"github.com/siti-nabila/rest-orc/tests/auth/mocks"
	"github.com/siti-nabila/rest-orc/tests/shared/testutils"
	"google.golang.org/grpc/metadata"
)

type contextKey string

const requestIDKey contextKey = "request-id"

var errDependency = errors.New("Me dependency failed")

func All() []testutils.Scenario {
	return []testutils.Scenario{
		{
			Name: "Authenticate forwards authorization and stores principal",
			Run:  authenticateStoresPrincipal,
		},
		{
			Name: "Authenticate rejects missing authorization before Me call",
			Run:  authenticateRejectsMissingAuthorization,
		},
		{
			Name: "Authenticate preserves Me dependency error",
			Run:  authenticatePreservesDependencyError,
		},
		{
			Name: "Authenticate rejects nil context before Me call",
			Run:  authenticateRejectsNilContext,
		},
		{
			Name: "authenticator constructor rejects nil client",
			Run:  constructorRejectsNilClient,
		},
	}
}

func authenticateStoresPrincipal(t *testing.T) {
	// Arrange
	user := fixtures.UserData()
	client := &mocks.MeClient{
		MeFunc: func(ctx context.Context) (*userv1.UserData, error) {
			if ctx.Value(requestIDKey) != "request-17" {
				t.Errorf("request ID = %v, want request-17", ctx.Value(requestIDKey))
			}
			outgoing, exists := metadata.FromOutgoingContext(ctx)
			if !exists {
				t.Fatal("outgoing metadata is missing")
			}
			if actual := outgoing.Get("authorization"); len(actual) != 1 || actual[0] != "Bearer example-token" {
				t.Errorf("authorization metadata = %v, want bearer token", actual)
			}
			if actual := outgoing.Get("x-request-id"); len(actual) != 1 || actual[0] != "request-17" {
				t.Errorf("existing metadata = %v, want request ID", actual)
			}
			return user, nil
		},
	}
	authenticator := newAuthenticator(t, client)
	ctx := context.WithValue(context.Background(), requestIDKey, "request-17")
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("x-request-id", "request-17"))

	// Act
	authenticatedContext, err := authenticator.Authenticate(
		ctx,
		"Bearer example-token",
	)

	// Assert
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	principal, exists := requestauth.PrincipalFromContext(authenticatedContext)
	if !exists {
		t.Fatal("principal is missing from authenticated context")
	}
	if !reflect.DeepEqual(principal, fixtures.Principal()) {
		t.Errorf("principal = %#v, want %#v", principal, fixtures.Principal())
	}
	if client.Calls != 1 {
		t.Errorf("Me calls = %d, want 1", client.Calls)
	}
}

func authenticateRejectsMissingAuthorization(t *testing.T) {
	// Arrange
	client := failOnMeClient(t)
	authenticator := newAuthenticator(t, client)

	// Act
	_, err := authenticator.Authenticate(context.Background(), "  ")

	// Assert
	if err == nil {
		t.Fatal("Authenticate() error = nil, want unauthorized error")
	}
	if client.Calls != 0 {
		t.Errorf("Me calls = %d, want 0", client.Calls)
	}
}

func authenticatePreservesDependencyError(t *testing.T) {
	// Arrange
	client := &mocks.MeClient{
		MeFunc: func(context.Context) (*userv1.UserData, error) {
			return nil, errDependency
		},
	}
	authenticator := newAuthenticator(t, client)

	// Act
	_, err := authenticator.Authenticate(
		context.Background(),
		"Bearer example-token",
	)

	// Assert
	if !errors.Is(err, errDependency) {
		t.Errorf("Authenticate() error = %v, want dependency error", err)
	}
}

func authenticateRejectsNilContext(t *testing.T) {
	// Arrange
	client := failOnMeClient(t)
	authenticator := newAuthenticator(t, client)

	// Act
	_, err := authenticator.Authenticate(nil, "Bearer example-token")

	// Assert
	if err == nil {
		t.Fatal("Authenticate(nil) error = nil, want error")
	}
	if client.Calls != 0 {
		t.Errorf("Me calls = %d, want 0", client.Calls)
	}
}

func constructorRejectsNilClient(t *testing.T) {
	// Act
	_, err := requestauth.NewAuthenticator(nil)

	// Assert
	if err == nil {
		t.Fatal("NewAuthenticator(nil) error = nil, want error")
	}
}

func newAuthenticator(
	t *testing.T,
	client requestauth.MeClient,
) *requestauth.Authenticator {
	t.Helper()

	authenticator, err := requestauth.NewAuthenticator(client)
	if err != nil {
		t.Fatalf("NewAuthenticator() error = %v", err)
	}
	return authenticator
}

func failOnMeClient(t *testing.T) *mocks.MeClient {
	t.Helper()

	return &mocks.MeClient{
		MeFunc: func(context.Context) (*userv1.UserData, error) {
			t.Fatal("unexpected Me dependency call")
			return nil, nil
		},
	}
}
