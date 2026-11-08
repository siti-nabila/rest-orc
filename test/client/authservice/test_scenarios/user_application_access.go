package test_scenarios

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	applicationv1 "github.com/siti-nabila/api-contracts/pb/application/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/rest-orc/internal/client/authservice"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/test/client/authservice/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func UserApplicationAccessScenarios() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "access SDK constructor guard compatibility", Run: accessClientConstructors},
		{Name: "access read SDK lifecycle", Run: accessClientListUserApplicationAccess},
		{Name: "access grant SDK lifecycle", Run: accessClientGrantApplicationAccess},
		{Name: "access revoke SDK lifecycle", Run: accessClientRevokeApplicationAccess},
	}
}
func newApplicationAccessClient(t *testing.T, sdk *mocks.ApplicationAccessService) (*authservice.Client, error) {
	t.Helper()
	return authservice.NewWithApplicationAccessServices(newTransport(t), failByDefaultService(t), &mocks.ProfileService{}, &mocks.PasswordService{}, &mocks.SessionService{}, &mocks.SelfSessionService{}, &mocks.AdminSessionService{}, &mocks.UserDetailService{}, &mocks.UserStatusService{}, &mocks.UserProfileService{}, &mocks.UserCreationService{}, &mocks.RoleCatalogService{}, &mocks.UserRoleManagementService{}, &mocks.ApplicationCatalogService{}, sdk, sdk)
}
func accessClientConstructors(t *testing.T) {
	transport := newTransport(t)
	for index, want := range []error{dictionary.ErrGRPCAuthTransportRequired, dictionary.ErrGRPCAuthServiceRequired, dictionary.ErrGRPCProfileServiceRequired, dictionary.ErrGRPCPasswordServiceRequired, dictionary.ErrSessionDependencyRequired, dictionary.ErrSessionDependencyRequired, dictionary.ErrSessionDependencyRequired, dictionary.ErrUserDetailServiceRequired, dictionary.ErrUserStatusServiceRequired, dictionary.ErrUserProfileServiceRequired, dictionary.ErrUserCreationServiceRequired, dictionary.ErrRoleCatalogServiceRequired, dictionary.ErrUserRoleManagementServiceRequired, dictionary.ErrApplicationCatalogServiceRequired, dictionary.ErrUserApplicationAccessServiceRequired, dictionary.ErrApplicationAccessServiceRequired} {
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
		var userProfile authservice.UserProfileService = &mocks.UserProfileService{}
		if index <= 9 {
			userProfile = nil
		}
		var creation authservice.UserCreationService = &mocks.UserCreationService{}
		if index <= 10 {
			creation = nil
		}
		var roles authservice.RoleCatalogService = &mocks.RoleCatalogService{}
		if index <= 11 {
			roles = nil
		}
		var assignments authservice.UserRoleManagementService = &mocks.UserRoleManagementService{}
		if index <= 12 {
			assignments = nil
		}
		var applications authservice.ApplicationCatalogService = &mocks.ApplicationCatalogService{}
		if index <= 13 {
			applications = nil
		}
		var reads authservice.UserApplicationAccessService = &mocks.ApplicationAccessService{}
		if index <= 14 {
			reads = nil
		}
		if _, err := authservice.NewWithApplicationAccessServices(tr, s, p, pw, session, self, admin, detail, status, userProfile, creation, roles, assignments, applications, reads, nil); !errors.Is(err, want) {
			t.Fatal(index, err, want)
		}
	}
}

