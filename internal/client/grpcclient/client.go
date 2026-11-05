package grpcclient

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/siti-nabila/rest-orc/internal/client/tlsconfig"
	"github.com/siti-nabila/rest-orc/internal/config"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

type Client struct {
	connection     *grpc.ClientConn
	requestTimeout time.Duration
}

func New(cfg config.GRPCClientConfig, options ...grpc.DialOption) (*Client, error) {
	if cfg.ServiceToken != "" {
		if err := cfg.ValidateServiceCredentials("grpc_client"); err != nil {
			return nil, err
		}
	}
	transportCredentials, err := newTransportCredentials(cfg)
	if err != nil {
		return nil, err
	}

	// Apply configuration-owned security after optional transport options.
	dialOptions := append([]grpc.DialOption{}, options...)
	dialOptions = append(dialOptions,
		grpc.WithTransportCredentials(transportCredentials),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(cfg.MaxReceiveMessageBytes),
			grpc.MaxCallSendMsgSize(cfg.MaxSendMessageBytes),
		),
	)
	if cfg.ServiceToken != "" {
		dialOptions = append(dialOptions, grpc.WithPerRPCCredentials(serviceCredentials{token: cfg.ServiceToken}))
	}
	if cfg.Keepalive.Enabled {
		dialOptions = append(dialOptions, grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                cfg.Keepalive.Time.Duration,
			Timeout:             cfg.Keepalive.Timeout.Duration,
			PermitWithoutStream: false,
		}))
	}

	connection, err := grpc.NewClient(cfg.Target, dialOptions...)
	if err != nil {
		return nil, dictionary.CreateGRPCClient(cfg.Target, err)
	}

	return &Client{
		connection:     connection,
		requestTimeout: cfg.RequestTimeout.Duration,
	}, nil
}

func (c *Client) Connection() grpc.ClientConnInterface {
	return c.connection
}

func (c *Client) RequestContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, c.requestTimeout)
}

func (c *Client) Close() error {
	if err := c.connection.Close(); err != nil {
		return dictionary.CloseGRPCClient(err)
	}
	return nil
}

func newTransportCredentials(cfg config.GRPCClientConfig) (credentials.TransportCredentials, error) {
	if !cfg.TLS.Enabled {
		return insecure.NewCredentials(), nil
	}

	tlsConfig, err := tlsconfig.New(cfg.TLS)
	if err != nil {
		return nil, dictionary.ConfigureGRPCClientTLS(err)
	}
	if cfg.ServiceToken != "" {
		tlsConfig.MinVersion = tls.VersionTLS13
	}
	return credentials.NewTLS(tlsConfig), nil
}
