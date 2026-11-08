package test_scenarios

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	sessionv1 "github.com/siti-nabila/api-contracts/pb/session/v1"
	"github.com/siti-nabila/rest-orc/internal/client/authservice"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/test/client/authservice/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func AdminSessionsScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "admin-session SDK guards payload context deadlines cancellation nil and error", Run: adminSessionClient}}
}

func adminSessionClient(t *testing.T) {
	for _, mode := range []string{"success", "nil", "error", "cancel", "timeout"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				transport := newTransport(t)
				ctx, cancel := context.WithTimeout(requestContext(), 100*time.Millisecond)
				defer cancel()
				deadline, _ := ctx.Deadline()
				if mode == "cancel" {
					cancel()
				}
				calls := 0
				var child context.Context
				check := func(c context.Context) error {
					child = c
					calls++
					assertRequestContext(t, c)
					d, _ := c.Deadline()
					if !d.Equal(deadline) {
						t.Fatal("deadline changed")
					}
					switch mode {
					case "error":
						return errDependency
					case "cancel":
						return c.Err()
					case "timeout":
						<-c.Done()
						return c.Err()
					}
					return nil
				}
				lq := &sessionv1.ListUserSessionsRequest{UserId: 23}
				rq := &sessionv1.RevokeUserSessionRequest{UserId: 23, SessionId: "raw"}
				sdk := &mocks.AdminSessionService{List: func(c context.Context, r *sessionv1.ListUserSessionsRequest) (*sessionv1.ListSessionsResponse, error) {
					if r != lq {
						t.Fatal("request changed")
					}
					e := check(c)
					if e != nil || mode == "nil" {
						return nil, e
					}
					return &sessionv1.ListSessionsResponse{}, nil
				}, Revoke: func(c context.Context, r *sessionv1.RevokeUserSessionRequest) (*sessionv1.RevokeSessionResponse, error) {
					if r != rq {
						t.Fatal("request changed")
					}
					e := check(c)
					if e != nil || mode == "nil" {
						return nil, e
					}
					return &sessionv1.RevokeSessionResponse{}, nil
				}, All: func(c context.Context, r *sessionv1.RevokeAllUserSessionsRequest) (*sessionv1.RevokeAllUserSessionsResponse, error) {
					if r == nil || r.ExpectedUserVersion == nil || *r.ExpectedUserVersion != 9007199254740993 {
						t.Fatal("missing target version")
					}
					e := check(c)
					if e != nil || mode == "nil" {
						return nil, e
					}
					return &sessionv1.RevokeAllUserSessionsResponse{}, nil
				}}
				selfSDK := &mocks.SelfSessionService{}
				service := failByDefaultService(t)
				profile := &mocks.ProfileService{}
				password := &mocks.PasswordService{}
				session := &mocks.SessionService{}
				for _, args := range []struct {
					transportNil, serviceNil, profileNil, passwordNil, sessionNil, selfNil bool
					want                                                                   error
				}{{true, true, true, true, true, true, dictionary.ErrGRPCAuthTransportRequired}, {false, true, true, true, true, true, dictionary.ErrGRPCAuthServiceRequired}, {false, false, true, true, true, true, dictionary.ErrGRPCProfileServiceRequired}, {false, false, false, true, true, true, dictionary.ErrGRPCPasswordServiceRequired}, {false, false, false, false, true, true, dictionary.ErrSessionDependencyRequired}, {false, false, false, false, false, true, dictionary.ErrSessionDependencyRequired}} {
					tr := transport
					if args.transportNil {
						tr = nil
					}
					var s authservice.Service = service
					if args.serviceNil {
						s = nil
					}
					var p authservice.ProfileService = profile
					if args.profileNil {
						p = nil
					}
					var pw authservice.PasswordService = password
					if args.passwordNil {
						pw = nil
					}
					var ss authservice.SessionService = session
					if args.sessionNil {
						ss = nil
					}
					var self authservice.SelfSessionService = selfSDK
					if args.selfNil {
						self = nil
					}
					if _, e := authservice.NewWithSelfSessionService(tr, s, p, pw, ss, self); !errors.Is(e, args.want) {
						t.Fatal("legacy constructor guard order")
					}
					if _, e := authservice.NewWithAdminSessionService(tr, s, p, pw, ss, self, nil); !errors.Is(e, args.want) {
						t.Fatal("admin constructor guard order", e)
					}
				}
				if _, e := authservice.NewWithAdminSessionService(transport, service, profile, password, session, selfSDK, nil); !errors.Is(e, dictionary.ErrSessionDependencyRequired) {
					t.Fatal(e)
				}
				client, e := authservice.NewWithAdminSessionService(transport, service, profile, password, session, selfSDK, sdk)
				if e != nil {
					t.Fatal(e)
				}
				for _, invoke := range []func() error{func() error { _, e := client.RevokeAllUserSessions(ctx, nil); return e }, func() error { _, e := client.ListUserSessions(nil, lq); return e }, func() error { _, e := client.ListUserSessions(ctx, nil); return e }, func() error { _, e := client.RevokeUserSession(nil, rq); return e }, func() error { _, e := client.RevokeUserSession(ctx, nil); return e }, func() error {
					_, e := client.RevokeAllUserSessions(nil, &sessionv1.RevokeAllUserSessionsRequest{UserId: 23, ExpectedUserVersion: adminVersion()})
					return e
				}} {
					if invoke() == nil {
						t.Fatal("guard failed")
					}
				}
				if calls != 0 {
					t.Fatal("guard dispatched")
				}
				for _, invoke := range []func() error{func() error { _, e := client.ListUserSessions(ctx, lq); return e }, func() error { _, e := client.RevokeUserSession(ctx, rq); return e }, func() error {
					_, e := client.RevokeAllUserSessions(ctx, &sessionv1.RevokeAllUserSessionsRequest{UserId: 23, ExpectedUserVersion: adminVersion()})
					return e
				}} {
					e := invoke()
					switch mode {
					case "success":
						if e != nil {
							t.Fatal(e)
						}
					case "nil":
						if !errors.Is(e, dictionary.ErrSessionResultInvalid) {
							t.Fatal(e)
						}
					case "error":
						if e != errDependency {
							t.Fatal(e)
						}
					case "cancel":
						if !errors.Is(e, context.Canceled) {
							t.Fatal(e)
						}
					case "timeout":
						if !errors.Is(e, context.DeadlineExceeded) {
							t.Fatal(e)
						}
					}
					if child.Err() == nil {
						t.Fatal("child not cleaned")
					}
				}
				if calls != 3 {
					t.Fatal("retries")
				}
			})
		})
	}
}

func adminVersion() *uint64 { v := uint64(9007199254740993); return &v }
