package test_scenarios

import (
	"context"
	"errors"
	"testing"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	"github.com/siti-nabila/rest-orc/internal/client/grpcauth"
	"github.com/siti-nabila/rest-orc/internal/client/grpcclient"
	grpcauthfixtures "github.com/siti-nabila/rest-orc/tests/client/grpcauth/fixtures"
	"github.com/siti-nabila/rest-orc/tests/client/grpcauth/mocks"
	grpcclientfixtures "github.com/siti-nabila/rest-orc/tests/client/grpcclient/fixtures"
	"github.com/siti-nabila/rest-orc/tests/shared/testutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

type anyEmpty = *emptypb.Empty
type callOption = grpc.CallOption
type contextKey string

const requestIDKey contextKey = "request-id"

var errDependency = errors.New("auth dependency failed")

func All() []testutils.Scenario {
	return []testutils.Scenario{
		{
			Name: "Me propagates context metadata and request timeout",
			Run:  mePropagatesContext,
		},
		{
			Name: "Me preserves dependency error identity",
			Run:  mePreservesDependencyError,
		},
		{
			Name: "Me rejects nil dependency response",
			Run:  meRejectsNilResponse,
		},
		{
			Name: "ListUsers forwards request and context metadata",
			Run:  listUsersForwardsRequestAndContext,
		},
		{
			Name: "ListUsers preserves dependency error identity",
			Run:  listUsersPreservesDependencyError,
		},
		{
			Name: "ListUsers rejects nil dependency response",
			Run:  listUsersRejectsNilResponse,
		},
		{
			Name: "nil request context is rejected before dependency call",
			Run:  nilContextIsRejected,
		},
		{
			Name: "nil ListUsers request is rejected before dependency call",
			Run:  nilListUsersRequestIsRejected,
		},
		{
			Name: "nil constructor dependencies are rejected",
			Run:  nilConstructorDependenciesAreRejected,
		},
	}
}

func mePropagatesContext(t *testing.T) {
	// Arrange
	expected := grpcauthfixtures.MeResponse()
	service := failByDefaultService(t)
	service.MeFunc = func(
		ctx context.Context,
		_ anyEmpty,
		_ ...callOption,
	) (*userv1.UserData, error) {
		assertRequestContext(t, ctx)
		return expected, nil
	}
	client := newClient(t, service)
	ctx := requestContext()

	// Act
	actual, err := client.Me(ctx)

	// Assert
	if err != nil {
		t.Fatalf("Me() error = %v", err)
	}
	if actual != expected {
		t.Error("Me() did not return dependency response")
	}
}

func mePreservesDependencyError(t *testing.T) {
	// Arrange
	service := failByDefaultService(t)
	service.MeFunc = func(
		context.Context,
		anyEmpty,
		...callOption,
	) (*userv1.UserData, error) {
		return nil, errDependency
	}
	client := newClient(t, service)

	// Act
	_, err := client.Me(context.Background())

	// Assert
	if !errors.Is(err, errDependency) {
		t.Errorf("Me() error = %v, want dependency error", err)
	}
}

func meRejectsNilResponse(t *testing.T) {
	// Arrange
	service := failByDefaultService(t)
	service.MeFunc = func(
		context.Context,
		anyEmpty,
		...callOption,
	) (*userv1.UserData, error) {
		return nil, nil
	}
	client := newClient(t, service)

	// Act
	_, err := client.Me(context.Background())

	// Assert
	if err == nil {
		t.Fatal("Me() error = nil, want nil response error")
	}
}

func listUsersForwardsRequestAndContext(t *testing.T) {
	// Arrange
	expectedRequest := grpcauthfixtures.ListUsersRequest()
	expectedResponse := grpcauthfixtures.ListUsersResponse()
	service := failByDefaultService(t)
	service.ListUsersFunc = func(
		ctx context.Context,
		request *userv1.ListUsersRequest,
		_ ...callOption,
	) (*userv1.ListUsersResponse, error) {
		assertRequestContext(t, ctx)
		if request != expectedRequest {
			t.Error("ListUsers() did not receive original request")
		}
		return expectedResponse, nil
	}
	client := newClient(t, service)

	// Act
	actual, err := client.ListUsers(requestContext(), expectedRequest)

	// Assert
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if actual != expectedResponse {
		t.Error("ListUsers() did not return dependency response")
	}
}

func listUsersPreservesDependencyError(t *testing.T) {
	// Arrange
	service := failByDefaultService(t)
	service.ListUsersFunc = func(
		context.Context,
		*userv1.ListUsersRequest,
		...callOption,
	) (*userv1.ListUsersResponse, error) {
		return nil, errDependency
	}
	client := newClient(t, service)

	// Act
	_, err := client.ListUsers(
		context.Background(),
		grpcauthfixtures.ListUsersRequest(),
	)

	// Assert
	if !errors.Is(err, errDependency) {
		t.Errorf("ListUsers() error = %v, want dependency error", err)
	}
}

func listUsersRejectsNilResponse(t *testing.T) {
	// Arrange
	service := failByDefaultService(t)
	service.ListUsersFunc = func(
		context.Context,
		*userv1.ListUsersRequest,
		...callOption,
	) (*userv1.ListUsersResponse, error) {
		return nil, nil
	}
	client := newClient(t, service)

	// Act
	_, err := client.ListUsers(
		context.Background(),
		grpcauthfixtures.ListUsersRequest(),
	)

	// Assert
	if err == nil {
		t.Fatal("ListUsers() error = nil, want nil response error")
	}
}

func nilContextIsRejected(t *testing.T) {
	// Arrange
	client := newClient(t, failByDefaultService(t))

	// Act
	_, err := client.Me(nil)

	// Assert
	if err == nil {
		t.Fatal("Me(nil) error = nil, want error")
	}
}

func nilListUsersRequestIsRejected(t *testing.T) {
	// Arrange
	client := newClient(t, failByDefaultService(t))

	// Act
	_, err := client.ListUsers(context.Background(), nil)

	// Assert
	if err == nil {
		t.Fatal("ListUsers(context, nil) error = nil, want error")
	}
}

func nilConstructorDependenciesAreRejected(t *testing.T) {
	// Arrange
	transport := newTransport(t)

	// Act
	_, transportErr := grpcauth.New(nil)
	_, serviceErr := grpcauth.NewWithService(transport, nil)

	// Assert
	if transportErr == nil {
		t.Error("New(nil) error = nil, want error")
	}
	if serviceErr == nil {
		t.Error("NewWithService(transport, nil) error = nil, want error")
	}
}

func newClient(t *testing.T, service grpcauth.Service) *grpcauth.Client {
	t.Helper()

	client, err := grpcauth.NewWithService(newTransport(t), service)
	if err != nil {
		t.Fatalf("NewWithService() error = %v", err)
	}
	return client
}

func newTransport(t *testing.T) *grpcclient.Client {
	t.Helper()

	transport, err := grpcclient.New(grpcclientfixtures.ValidConfig())
	if err != nil {
		t.Fatalf("grpcclient.New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := transport.Close(); err != nil {
			t.Errorf("close gRPC transport: %v", err)
		}
	})
	return transport
}

