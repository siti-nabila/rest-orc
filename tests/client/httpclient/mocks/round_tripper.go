package mocks

import (
	"net/http"
	"sync/atomic"

	"github.com/siti-nabila/rest-orc/tests/client/httpclient/fixtures"
)

type RoundTripperFunc func(*http.Request) (*http.Response, error)

func (fn RoundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

type CloseTrackingRoundTripper struct {
	closed atomic.Bool
}

func (*CloseTrackingRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return fixtures.Response(http.StatusNoContent), nil
}

func (transport *CloseTrackingRoundTripper) CloseIdleConnections() {
	transport.closed.Store(true)
}

func (transport *CloseTrackingRoundTripper) IsClosed() bool {
	return transport.closed.Load()
}
