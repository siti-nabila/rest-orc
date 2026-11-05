package test_scenarios

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/rest-orc/internal/client/authservice"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/test/client/authservice/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func UserStatusScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "status client additive constructor guard ordering", Run: statusClientConstructors}, {Name: "status client SDK request metadata deadlines error and cleanup", Run: statusClientRequests}}
}
func statusClientConstructors(t *testing.T) {
	transport := newTransport(t)
	for index, want := range []error{dictionary.ErrGRPCAuthTransportRequired, dictionary.ErrGRPCAuthServiceRequired, dictionary.ErrGRPCProfileServiceRequired, dictionary.ErrGRPCPasswordServiceRequired, dictionary.ErrSessionDependencyRequired, dictionary.ErrSessionDependencyRequired, dictionary.ErrSessionDependencyRequired, dictionary.ErrUserDetailServiceRequired, dictionary.ErrUserStatusServiceRequired} {
		tr := transport
		var s authservice.Service = failByDefaultService(t)
		var p authservice.ProfileService = &mocks.ProfileService{}
		var pw authservice.PasswordService = &mocks.PasswordService{}
		var session authservice.SessionService = &mocks.SessionService{}
		var self authservice.SelfSessionService = &mocks.SelfSessionService{}
		var admin authservice.AdminSessionService = &mocks.AdminSessionService{}
		// All later dependencies are nil so the first missing one controls the error.
		if index == 0 {
			tr = nil
		}
		if index <= 1 {
			s = nil
		}
		if index <= 2 {
			p = nil
		}
		if index <= 3 {
			pw = nil
		}
		if index <= 4 {
			session = nil
		}
		if index <= 5 {
			self = nil
		}
		if index <= 6 {
			admin = nil
		}
		var detail authservice.UserDetailService = &mocks.UserDetailService{}
		if index <= 7 {
			detail = nil
		}
		if _, err := authservice.NewWithUserStatusService(tr, s, p, pw, session, self, admin, detail, nil); !errors.Is(err, want) {
			t.Fatal(index, err, want)
		}
	}
}
func statusClientRequests(t *testing.T) {
	for _, mode := range []string{"success", "nil", "error", "cancel", "timeout"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(requestContext(), 100*time.Millisecond)
				defer cancel()
				deadline, _ := ctx.Deadline()
				if mode == "cancel" {
					cancel()
				}
				calls := 0
				var child context.Context
				request := &userv1.ChangeUserStatusRequest{UserId: 23}
				response := &userv1.ChangeUserStatusResponse{}
				sdk := &mocks.UserStatusService{Change: func(c context.Context, r *userv1.ChangeUserStatusRequest) (*userv1.ChangeUserStatusResponse, error) {
					calls++
					child = c
					assertRequestContext(t, c)
					d, _ := c.Deadline()
					if !d.Equal(deadline) || r != request {
						t.Fatal("context/request changed")
					}
					switch mode {
					case "nil":
						return nil, nil
					case "error":
						return nil, errDependency
					case "cancel":
						return nil, c.Err()
					case "timeout":
						<-c.Done()
						return nil, c.Err()
					}
					return response, nil
				}}
				client, err := authservice.NewWithUserStatusService(newTransport(t), failByDefaultService(t), &mocks.ProfileService{}, &mocks.PasswordService{}, &mocks.SessionService{}, &mocks.SelfSessionService{}, &mocks.AdminSessionService{}, &mocks.UserDetailService{}, sdk)
				if err != nil {
					t.Fatal(err)
				}
				if _, err = client.ChangeUserStatus(nil, request); !errors.Is(err, dictionary.ErrGRPCAuthRequestContextRequired) {
					t.Fatal(err)
				}
				if _, err = client.ChangeUserStatus(ctx, nil); !errors.Is(err, dictionary.ErrChangeUserStatusRequestRequired) {
					t.Fatal(err)
				}
				if calls != 0 {
					t.Fatal("guard dispatched")
				}
				got, err := client.ChangeUserStatus(ctx, request)
				switch mode {
				case "success":
					if got != response || err != nil {
						t.Fatal(got, err)
					}
				case "nil":
					if !errors.Is(err, dictionary.ErrChangeUserStatusResponseRequired) {
						t.Fatal(err)
					}
				case "error":
					if err != errDependency {
						t.Fatal(err)
					}
				case "cancel":
					if !errors.Is(err, context.Canceled) {
						t.Fatal(err)
					}
				case "timeout":
					if !errors.Is(err, context.DeadlineExceeded) {
						t.Fatal(err)
					}
				}
				if calls != 1 || child == nil || child.Err() == nil {
					t.Fatal("retry or missing cleanup")
				}
			})
		})
	}
}