func failByDefaultService(t *testing.T) *mocks.Service {
	t.Helper()

	return &mocks.Service{
		MeFunc: func(context.Context, anyEmpty, ...callOption) (*userv1.UserData, error) {
			t.Fatal("unexpected Me dependency call")
			return nil, nil
		},
		ListUsersFunc: func(
			context.Context,
			*userv1.ListUsersRequest,
			...callOption,
		) (*userv1.ListUsersResponse, error) {
			t.Fatal("unexpected ListUsers dependency call")
			return nil, nil
		},
	}
}

func requestContext() context.Context {
	ctx := context.WithValue(context.Background(), requestIDKey, "request-17")
	return metadata.AppendToOutgoingContext(
		ctx,
		"authorization",
		"Bearer test-token",
		locale.MetadataKey,
		"zh-CN",
	)
}

func assertRequestContext(t *testing.T, ctx context.Context) {
	t.Helper()

	if _, exists := ctx.Deadline(); !exists {
		t.Error("dependency context has no request deadline")
	}
	if value := ctx.Value(requestIDKey); value != "request-17" {
		t.Errorf("request ID = %v, want request-17", value)
	}
	metadataValue, exists := metadata.FromOutgoingContext(ctx)
	if !exists {
		t.Fatal("outgoing gRPC metadata is missing")
	}
	values := metadataValue.Get("authorization")
	if len(values) != 1 || values[0] != "Bearer test-token" {
		t.Errorf("authorization metadata = %v, want test token", values)
	}
	languages := metadataValue.Get(locale.MetadataKey)
	if len(languages) != 1 || languages[0] != "zh-CN" {
		t.Errorf("language metadata = %v, want zh-CN", languages)
	}
}
