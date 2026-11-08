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

func UserRoleScenarios() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "assignment SDK constructor preserves guard precedence", Run: userRoleClientConstructors},
		{Name: "assignment SDK methods context guards identity and cleanup", Run: userRoleClientRequests},
	}
}
func userRoleClientConstructors(t *testing.T) {
	transport := newTransport(t)
	for index, want := range []error{dictionary.ErrGRPCAuthTransportRequired, dictionary.ErrGRPCAuthServiceRequired, dictionary.ErrGRPCProfileServiceRequired, dictionary.ErrGRPCPasswordServiceRequired, dictionary.ErrSessionDependencyRequired, dictionary.ErrSessionDependencyRequired, dictionary.ErrSessionDependencyRequired, dictionary.ErrUserDetailServiceRequired, dictionary.ErrUserStatusServiceRequired, dictionary.ErrUserProfileServiceRequired, dictionary.ErrUserCreationServiceRequired, dictionary.ErrRoleCatalogServiceRequired, dictionary.ErrUserRoleManagementServiceRequired} {
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
		if _, err := authservice.NewWithUserRoleManagementService(tr, s, p, pw, session, self, admin, detail, status, userProfile, creation, roles, nil); !errors.Is(err, want) {
			t.Fatal(index, err, want)
		}
	}
}

func userRoleClientRequests(t *testing.T) {
	for _, method := range []string{"ListUserRoles", "GrantUserRoles", "RevokeUserRoles"} {
		for _, mode := range []string{"success", "nil", "error", "cancel", "timeout"} {
			t.Run(method+"/"+mode, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					ctx, cancel := context.WithTimeout(requestContext(), 100*time.Millisecond)
					defer cancel()
					deadline, _ := ctx.Deadline()
					if mode == "cancel" {
						cancel()
					}
					calls := 0
					var child context.Context
					inspect := func(c context.Context) error {
						calls++
						child = c
						assertRequestContext(t, c)
						d, _ := c.Deadline()
						if !d.Equal(deadline) {
							t.Fatal("deadline extended")
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
					sdk := &mocks.UserRoleManagementService{}
					reqListUserRoles := &userv1.ListUserRolesRequest{UserId: 23}
					respListUserRoles := &userv1.ListUserRolesResponse{}
					sdk.ListUserRolesFunc = func(c context.Context, r *userv1.ListUserRolesRequest) (*userv1.ListUserRolesResponse, error) {
						if method != "ListUserRoles" || r != reqListUserRoles {
							t.Fatal("wrong request")
						}
						e := inspect(c)
						if mode == "nil" {
							return nil, nil
						}
						return respListUserRoles, e
					}
					reqGrantUserRoles := &userv1.GrantUserRolesRequest{UserId: 23}
					respGrantUserRoles := &userv1.GrantUserRolesResponse{}
					sdk.GrantUserRolesFunc = func(c context.Context, r *userv1.GrantUserRolesRequest) (*userv1.GrantUserRolesResponse, error) {
						if method != "GrantUserRoles" || r != reqGrantUserRoles {
							t.Fatal("wrong request")
						}
						e := inspect(c)
						if mode == "nil" {
							return nil, nil
						}
						return respGrantUserRoles, e
					}
					reqRevokeUserRoles := &userv1.RevokeUserRolesRequest{UserId: 23}
					respRevokeUserRoles := &userv1.RevokeUserRolesResponse{}
					sdk.RevokeUserRolesFunc = func(c context.Context, r *userv1.RevokeUserRolesRequest) (*userv1.RevokeUserRolesResponse, error) {
						if method != "RevokeUserRoles" || r != reqRevokeUserRoles {
							t.Fatal("wrong request")
						}
						e := inspect(c)
						if mode == "nil" {
							return nil, nil
						}
						return respRevokeUserRoles, e
					}
					client, e := authservice.NewWithUserRoleManagementService(newTransport(t), failByDefaultService(t), &mocks.ProfileService{}, &mocks.PasswordService{}, &mocks.SessionService{}, &mocks.SelfSessionService{}, &mocks.AdminSessionService{}, &mocks.UserDetailService{}, &mocks.UserStatusService{}, &mocks.UserProfileService{}, &mocks.UserCreationService{}, &mocks.RoleCatalogService{}, sdk)
					if e != nil {
						t.Fatal(e)
					}
					if _, e = client.ListUserRoles(nil, reqListUserRoles); !errors.Is(e, dictionary.ErrGRPCAuthRequestContextRequired) {
						t.Fatal(e)
					}
					if _, e = client.ListUserRoles(ctx, nil); !errors.Is(e, dictionary.ErrListUserRolesRequestRequired) {
						t.Fatal(e)
					}
					if _, e = client.GrantUserRoles(nil, reqGrantUserRoles); !errors.Is(e, dictionary.ErrGRPCAuthRequestContextRequired) {
						t.Fatal(e)
					}
					if _, e = client.GrantUserRoles(ctx, nil); !errors.Is(e, dictionary.ErrGrantUserRolesRequestRequired) {
						t.Fatal(e)
					}
					if _, e = client.RevokeUserRoles(nil, reqRevokeUserRoles); !errors.Is(e, dictionary.ErrGRPCAuthRequestContextRequired) {
						t.Fatal(e)
					}
					if _, e = client.RevokeUserRoles(ctx, nil); !errors.Is(e, dictionary.ErrRevokeUserRolesRequestRequired) {
						t.Fatal(e)
					}
					if calls != 0 {
						t.Fatal("guard dispatch")
					}
					var got, want any
					var nilErr error
					switch method {
					case "ListUserRoles":
						got, e = client.ListUserRoles(ctx, reqListUserRoles)
						want = respListUserRoles
						nilErr = dictionary.ErrListUserRolesResponseRequired
					case "GrantUserRoles":
						got, e = client.GrantUserRoles(ctx, reqGrantUserRoles)
						want = respGrantUserRoles
						nilErr = dictionary.ErrGrantUserRolesResponseRequired
					case "RevokeUserRoles":
						got, e = client.RevokeUserRoles(ctx, reqRevokeUserRoles)
						want = respRevokeUserRoles
						nilErr = dictionary.ErrRevokeUserRolesResponseRequired
					}
					switch mode {
					case "success":
						if e != nil || got != want {
							t.Fatal(got, e)
						}
					case "nil":
						if !errors.Is(e, nilErr) {
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
					if calls != 1 || child == nil || child.Err() == nil {
						t.Fatal("dispatch/cleanup", calls)
					}
				})
			})
		}
	}
}
