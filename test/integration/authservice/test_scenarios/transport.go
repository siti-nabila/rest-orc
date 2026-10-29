package test_scenarios

import (
	"context"
	"crypto/tls"
	"net"
	"sync"
	"testing"
	"time"

	authv1 "github.com/siti-nabila/api-contracts/pb/auth/v1"
	profilev1 "github.com/siti-nabila/api-contracts/pb/profile/v1"
	sessionv1 "github.com/siti-nabila/api-contracts/pb/session/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/rest-orc/internal/client/authservice"
	"github.com/siti-nabila/rest-orc/internal/client/grpcclient"
	"github.com/siti-nabila/rest-orc/internal/config"
	"github.com/siti-nabila/rest-orc/test/integration/authservice/fixtures"
	"github.com/siti-nabila/rest-orc/test/integration/authservice/mocks"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/test/bufconn"
)

type call struct {
	method   string
	metadata metadata.MD
	request  any
	tls      uint16
	deadline time.Time
}

type transport struct {
	client     *authservice.Client
	connection *grpcclient.Client
	server     *grpc.Server
	listener   *bufconn.Listener
	serveDone  chan struct{}
	mu         sync.Mutex
	calls      []call
	dials      int
}

func secureConfig(t *testing.T) config.GRPCClientConfig {
	t.Helper()
	return config.GRPCClientConfig{
		Target: "passthrough:///auth.test", ServiceToken: fixtures.Token(17),
		RequestTimeout:         config.Duration{Duration: 5 * time.Second},
		MaxReceiveMessageBytes: 4 * 1024 * 1024, MaxSendMessageBytes: 4 * 1024 * 1024,
		TLS: config.TLSConfig{Enabled: true, CAFile: fixtures.CAFile(t, "ca")},
	}
}

func newTransport(t *testing.T, cfg config.GRPCClientConfig, serverTLS *tls.Config, intercept grpc.UnaryServerInterceptor) *transport {
	t.Helper()
	h := &transport{listener: bufconn.Listen(1024 * 1024), serveDone: make(chan struct{})}
	options := []grpc.ServerOption{grpc.UnaryInterceptor(func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, _ := metadata.FromIncomingContext(ctx)
		record := call{method: info.FullMethod, metadata: md.Copy(), request: req}
		record.deadline, _ = ctx.Deadline()
		if p, ok := peer.FromContext(ctx); ok {
			if state, ok := p.AuthInfo.(credentials.TLSInfo); ok {
				record.tls = state.State.Version
			}
		}
		h.mu.Lock()
		h.calls = append(h.calls, record)
		h.mu.Unlock()
		if intercept != nil {
			return intercept(ctx, req, info, handler)
		}
		return handler(ctx, req)
	})}
	if serverTLS != nil {
		options = append(options, grpc.Creds(credentials.NewTLS(serverTLS)))
	}
	h.server = grpc.NewServer(options...)
	userv1.RegisterUserServiceServer(h.server, &mocks.Server{})
	profilev1.RegisterProfileServiceServer(h.server, &mocks.ProfileServer{})
	authv1.RegisterAuthServiceServer(h.server, &mocks.PasswordServer{})
	sessionv1.RegisterSessionServiceServer(h.server, &mocks.SelfSessionServer{})
	go func() {
		defer close(h.serveDone)
		_ = h.server.Serve(h.listener)
	}()
	connection, err := grpcclient.New(cfg, grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
		h.mu.Lock()
		h.dials++
		h.mu.Unlock()
		return h.listener.DialContext(ctx)
	}))
	if err != nil {
		h.server.Stop()
		_ = h.listener.Close()
		<-h.serveDone
		t.Fatal("could not construct configured test transport")
	}
	h.connection = connection
	h.client, err = authservice.New(connection)
	if err != nil {
		h.close(t)
		t.Fatal(err)
	}
	return h
}

func defaultServerTLS(t *testing.T) *tls.Config {
	t.Helper()
	return &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{fixtures.ServerCertificate(t, "server")}}
}

func (h *transport) snapshot() ([]call, int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]call(nil), h.calls...), h.dials
}

func (h *transport) close(t *testing.T) {
	t.Helper()
	if err := h.connection.Close(); err != nil {
		t.Error("could not close test transport")
	}
	h.server.Stop()
	_ = h.listener.Close()
	<-h.serveDone
}
