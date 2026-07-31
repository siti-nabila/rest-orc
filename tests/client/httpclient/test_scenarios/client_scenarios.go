package test_scenarios

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/siti-nabila/rest-orc/internal/client/httpclient"
	"github.com/siti-nabila/rest-orc/internal/config"
	"github.com/siti-nabila/rest-orc/tests/client/httpclient/fixtures"
	"github.com/siti-nabila/rest-orc/tests/client/httpclient/mocks"
	"github.com/siti-nabila/rest-orc/tests/shared/testutils"
)

const serviceURL = "http://service-proxy:8080"

func All() []testutils.Scenario {
	return []testutils.Scenario{
		{
			Name: "relative request URL is resolved against configured base URL",
			Run:  relativeURLIsResolved,
		},
		{
			Name: "one RoundTripper instance handles repeated requests",
			Run:  roundTripperIsReused,
		},
		{
			Name: "same constructor supports multiple HTTP services",
			Run:  constructorSupportsMultipleServices,
		},
		{
			Name: "request timeout adds a context deadline",
			Run:  requestTimeoutAddsDeadline,
		},
		{
			Name: "enabled HTTP keepalive configures connection reuse",
			Run:  enabledKeepaliveConfiguresTransport,
		},
		{
			Name: "disabled HTTP keepalive disables connection reuse",
			Run:  disabledKeepaliveConfiguresTransport,
		},
		{
			Name: "parent cancellation reaches the outbound request",
			Run:  parentCancellationReachesOutboundRequest,
		},
		{
			Name: "nil context is rejected before dependency call",
			Run:  nilContextIsRejected,
		},
		{
			Name: "nil request is rejected before dependency call",
			Run:  nilRequestIsRejected,
		},
		{
			Name: "request without URL is rejected before dependency call",
			Run:  requestWithoutURLIsRejected,
		},
		{
			Name: "nil RoundTripper dependency is rejected",
			Run:  nilRoundTripperIsRejected,
		},
		{
			Name: "TLS setting must match configured URL scheme",
			Run:  tlsMismatchIsRejected,
		},
		{
			Name: "client close releases idle transport connections",
			Run:  closeReleasesIdleConnections,
		},
	}
}

func relativeURLIsResolved(t *testing.T) {
	// Arrange
	var actualURL string
	transport := mocks.RoundTripperFunc(func(request *http.Request) (*http.Response, error) {
		actualURL = request.URL.String()
		return fixtures.Response(http.StatusOK), nil
	})
	client := newClient(t, fixtures.ValidConfig(serviceURL, true), transport)
	request := &http.Request{
		Method: http.MethodGet,
		URL:    &url.URL{Path: "/users"},
		Header: make(http.Header),
	}

	// Act
	response, err := client.Do(context.Background(), request)

	// Assert
	closeResponse(t, response, err)
	const expectedURL = serviceURL + "/users"
	if actualURL != expectedURL {
		t.Errorf("request URL = %q, want %q", actualURL, expectedURL)
	}
}

func roundTripperIsReused(t *testing.T) {
	// Arrange
	var calls atomic.Int32
	transport := mocks.RoundTripperFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return fixtures.Response(http.StatusOK), nil
	})
	client := newClient(t, fixtures.ValidConfig(serviceURL, true), transport)

	// Act
	for range 2 {
		request, err := http.NewRequest(http.MethodGet, serviceURL, nil)
		if err != nil {
			t.Fatalf("NewRequest() error = %v", err)
		}
		response, err := client.Do(context.Background(), request)
		closeResponse(t, response, err)
	}

	// Assert
	if got := calls.Load(); got != 2 {
		t.Errorf("RoundTripper calls = %d, want 2", got)
	}
}

func constructorSupportsMultipleServices(t *testing.T) {
	// Arrange
	var authRequestURL string
	authTransport := mocks.RoundTripperFunc(func(request *http.Request) (*http.Response, error) {
		authRequestURL = request.URL.String()
		return fixtures.Response(http.StatusOK), nil
	})
	authClient := newClient(
		t,
		fixtures.ValidConfig("http://auth-proxy:8080", true),
		authTransport,
	)

	var orderRequestURL string
	orderTransport := mocks.RoundTripperFunc(func(request *http.Request) (*http.Response, error) {
		orderRequestURL = request.URL.String()
		return fixtures.Response(http.StatusOK), nil
	})
	orderClient := newClient(
		t,
		fixtures.ValidConfig("http://order-proxy:8080", true),
		orderTransport,
	)
	request := &http.Request{
		Method: http.MethodGet,
		URL:    &url.URL{Path: "/health"},
		Header: make(http.Header),
	}

	// Act
	authResponse, authErr := authClient.Do(context.Background(), request)
	closeResponse(t, authResponse, authErr)
	orderResponse, orderErr := orderClient.Do(context.Background(), request)
	closeResponse(t, orderResponse, orderErr)

	// Assert
	if authRequestURL != "http://auth-proxy:8080/health" {
		t.Errorf("auth request URL = %q, want auth service URL", authRequestURL)
	}
	if orderRequestURL != "http://order-proxy:8080/health" {
		t.Errorf("order request URL = %q, want order service URL", orderRequestURL)
	}
}

