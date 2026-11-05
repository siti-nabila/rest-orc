package test_scenarios

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	authv1 "github.com/siti-nabila/api-contracts/pb/auth/v1"
	authservice "github.com/siti-nabila/rest-orc/internal/client/authservice"
	"github.com/siti-nabila/rest-orc/internal/client/grpcclient"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/test/client/authservice/mocks"
	grpcfixtures "github.com/siti-nabila/rest-orc/test/client/grpcclient/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc"
)

func PasswordScenarios() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "password client preserves raw request metadata bounded context and cleanup", Run: passwordClientContext},
		{Name: "password client guards dependencies context request and response", Run: passwordClientGuards},
		{Name: "existing client constructors provide the generated AuthService SDK", Run: passwordClientGeneratedSDK},
	}
}

func passwordClientContext(t *testing.T) {
	for _, mode := range []string{"configured deadline", "earlier deadline", "cancelled", "dependency error", "nil response"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(requestContext())
				defer cancel()
				wantDeadline := time.Now().Add(grpcfixtures.ValidConfig().RequestTimeout.Duration)
				if mode == "earlier deadline" {
					var stop context.CancelFunc
					ctx, stop = context.WithTimeout(ctx, time.Millisecond*500)
					defer stop()
					wantDeadline, _ = ctx.Deadline()
				}
				if mode == "cancelled" {
					cancel()
				}
				request := &authv1.ChangePasswordRequest{CurrentPassword: " PaSs é ", NewPassword: "new-password"}
				response := &authv1.ChangePasswordResponse{Changed: true, LoginRequired: true}
				failure := fmt.Errorf("password dependency: %w", errDependency)
				var child context.Context
				service := &mocks.PasswordService{ChangeFunc: func(actual context.Context, payload *authv1.ChangePasswordRequest, _ ...grpc.CallOption) (*authv1.ChangePasswordResponse, error) {
					child = actual
					assertRequestContext(t, actual)
					deadline, _ := actual.Deadline()
					if !deadline.Equal(wantDeadline) || payload != request {
						t.Error("request or deadline changed")
					}
					switch mode {
					case "cancelled":
						return nil, actual.Err()
					case "dependency error":
						return nil, failure
					case "nil response":
						return nil, nil
					default:
						return response, nil
					}
				}}
				client, err := authservice.NewWithPasswordService(newTransport(t), failByDefaultService(t), &mocks.ProfileService{}, service)
				if err != nil {
					t.Fatal(err)
				}
				got, err := client.ChangePassword(ctx, request)
				switch mode {
				case "cancelled":
					if !errors.Is(err, context.Canceled) {
						t.Fatal("cancellation lost")
					}
				case "dependency error":
					if err != failure || !errors.Is(err, errDependency) {
						t.Fatal("dependency identity/cause lost")
					}
				case "nil response":
					if !errors.Is(err, dictionary.ErrGRPCChangePasswordResponseRequired) {
						t.Fatal("nil response guard differs")
					}
				default:
					if err != nil || got != response {
						t.Fatal("successful response changed")
					}
				}
				if service.Calls != 1 || child == nil || child.Err() != context.Canceled {
					t.Fatal("request cleanup or call count differs")
				}
			})
		})
	}
}

func passwordClientGuards(t *testing.T) {
	transport := newTransport(t)
	user, profile, password := failByDefaultService(t), &mocks.ProfileService{}, &mocks.PasswordService{}
	for _, tc := range []struct {
		name      string
		transport *grpcclient.Client
		user      authservice.Service
		profile   authservice.ProfileService
		password  authservice.PasswordService
		want      error
	}{
		{"transport", nil, user, profile, password, dictionary.ErrGRPCAuthTransportRequired},
		{"user", transport, nil, profile, password, dictionary.ErrGRPCAuthServiceRequired},
		{"profile", transport, user, nil, password, dictionary.ErrGRPCProfileServiceRequired},
		{"password", transport, user, profile, nil, dictionary.ErrGRPCPasswordServiceRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := authservice.NewWithPasswordService(tc.transport, tc.user, tc.profile, tc.password)
			if got != nil || !errors.Is(err, tc.want) {
				t.Fatal("constructor guard differs")
			}
		})
	}
	client, err := authservice.NewWithPasswordService(transport, user, profile, password)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.ChangePassword(nil, &authv1.ChangePasswordRequest{}); !errors.Is(err, dictionary.ErrGRPCAuthRequestContextRequired) {
		t.Fatal("nil context accepted")
	}
	if _, err := client.ChangePassword(context.Background(), nil); !errors.Is(err, dictionary.ErrGRPCChangePasswordRequestRequired) {
		t.Fatal("nil request accepted")
	}
	if password.Calls != 0 {
		t.Fatal("guard dispatched password RPC")
	}
}

func passwordClientGeneratedSDK(t *testing.T) {
	for _, constructor := range []struct {
		name string
		new  func(*grpcclient.Client) (*authservice.Client, error)
	}{
		{"New", authservice.New},
		{"NewWithService", func(transport *grpcclient.Client) (*authservice.Client, error) {
			return authservice.NewWithService(transport, failByDefaultService(t))
		}},
		{"NewWithServices", func(transport *grpcclient.Client) (*authservice.Client, error) {
			return authservice.NewWithServices(transport, failByDefaultService(t), &mocks.ProfileService{})
		}},
	} {
		t.Run(constructor.name, func(t *testing.T) {
			calls := 0
			request := &authv1.ChangePasswordRequest{CurrentPassword: "legacy password", NewPassword: "new-password"}
			transport, err := grpcclient.New(grpcfixtures.ValidConfig(), grpc.WithUnaryInterceptor(func(_ context.Context, method string, payload, reply any, _ *grpc.ClientConn, _ grpc.UnaryInvoker, _ ...grpc.CallOption) error {
				calls++
				if method != authv1.AuthService_ChangePassword_FullMethodName || payload != request {
					t.Error("generated SDK dispatch differs")
				}
				result := reply.(*authv1.ChangePasswordResponse)
				result.Changed, result.LoginRequired = true, true
				return nil
			}))
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := transport.Close(); err != nil {
					t.Error(err)
				}
			}()
			client, err := constructor.new(transport)
			if err != nil {
				t.Fatal(err)
			}
			got, err := client.ChangePassword(context.Background(), request)
			if err != nil || !got.GetChanged() || !got.GetLoginRequired() || calls != 1 {
				t.Fatal("constructor did not configure the password SDK")
			}
		})
	}
}
