package test_scenarios

import (
	"context"
	"errors"

	authv1 "github.com/siti-nabila/api-contracts/pb/auth/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/rest-orc/internal/client/authservice"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/test/client/authservice/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc"
	"testing"
	"testing/synctest"
	"time"
)

func SessionScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "session SDK wrappers preserve requests deadlines errors and cleanup", Run: sessionClient}}
}

func sessionClient(t *testing.T) {
	for _, mode := range []string{"success", "error", "nil", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(requestContext(), time.Millisecond*100)
				defer cancel()
				if mode == "cancel" {
					cancel()
				}
				deadline, _ := ctx.Deadline()
				calls := 0
				var child context.Context
				check := func(actual context.Context) error {
					child = actual
					calls++
					assertRequestContext(t, actual)
					d, _ := actual.Deadline()
					if !d.Equal(deadline) {
						t.Fatal("parent deadline lost")
					}
					if mode == "error" {
						return errDependency
					}
					if mode == "cancel" {
						return actual.Err()
					}
					return nil
				}
				rq := &authv1.RefreshSessionRequest{RefreshToken: "opaque"}
				proof := "refresh"
				lq := &authv1.LogoutRequest{RefreshToken: &proof}
				sdk := &mocks.SessionService{
					RefreshFunc: func(c context.Context, r *authv1.RefreshSessionRequest, _ ...grpc.CallOption) (*userv1.UserTokenResponse, error) {
						if r != rq {
							t.Fatal("request changed")
						}
						err := check(c)
						if err != nil || mode == "nil" {
							return nil, err
						}
						return &userv1.UserTokenResponse{Token: "result"}, nil
					},
					LogoutFunc: func(c context.Context, r *authv1.LogoutRequest, _ ...grpc.CallOption) (*authv1.LogoutResponse, error) {
						if r != lq {
							t.Fatal("presence changed")
						}
						err := check(c)
						if err != nil || mode == "nil" {
							return nil, err
						}
						return &authv1.LogoutResponse{}, nil
					},
				}
				transport := newTransport(t)
				client, err := authservice.NewWithSessionService(transport, failByDefaultService(t), &mocks.ProfileService{}, &mocks.PasswordService{}, sdk)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := authservice.NewWithSessionService(transport, failByDefaultService(t), &mocks.ProfileService{}, &mocks.PasswordService{}, nil); !errors.Is(err, dictionary.ErrSessionDependencyRequired) {
					t.Fatal("nil sdk accepted")
				}
				for _, invoke := range []func() error{func() error { _, e := client.RefreshSession(nil, rq); return e }, func() error { _, e := client.RefreshSession(ctx, nil); return e }, func() error { _, e := client.Logout(nil, lq); return e }, func() error { _, e := client.Logout(ctx, nil); return e }} {
					if invoke() == nil {
						t.Fatal("invalid request accepted")
					}
				}
				if calls != 0 {
					t.Fatal("guard dispatched")
				}
				for _, invoke := range []func() error{func() error { _, e := client.RefreshSession(ctx, rq); return e }, func() error { _, e := client.Logout(ctx, lq); return e }} {
					err := invoke()
					switch mode {
					case "success":
						if err != nil {
							t.Fatal(err)
						}
					case "nil":
						if !errors.Is(err, dictionary.ErrSessionResultInvalid) {
							t.Fatal("nil result accepted")
						}
					case "error":
						if !errors.Is(err, errDependency) {
							t.Fatal("identity lost")
						}
					case "cancel":
						if !errors.Is(err, context.Canceled) {
							t.Fatal("cancel lost")
						}
					}
					if child.Err() != context.Canceled {
						t.Fatal("child not cleaned")
					}
				}
				if calls != 2 {
					t.Fatal("unexpected retry")
				}
			})
		})
	}
}
