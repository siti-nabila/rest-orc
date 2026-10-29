package test_scenarios

import (
	"context"
	"errors"
	"fmt"

	"github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"google.golang.org/grpc/metadata"
	"reflect"
	"testing"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	authgateway "github.com/siti-nabila/rest-orc/internal/features/auth/gateway/authservice"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/test/features/auth/gateway/authservice/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func All() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "auth gateway preserves raw mapping, context, operation and errors", Run: authMapping},
		{Name: "auth gateway rejects nil client and context without dispatch", Run: authGuards},
	}
}

func authMapping(t *testing.T) {
	dependencyError := fmt.Errorf("dependency: %w", dictionary.ErrGRPCAuthServiceRequired)
	for _, mode := range []string{"register", "login"} {
		for _, tc := range []struct {
			name     string
			response *userv1.UserTokenResponse
			err      error
		}{
			{"success", fixtures.LoginTokenResponse(), nil},
			{"nil response", nil, nil},
			{"dependency error", nil, dependencyError},
			{"response with error", &userv1.UserTokenResponse{Token: "ignored"}, dependencyError},
		} {
			t.Run(mode+"/"+tc.name, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer unrelated", "x-request-id", "request"))
				input := domain.AuthRequest{Email: " User.Name@Example.COM ", Password: " legacy password "}
				call := func(actual context.Context, request *userv1.AuthRequest) (*userv1.UserTokenResponse, error) {
					md, _ := metadata.FromOutgoingContext(actual)
					if len(md.Get("authorization")) != 0 || md.Get("x-request-id")[0] != "request" || actual.Err() != ctx.Err() || request.GetEmail() != input.Email || request.GetPassword() != input.Password {
						t.Error("context or raw credential mapping changed")
					}
					return tc.response, tc.err
				}
				client := &mocks.Client{}
				if mode == "register" {
					client.RegisterFunc = call
				} else {
					client.LoginFunc = call
				}
				gateway, err := authgateway.New(client)
				if err != nil {
					t.Fatal(err)
				}
				var got any
				var wantErr error
				if mode == "register" {
					if tc.name == "success" {
						tc.response = fixtures.RegistrationResponse()
					}
					got, err = gateway.Register(ctx, input)
					wantErr = dictionary.ErrGRPCAuthRegisterResponseRequired
				} else {
					got, err = gateway.Login(ctx, input)
					wantErr = dictionary.ErrGRPCAuthLoginResponseRequired
				}
				switch {
				case tc.err != nil:
					if err != tc.err || !errors.Is(err, dictionary.ErrGRPCAuthServiceRequired) || !reflect.ValueOf(got).IsZero() {
						t.Error("dependency error identity/cause or empty result changed")
					}
				case tc.response == nil:
					if !errors.Is(err, wantErr) || !reflect.ValueOf(got).IsZero() {
						t.Error("nil response guard changed")
					}
				default:
					var want any = fixtures.LoginResult()
					if mode == "register" {
						want = domain.RegisterResponse{RegistrationID: 17, RegistrationStatus: "PENDING"}
					}
					if err != nil || got != want {
						t.Error("session/registration mapping changed")
					}
				}
				wantRegister, wantLogin := 0, 1
				if mode == "register" {
					wantRegister, wantLogin = 1, 0
				}
				if client.RegisterCalls != wantRegister || client.LoginCalls != wantLogin {
					t.Error("wrong RPC or unexpected retry")
				}
			})
		}
	}
}

func authGuards(t *testing.T) {
	if gateway, err := authgateway.New(nil); gateway != nil || !errors.Is(err, dictionary.ErrAuthGatewayClientRequired) || !errors.Is(err, dictionary.ErrUsersGatewayClientRequired) {
		t.Error("nil client guard identity changed")
	}
	client := &mocks.Client{}
	gateway, err := authgateway.New(client)
	if err != nil {
		t.Fatal(err)
	}
	for _, execute := range []func(context.Context, domain.AuthRequest) (any, error){func(c context.Context, r domain.AuthRequest) (any, error) { return gateway.Register(c, r) }, func(c context.Context, r domain.AuthRequest) (any, error) { return gateway.Login(c, r) }} {
		got, err := execute(nil, domain.AuthRequest{})
		if !reflect.ValueOf(got).IsZero() || !errors.Is(err, dictionary.ErrAuthGatewayContextRequired) || !errors.Is(err, dictionary.ErrUsersGatewayContextRequired) {
			t.Error("nil context guard identity changed")
		}
	}
	if client.RegisterCalls != 0 || client.LoginCalls != 0 {
		t.Error("nil context reached downstream")
	}
}
