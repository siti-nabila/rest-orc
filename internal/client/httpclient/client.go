package httpclient

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/siti-nabila/rest-orc/internal/client/tlsconfig"
	"github.com/siti-nabila/rest-orc/internal/config"
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
		return nil, fmt.Errorf("configure HTTP client TLS: %w", err)
	}
	transport.TLSClientConfig = tlsConfig
	return transport, nil
}

func NewWithRoundTripper(
	cfg config.HTTPClientConfig,
	roundTripper http.RoundTripper,
) (*Client, error) {
	if roundTripper == nil {
		return nil, fmt.Errorf("create HTTP client: round tripper must not be nil")
	}

	baseURL, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse HTTP client base URL %q: %w", cfg.BaseURL, err)
	}
	if !baseURL.IsAbs() || baseURL.Host == "" {
		return nil, fmt.Errorf("parse HTTP client base URL %q: URL must be absolute", cfg.BaseURL)
	}
	if baseURL.Scheme != "http" && baseURL.Scheme != "https" {
		return nil, fmt.Errorf(
			"parse HTTP client base URL %q: scheme must be http or https",
			cfg.BaseURL,
		)
	}
	if cfg.TLS.Enabled != (baseURL.Scheme == "https") {
		return nil, fmt.Errorf(
			"create HTTP client: TLS enabled setting does not match base URL scheme %q",
			baseURL.Scheme,
		)
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
		return nil, fmt.Errorf("execute HTTP request: context must not be nil")
	}
	if request == nil {
		return nil, fmt.Errorf("execute HTTP request: request must not be nil")
	}
	if request.URL == nil {
		return nil, fmt.Errorf("execute HTTP request: request URL must not be nil")
	}

	requestContext, cancel := context.WithTimeout(ctx, c.requestTimeout)

	outbound := request.Clone(requestContext)
	if !outbound.URL.IsAbs() {
		outbound.URL = c.baseURL.ResolveReference(outbound.URL)
	}

	response, err := c.httpClient.Do(outbound)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("execute HTTP request: %w", err)
	}
	if response.Body == nil {
		cancel()
		return nil, fmt.Errorf("execute HTTP request: response body must not be nil")
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
