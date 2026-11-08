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

func UserProfileScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "target profile client additive constructor guard ordering", Run: targetProfileClientConstructors}, {Name: "target profile client SDK request metadata deadlines error and cleanup", Run: targetProfileClientRequests}}
}
func targetProfileClientConstructors(t *testing.T) {
	transport := newTransport(t)
	for index, want := range []error{dictionary.ErrGRPCAuthTransportRequired, dictionary.ErrGRPCAuthServiceRequired, dictionary.ErrGRPCProfileServiceRequired, dictionary.ErrGRPCPasswordServiceRequired, dictionary.ErrSessionDependencyRequired, dictionary.ErrSessionDependencyRequired, dictionary.ErrSessionDependencyRequired, dictionary.ErrUserDetailServiceRequired, dictionary.ErrUserStatusServiceRequired, dictionary.ErrUserProfileServiceRequired} {
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
		var status authservice.UserStatusService = &mocks.UserStatusService{}
		if index <= 8 {
			status = nil
		}
		if _, err := authservice.NewWithUserProfileService(tr, s, p, pw, session, self, admin, detail, status, nil); !errors.Is(err, want) {
			t.Fatal(index, err, want)
		}
	}
}
func targetProfileClientRequests(t *testing.T) {
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
				request := &userv1.UpdateUserRequest{UserId: 23}
				response := &userv1.UpdateUserResponse{}
				sdk := &mocks.UserProfileService{Update: func(c context.Context, r *userv1.UpdateUserRequest) (*userv1.UpdateUserResponse, error) {
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
				client, err := authservice.NewWithUserProfileService(newTransport(t), failByDefaultService(t), &mocks.ProfileService{}, &mocks.PasswordService{}, &mocks.SessionService{}, &mocks.SelfSessionService{}, &mocks.AdminSessionService{}, &mocks.UserDetailService{}, &mocks.UserStatusService{}, sdk)
				if err != nil {
					t.Fatal(err)
				}
				if _, err = client.UpdateUser(nil, request); !errors.Is(err, dictionary.ErrGRPCAuthRequestContextRequired) {
					t.Fatal(err)
				}
				if _, err = client.UpdateUser(ctx, nil); !errors.Is(err, dictionary.ErrUpdateUserRequestRequired) {
					t.Fatal(err)
				}
				if calls != 0 {
					t.Fatal("guard dispatched")
				}
				got, err := client.UpdateUser(ctx, request)
				switch mode {
				case "success":
					if got != response || err != nil {
						t.Fatal(got, err)
					}
				case "nil":
					if !errors.Is(err, dictionary.ErrUpdateUserResponseRequired) {
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
