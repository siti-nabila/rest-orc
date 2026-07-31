package httpclient

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/siti-nabila/rest-orc/internal/client/tlsconfig"
	"github.com/siti-nabila/rest-orc/internal/config"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

type Client struct {
	baseURL        *url.URL
	httpClient     *http.Client
	closeTransport func()
	requestTimeout time.Duration
}

type cancelReadCloser struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (body *cancelReadCloser) Close() error {
	err := body.ReadCloser.Close()
	body.cancel()
	return err
}

func New(cfg config.HTTPClientConfig) (*Client, error) {
	transport, err := NewTransport(cfg)
	if err != nil {
		return nil, err
	}
	return NewWithRoundTripper(cfg, transport)
}

func NewTransport(cfg config.HTTPClientConfig) (*http.Transport, error) {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   cfg.RequestTimeout.Duration,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		DisableKeepAlives:     !cfg.Keepalive.Enabled,
		MaxIdleConns:          cfg.Keepalive.MaxIdleConnections,
		MaxIdleConnsPerHost:   cfg.Keepalive.MaxIdleConnectionsPerHost,
		MaxConnsPerHost:       cfg.Keepalive.MaxConnectionsPerHost,
		IdleConnTimeout:       cfg.Keepalive.IdleConnectionTimeout.Duration,
		ResponseHeaderTimeout: cfg.Keepalive.ResponseHeaderTimeout.Duration,
		TLSHandshakeTimeout:   cfg.RequestTimeout.Duration,
		ExpectContinueTimeout: time.Second,
	}

	if !cfg.TLS.Enabled {
		return transport, nil
	}

	tlsConfig, err := tlsconfig.New(cfg.TLS)
	if err != nil {
		return nil, dictionary.ConfigureHTTPClientTLS(err)
	}
	transport.TLSClientConfig = tlsConfig
	return transport, nil
}

func NewWithRoundTripper(
	cfg config.HTTPClientConfig,
	roundTripper http.RoundTripper,
) (*Client, error) {
	if roundTripper == nil {
		return nil, dictionary.ErrHTTPRoundTripperRequired
	}

	baseURL, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, dictionary.ParseHTTPClientBaseURL(cfg.BaseURL, err)
	}
	if !baseURL.IsAbs() || baseURL.Host == "" {
		return nil, dictionary.AbsoluteHTTPClientBaseURLRequired(cfg.BaseURL)
	}
	if baseURL.Scheme != "http" && baseURL.Scheme != "https" {
		return nil, dictionary.UnsupportedHTTPClientBaseURLScheme(cfg.BaseURL)
	}
	if cfg.TLS.Enabled != (baseURL.Scheme == "https") {
		return nil, dictionary.HTTPClientTLSSchemeMismatch(baseURL.Scheme)
	}

	closeTransport := func() {}
	if closer, ok := roundTripper.(interface{ CloseIdleConnections() }); ok {
		closeTransport = closer.CloseIdleConnections
	}

	return &Client{
		baseURL:        baseURL,
		httpClient:     &http.Client{Transport: roundTripper},
		closeTransport: closeTransport,
		requestTimeout: cfg.RequestTimeout.Duration,
	}, nil
}

func (c *Client) Do(ctx context.Context, request *http.Request) (*http.Response, error) {
	if ctx == nil {
		return nil, dictionary.ErrHTTPRequestContextRequired
	}
	if request == nil {
		return nil, dictionary.ErrHTTPRequestRequired
	}
	if request.URL == nil {
		return nil, dictionary.ErrHTTPRequestURLRequired
	}

	requestContext, cancel := context.WithTimeout(ctx, c.requestTimeout)

	outbound := request.Clone(requestContext)
	if !outbound.URL.IsAbs() {
		outbound.URL = c.baseURL.ResolveReference(outbound.URL)
	}

	response, err := c.httpClient.Do(outbound)
	if err != nil {
		cancel()
		return nil, dictionary.ExecuteHTTPRequest(err)
	}
	if response.Body == nil {
		cancel()
		return nil, dictionary.ErrHTTPResponseBodyRequired
	}
	response.Body = &cancelReadCloser{
		ReadCloser: response.Body,
		cancel:     cancel,
	}
	return response, nil
}

func (c *Client) Close() {
	c.closeTransport()
}
