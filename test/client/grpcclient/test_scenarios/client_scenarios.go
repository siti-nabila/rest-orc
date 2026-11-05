package test_scenarios

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/siti-nabila/rest-orc/internal/client/grpcclient"
	"github.com/siti-nabila/rest-orc/internal/config"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/test/client/grpcclient/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc/connectivity"
)

func All() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "service token requires canonical encoding and secure transport", Run: serviceCredentialConfiguration},
		{
			Name: "disabled keepalive creates a closable connection",
			Run:  disabledKeepaliveCreatesClosableConnection,
		},
		{
			Name: "enabled keepalive creates a closable connection",
			Run:  enabledKeepaliveCreatesClosableConnection,
		},
		{
			Name: "same constructor creates independent connections for multiple services",
			Run:  constructorSupportsMultipleServices,
		},
		{
			Name: "request context propagates parent cancellation",
			Run:  requestContextPropagatesParentCancellation,
		},
		{
			Name: "unreadable TLS CA returns dependency context",
			Run:  unreadableCAReturnsDependencyContext,
		},
	}
}

func disabledKeepaliveCreatesClosableConnection(t *testing.T) {
	assertClosableConnection(t, fixtures.ConfigWithKeepalive(false))
}

func enabledKeepaliveCreatesClosableConnection(t *testing.T) {
	assertClosableConnection(t, fixtures.ConfigWithKeepalive(true))
}

func requestContextPropagatesParentCancellation(t *testing.T) {
	// Arrange
	client, err := grpcclient.New(fixtures.ValidConfig())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
	parent, cancelParent := context.WithCancel(context.Background())
	ctx, cancelRequest := client.RequestContext(parent)
	defer cancelRequest()

	// Act
	cancelParent()

	// Assert
	select {
	case <-ctx.Done():
		if ctx.Err() != context.Canceled {
			t.Errorf("RequestContext error = %v, want context.Canceled", ctx.Err())
		}
	default:
		t.Fatal("RequestContext was not canceled with its parent")
	}
}

func unreadableCAReturnsDependencyContext(t *testing.T) {
	// Arrange
	cfg := fixtures.ConfigWithUnreadableCA()

	// Act
	_, err := grpcclient.New(cfg)

	// Assert
	if err == nil {
		t.Fatal("New() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "read TLS CA file") {
		t.Errorf("New() error = %q, want TLS CA context", err)
	}
}

func constructorSupportsMultipleServices(t *testing.T) {
	// Arrange
	authConfig := fixtures.ConfigWithTarget("passthrough:///auth-proxy:50051")
	orderConfig := fixtures.ConfigWithTarget("passthrough:///order-proxy:50052")

	// Act
	authClient, authErr := grpcclient.New(authConfig)
	if authErr != nil {
		t.Fatalf("New(auth config) error = %v", authErr)
	}
	t.Cleanup(func() {
		if err := authClient.Close(); err != nil {
			t.Errorf("close auth connection: %v", err)
		}
	})

	orderClient, orderErr := grpcclient.New(orderConfig)
	if orderErr != nil {
		t.Fatalf("New(order config) error = %v", orderErr)
	}
	t.Cleanup(func() {
		if err := orderClient.Close(); err != nil {
			t.Errorf("close order connection: %v", err)
		}
	})

	// Assert
	if authClient.Connection() == orderClient.Connection() {
		t.Error("different service targets share the same connection")
	}
}

func assertClosableConnection(t *testing.T, cfg config.GRPCClientConfig) {
	t.Helper()

	// Arrange
	client, err := grpcclient.New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Act
	connection := client.Connection()
	closeErr := client.Close()

	// Assert
	if connection == nil {
		t.Fatal("Connection() = nil")
	}
	if closeErr != nil {
		t.Fatalf("Close() error = %v", closeErr)
	}
	stateReader, ok := connection.(interface {
		GetState() connectivity.State
	})
	if !ok {
		t.Fatal("Connection() does not expose connectivity state")
	}
	if stateReader.GetState() != connectivity.Shutdown {
		t.Errorf("connection state = %s, want SHUTDOWN", stateReader.GetState())
	}
}

func serviceCredentialConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name, token string
		tls         bool
		problem     error
	}{
		{"malformed", "private-service-sentinel", true, dictionary.ErrInvalidServiceToken},
		{"insecure", base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{17}, 32)), false, dictionary.ErrServiceCredentialsRequireTLS},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := fixtures.ValidConfig()
			cfg.ServiceToken = tc.token
			cfg.TLS.Enabled = tc.tls
			client, err := grpcclient.New(cfg)
			if client != nil || !errors.Is(err, tc.problem) {
				t.Fatal("constructor accepted invalid service credential configuration")
			}
			if strings.Contains(err.Error(), tc.token) {
				t.Error("constructor exposed credential")
			}
		})
	}
	cfg := fixtures.ValidConfig()
	cfg.TLS.Enabled = true
	cfg.ServiceToken = base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{17}, 32))
	assertClosableConnection(t, cfg)
}