func accessClientListUserApplicationAccess(t *testing.T) {
	for _, mode := range []string{"success", "nil", "error", "cancel", "timeout", "client deadline"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(requestContext(), 100*time.Millisecond)
				defer cancel()
				if mode == "client deadline" {
					ctx = requestContext()
				}
				if mode == "cancel" {
					cancel()
				}
				calls := 0
				var child context.Context
				req := &userv1.ListUserApplicationAccessRequest{}
				result := &userv1.ListUserApplicationAccessResponse{}
				sdk := &mocks.ApplicationAccessService{Read: func(c context.Context, r *userv1.ListUserApplicationAccessRequest) (*userv1.ListUserApplicationAccessResponse, error) {
					calls++
					child = c
					assertRequestContext(t, c)
					if r != req {
						t.Fatal("request identity")
					}
					d, ok := c.Deadline()
					if !ok {
						t.Fatal("missing deadline")
					}
					if parent, has := ctx.Deadline(); has && !d.Equal(parent) {
						t.Fatal("extended deadline")
					}
					switch mode {
					case "nil":
						return nil, nil
					case "error":
						return result, errDependency
					case "cancel":
						return result, c.Err()
					case "timeout", "client deadline":
						<-c.Done()
						return nil, c.Err()
					}
					return result, nil
				}}
				client, e := newApplicationAccessClient(t, sdk)
				if e != nil {
					t.Fatal(e)
				}
				if _, e = client.ListUserApplicationAccess(nil, req); !errors.Is(e, dictionary.ErrGRPCAuthRequestContextRequired) {
					t.Fatal(e)
				}
				if _, e = client.ListUserApplicationAccess(ctx, nil); !errors.Is(e, dictionary.ErrListUserApplicationAccessRequestRequired) {
					t.Fatal(e)
				}
				if calls != 0 {
					t.Fatal("guard dispatch")
				}
				got, e := client.ListUserApplicationAccess(ctx, req)
				switch mode {
				case "success":
					if e != nil || got != result {
						t.Fatal(got, e)
					}
				case "nil":
					if !errors.Is(e, dictionary.ErrListUserApplicationAccessResponseRequired) {
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
				case "timeout", "client deadline":
					if !errors.Is(e, context.DeadlineExceeded) {
						t.Fatal(e)
					}
				}
				if calls != 1 || child == nil || child.Err() == nil {
					t.Fatal("dispatch or cleanup", calls)
				}
			})
		})
	}
}
func accessClientGrantApplicationAccess(t *testing.T) {
	for _, mode := range []string{"success", "nil", "error", "cancel", "timeout", "client deadline"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(requestContext(), 100*time.Millisecond)
				defer cancel()
				if mode == "client deadline" {
					ctx = requestContext()
				}
				if mode == "cancel" {
					cancel()
				}
				calls := 0
				var child context.Context
				req := &applicationv1.GrantApplicationAccessRequest{}
				result := &applicationv1.GrantApplicationAccessResponse{}
				sdk := &mocks.ApplicationAccessService{Grant: func(c context.Context, r *applicationv1.GrantApplicationAccessRequest) (*applicationv1.GrantApplicationAccessResponse, error) {
					calls++
					child = c
					assertRequestContext(t, c)
					if r != req {
						t.Fatal("request identity")
					}
					d, ok := c.Deadline()
					if !ok {
						t.Fatal("missing deadline")
					}
					if parent, has := ctx.Deadline(); has && !d.Equal(parent) {
						t.Fatal("extended deadline")
					}
					switch mode {
					case "nil":
						return nil, nil
					case "error":
						return result, errDependency
					case "cancel":
						return result, c.Err()
					case "timeout", "client deadline":
						<-c.Done()
						return nil, c.Err()
					}
					return result, nil
				}}
				client, e := newApplicationAccessClient(t, sdk)
				if e != nil {
					t.Fatal(e)
				}
				if _, e = client.GrantApplicationAccess(nil, req); !errors.Is(e, dictionary.ErrGRPCAuthRequestContextRequired) {
					t.Fatal(e)
				}
				if _, e = client.GrantApplicationAccess(ctx, nil); !errors.Is(e, dictionary.ErrGrantApplicationAccessRequestRequired) {
					t.Fatal(e)
				}
				if calls != 0 {
					t.Fatal("guard dispatch")
				}
				got, e := client.GrantApplicationAccess(ctx, req)
				switch mode {
				case "success":
					if e != nil || got != result {
						t.Fatal(got, e)
					}
				case "nil":
					if !errors.Is(e, dictionary.ErrGrantApplicationAccessResponseRequired) {
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
				case "timeout", "client deadline":
					if !errors.Is(e, context.DeadlineExceeded) {
						t.Fatal(e)
					}
				}
				if calls != 1 || child == nil || child.Err() == nil {
					t.Fatal("dispatch or cleanup", calls)
				}
			})
		})
	}
}
func accessClientRevokeApplicationAccess(t *testing.T) {
	for _, mode := range []string{"success", "nil", "error", "cancel", "timeout", "client deadline"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(requestContext(), 100*time.Millisecond)
				defer cancel()
				if mode == "client deadline" {
					ctx = requestContext()
				}
				if mode == "cancel" {
					cancel()
				}
				calls := 0
				var child context.Context
				req := &applicationv1.RevokeApplicationAccessRequest{}
				result := &applicationv1.RevokeApplicationAccessResponse{}
				sdk := &mocks.ApplicationAccessService{Revoke: func(c context.Context, r *applicationv1.RevokeApplicationAccessRequest) (*applicationv1.RevokeApplicationAccessResponse, error) {
					calls++
					child = c
					assertRequestContext(t, c)
					if r != req {
						t.Fatal("request identity")
					}
					d, ok := c.Deadline()
					if !ok {
						t.Fatal("missing deadline")
					}
					if parent, has := ctx.Deadline(); has && !d.Equal(parent) {
						t.Fatal("extended deadline")
					}
					switch mode {
					case "nil":
						return nil, nil
					case "error":
						return result, errDependency
					case "cancel":
						return result, c.Err()
					case "timeout", "client deadline":
						<-c.Done()
						return nil, c.Err()
					}
					return result, nil
				}}
				client, e := newApplicationAccessClient(t, sdk)
				if e != nil {
					t.Fatal(e)
				}
				if _, e = client.RevokeApplicationAccess(nil, req); !errors.Is(e, dictionary.ErrGRPCAuthRequestContextRequired) {
					t.Fatal(e)
				}
				if _, e = client.RevokeApplicationAccess(ctx, nil); !errors.Is(e, dictionary.ErrRevokeApplicationAccessRequestRequired) {
					t.Fatal(e)
				}
				if calls != 0 {
					t.Fatal("guard dispatch")
				}
				got, e := client.RevokeApplicationAccess(ctx, req)
				switch mode {
				case "success":
					if e != nil || got != result {
						t.Fatal(got, e)
					}
				case "nil":
					if !errors.Is(e, dictionary.ErrRevokeApplicationAccessResponseRequired) {
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
				case "timeout", "client deadline":
					if !errors.Is(e, context.DeadlineExceeded) {
						t.Fatal(e)
					}
				}
				if calls != 1 || child == nil || child.Err() == nil {
					t.Fatal("dispatch or cleanup", calls)
				}
			})
		})
	}
}
