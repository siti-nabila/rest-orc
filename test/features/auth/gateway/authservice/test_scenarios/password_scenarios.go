package test_scenarios

import (
	"context"
	"errors"
	"fmt"
	"testing"

	authv1 "github.com/siti-nabila/api-contracts/pb/auth/v1"
	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	authgateway "github.com/siti-nabila/rest-orc/internal/features/auth/gateway/authservice"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/test/features/auth/gateway/authservice/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func PasswordScenarios() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "password gateway maps raw credentials and every authoritative flag combination", Run: passwordGatewayMapping},
		{Name: "password gateway guards dependencies and preserves wrapped errors", Run: passwordGatewayGuards},
	}
}

func passwordGatewayMapping(t *testing.T) {
	for _, changed := range []bool{false, true} {
		for _, loginRequired := range []bool{false, true} {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			request := domain.ChangePasswordRequest{CurrentPassword: " PaSs é\t ", NewPassword: "new-password"}
			client := &mocks.PasswordClient{ChangeFunc: func(actual context.Context, payload *authv1.ChangePasswordRequest) (*authv1.ChangePasswordResponse, error) {
				if actual != ctx || payload.CurrentPassword != request.CurrentPassword || payload.NewPassword != request.NewPassword {
					t.Error("gateway changed a credential or context")
				}
				return &authv1.ChangePasswordResponse{Changed: changed, LoginRequired: loginRequired}, nil
			}}
			gateway, err := authgateway.NewPassword(client)
			if err != nil {
				t.Fatal(err)
			}
			got, err := gateway.ChangePassword(ctx, request)
			if err != nil || got != (domain.ChangePasswordResponse{Changed: changed, LoginRequired: loginRequired}) || client.Calls != 1 {
				t.Fatal("mapped flags or call count differ")
			}
		}
	}
}

func passwordGatewayGuards(t *testing.T) {
	if got, err := authgateway.NewPassword(nil); got != nil || !errors.Is(err, dictionary.ErrAuthGatewayClientRequired) {
		t.Fatal("nil dependency guard differs")
	}
	cause := errors.New("password dependency sentinel")
	failure := fmt.Errorf("password dependency: %w", cause)
	for _, dependencyError := range []error{nil, failure} {
		client := &mocks.PasswordClient{ChangeFunc: func(context.Context, *authv1.ChangePasswordRequest) (*authv1.ChangePasswordResponse, error) {
			return nil, dependencyError
		}}
		gateway, err := authgateway.NewPassword(client)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := gateway.ChangePassword(nil, domain.ChangePasswordRequest{}); !errors.Is(err, dictionary.ErrAuthGatewayContextRequired) || client.Calls != 0 {
			t.Fatal("nil context dispatched")
		}
		got, err := gateway.ChangePassword(context.Background(), domain.ChangePasswordRequest{})
		if dependencyError == nil {
			if !errors.Is(err, dictionary.ErrGRPCChangePasswordResponseRequired) {
				t.Fatal("nil result accepted")
			}
		} else if err != failure || !errors.Is(err, cause) {
			t.Fatal("error identity/cause lost")
		}
		if got != (domain.ChangePasswordResponse{}) || client.Calls != 1 {
			t.Fatal("unexpected success or retry")
		}
	}
}
