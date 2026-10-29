package test_scenarios

import (
	"context"
	"errors"
	profilev1 "github.com/siti-nabila/api-contracts/pb/profile/v1"
	authservice "github.com/siti-nabila/rest-orc/internal/client/authservice"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/test/client/authservice/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc"
	"testing"
	"testing/synctest"
	"time"
)

func ProfileScenarios() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "profile client preserves payload metadata deadline cancellation and error", Run: profileClientContext},
		{Name: "profile client guards dependencies context request and response", Run: profileClientGuards},
	}
}
func profileClientContext(t *testing.T) {
	for _, mode := range []string{"success", "error", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(requestContext(), time.Millisecond*500)
				defer cancel()
				deadline, _ := ctx.Deadline()
				if mode == "cancelled" {
					cancel()
				}
				expected := &profilev1.ProfileResponse{Version: 17}
				request := &profilev1.ProfileRequest{Name: "NASA"}
				var child context.Context
				service := &mocks.ProfileService{UpdateFunc: func(actual context.Context, got *profilev1.ProfileRequest, _ ...grpc.CallOption) (*profilev1.ProfileResponse, error) {
					child = actual
					assertRequestContext(t, actual)
					actualDeadline, _ := actual.Deadline()
					if !actualDeadline.Equal(deadline) {
						t.Fatalf("deadline=%v want %v", actualDeadline, deadline)
					}
					if got != request {
						t.Fatal("request replaced")
					}
					switch mode {
					case "error":
						return nil, errDependency
					case "cancelled":
						if !errors.Is(actual.Err(), context.Canceled) {
							t.Fatal("cancellation lost")
						}
						return nil, actual.Err()
					}
					return expected, nil
				}}
				client, err := authservice.NewWithServices(newTransport(t), failByDefaultService(t), service)
				if err != nil {
					t.Fatal(err)
				}
				got, err := client.UpdateProfile(ctx, request)
				switch mode {
				case "success":
					if err != nil || got != expected {
						t.Fatalf("result=%v err=%v", got, err)
					}
				case "error":
					if !errors.Is(err, errDependency) {
						t.Fatalf("error=%v", err)
					}
				case "cancelled":
					if !errors.Is(err, context.Canceled) {
						t.Fatalf("error=%v", err)
					}
				}
				if service.Calls != 1 || child.Err() != context.Canceled {
					t.Fatal("call count or child cleanup")
				}
			})
		})
	}
}
func profileClientGuards(t *testing.T) {
	transport := newTransport(t)
	user := failByDefaultService(t)
	service := &mocks.ProfileService{UpdateFunc: func(context.Context, *profilev1.ProfileRequest, ...grpc.CallOption) (*profilev1.ProfileResponse, error) {
		return nil, nil
	}}
	for _, tc := range []struct {
		name             string
		want, errorValue error
	}{
		{name: "nil transport", want: dictionary.ErrGRPCAuthTransportRequired, errorValue: func() error { _, err := authservice.NewWithServices(nil, user, service); return err }()},
		{name: "nil user", want: dictionary.ErrGRPCAuthServiceRequired, errorValue: func() error { _, err := authservice.NewWithServices(transport, nil, service); return err }()},
		{name: "nil profile", want: dictionary.ErrGRPCProfileServiceRequired, errorValue: func() error { _, err := authservice.NewWithServices(transport, user, nil); return err }()},
		{name: "legacy nil transport", want: dictionary.ErrGRPCAuthTransportRequired, errorValue: func() error { _, err := authservice.NewWithService(nil, user); return err }()},
	} {
		if !errors.Is(tc.errorValue, tc.want) {
			t.Fatalf("%s: %v", tc.name, tc.errorValue)
		}
	}
	client, err := authservice.NewWithServices(transport, user, service)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.UpdateProfile(nil, &profilev1.ProfileRequest{}); !errors.Is(err, dictionary.ErrGRPCAuthRequestContextRequired) {
		t.Fatal(err)
	}
	if _, err := client.UpdateProfile(context.Background(), nil); !errors.Is(err, dictionary.ErrGRPCProfileRequestRequired) {
		t.Fatal(err)
	}
	if service.Calls != 0 {
		t.Fatal("guard dispatched")
	}
	if _, err := client.UpdateProfile(context.Background(), &profilev1.ProfileRequest{}); !errors.Is(err, dictionary.ErrGRPCProfileResponseRequired) {
		t.Fatal(err)
	}
	if service.Calls != 1 {
		t.Fatal("response guard call count")
	}
}
