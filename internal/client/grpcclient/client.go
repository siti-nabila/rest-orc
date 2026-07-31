package grpcclient

import (
	"context"
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

func New(cfg config.GRPCClientConfig) (*Client, error) {
	transportCredentials, err := newTransportCredentials(cfg.TLS)
	if err != nil {
		return nil, err
	}

	options := []grpc.DialOption{
		grpc.WithTransportCredentials(transportCredentials),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(cfg.MaxReceiveMessageBytes),
			grpc.MaxCallSendMsgSize(cfg.MaxSendMessageBytes),
		),
	}
	if cfg.Keepalive.Enabled {
		options = append(options, grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                cfg.Keepalive.Time.Duration,
			Timeout:             cfg.Keepalive.Timeout.Duration,
			PermitWithoutStream: false,
		}))
	}

	connection, err := grpc.NewClient(cfg.Target, options...)
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

func newTransportCredentials(cfg config.TLSConfig) (credentials.TransportCredentials, error) {
	if !cfg.Enabled {
		return insecure.NewCredentials(), nil
	}

	tlsConfig, err := tlsconfig.New(cfg)
	if err != nil {
		return nil, dictionary.ConfigureGRPCClientTLS(err)
	}
	return credentials.NewTLS(tlsConfig), nil
}