func requestTimeoutAddsDeadline(t *testing.T) {
	// Arrange
	var hasDeadline bool
	transport := mocks.RoundTripperFunc(func(request *http.Request) (*http.Response, error) {
		_, hasDeadline = request.Context().Deadline()
		return fixtures.Response(http.StatusOK), nil
	})
	client := newClient(t, fixtures.ValidConfig(serviceURL, true), transport)
	request, err := http.NewRequest(http.MethodGet, serviceURL, nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	// Act
	response, callErr := client.Do(context.Background(), request)

	// Assert
	closeResponse(t, response, callErr)
	if !hasDeadline {
		t.Error("outbound request context has no deadline")
	}
}

func enabledKeepaliveConfiguresTransport(t *testing.T) {
	// Arrange
	cfg := fixtures.ValidConfig(serviceURL, true)

	// Act
	transport, err := httpclient.NewTransport(cfg)

	// Assert
	if err != nil {
		t.Fatalf("NewTransport() error = %v", err)
	}
	if transport.DisableKeepAlives {
		t.Error("DisableKeepAlives = true, want false")
	}
	if transport.MaxConnsPerHost != cfg.Keepalive.MaxConnectionsPerHost {
		t.Errorf(
			"MaxConnsPerHost = %d, want %d",
			transport.MaxConnsPerHost,
			cfg.Keepalive.MaxConnectionsPerHost,
		)
	}
}

func disabledKeepaliveConfiguresTransport(t *testing.T) {
	// Arrange
	cfg := fixtures.ValidConfig(serviceURL, false)

	// Act
	transport, err := httpclient.NewTransport(cfg)

	// Assert
	if err != nil {
		t.Fatalf("NewTransport() error = %v", err)
	}
	if !transport.DisableKeepAlives {
		t.Error("DisableKeepAlives = false, want true")
	}
}

func parentCancellationReachesOutboundRequest(t *testing.T) {
	// Arrange
	requestReceived := make(chan struct{})
	transport := mocks.RoundTripperFunc(func(request *http.Request) (*http.Response, error) {
		close(requestReceived)
		<-request.Context().Done()
		return nil, request.Context().Err()
	})
	client := newClient(t, fixtures.ValidConfig(serviceURL, true), transport)
	ctx, cancel := context.WithCancel(context.Background())
	request, err := http.NewRequest(http.MethodGet, serviceURL, nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	result := make(chan error, 1)

	// Act
	go func() {
		_, callErr := client.Do(ctx, request)
		result <- callErr
	}()
	<-requestReceived
	cancel()
	callErr := <-result

	// Assert
	if !errors.Is(callErr, context.Canceled) {
		t.Errorf("Do() error = %v, want context.Canceled", callErr)
	}
}

func nilContextIsRejected(t *testing.T) {
	// Arrange
	client := newClient(t, fixtures.ValidConfig(serviceURL, true), failOnCall(t))
	request, err := http.NewRequest(http.MethodGet, serviceURL, nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	// Act
	_, callErr := client.Do(nil, request)

	// Assert
	if callErr == nil {
		t.Fatal("Do(nil, request) error = nil, want error")
	}
}

func nilRequestIsRejected(t *testing.T) {
	// Arrange
	client := newClient(t, fixtures.ValidConfig(serviceURL, true), failOnCall(t))

	// Act
	_, err := client.Do(context.Background(), nil)

	// Assert
	if err == nil {
		t.Fatal("Do(context, nil) error = nil, want error")
	}
}

func requestWithoutURLIsRejected(t *testing.T) {
	// Arrange
	client := newClient(t, fixtures.ValidConfig(serviceURL, true), failOnCall(t))
	request := &http.Request{Method: http.MethodGet}

	// Act
	_, err := client.Do(context.Background(), request)

	// Assert
	if err == nil {
		t.Fatal("Do(context, request without URL) error = nil, want error")
	}
}

func nilRoundTripperIsRejected(t *testing.T) {
	// Arrange
	cfg := fixtures.ValidConfig(serviceURL, true)

	// Act
	_, err := httpclient.NewWithRoundTripper(cfg, nil)

	// Assert
	if err == nil {
		t.Fatal("NewWithRoundTripper(nil) error = nil, want error")
	}
}

func tlsMismatchIsRejected(t *testing.T) {
	// Arrange
	cfg := fixtures.TLSMismatchConfig()
	transport := mocks.RoundTripperFunc(func(*http.Request) (*http.Response, error) {
		return fixtures.Response(http.StatusOK), nil
	})

	// Act
	_, err := httpclient.NewWithRoundTripper(cfg, transport)

	// Assert
	if err == nil {
		t.Fatal("NewWithRoundTripper() error = nil, want TLS mismatch error")
	}
}

func closeReleasesIdleConnections(t *testing.T) {
	// Arrange
	transport := &mocks.CloseTrackingRoundTripper{}
	client := newClient(t, fixtures.ValidConfig(serviceURL, true), transport)

	// Act
	client.Close()

	// Assert
	if !transport.IsClosed() {
		t.Error("CloseIdleConnections was not called")
	}
}

func newClient(
	t *testing.T,
	cfg config.HTTPClientConfig,
	transport http.RoundTripper,
) *httpclient.Client {
	t.Helper()

	client, err := httpclient.NewWithRoundTripper(cfg, transport)
	if err != nil {
		t.Fatalf("NewWithRoundTripper() error = %v", err)
	}
	t.Cleanup(client.Close)
	return client
}

func closeResponse(t *testing.T, response *http.Response, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatalf("close response body: %v", err)
	}
}

func failOnCall(t *testing.T) http.RoundTripper {
	t.Helper()

	return mocks.RoundTripperFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("RoundTrip must not be called")
		return nil, nil
	})
}
