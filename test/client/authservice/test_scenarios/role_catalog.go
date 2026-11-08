package test_scenarios

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	rolev1 "github.com/siti-nabila/api-contracts/pb/role/v1"
	"github.com/siti-nabila/rest-orc/internal/client/authservice"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/test/client/authservice/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func RoleCatalogScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "role SDK additive constructor dependency order", Run: roleCatalogClientConstructors}, {Name: "role SDK request guards context errors and cleanup", Run: roleCatalogClientRequests}}
}
func roleCatalogClientConstructors(t *testing.T) {
	transport := newTransport(t)
	for index, want := range []error{dictionary.ErrGRPCAuthTransportRequired, dictionary.ErrGRPCAuthServiceRequired, dictionary.ErrGRPCProfileServiceRequired, dictionary.ErrGRPCPasswordServiceRequired, dictionary.ErrSessionDependencyRequired, dictionary.ErrSessionDependencyRequired, dictionary.ErrSessionDependencyRequired, dictionary.ErrUserDetailServiceRequired, dictionary.ErrUserStatusServiceRequired, dictionary.ErrUserProfileServiceRequired, dictionary.ErrUserCreationServiceRequired, dictionary.ErrRoleCatalogServiceRequired} {
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
		if _, err := authservice.NewWithRoleCatalogService(tr, s, p, pw, session, self, admin, detail, status, userProfile, creation, nil); !errors.Is(err, want) {
			t.Fatal(index, err, want)
		}
	}
}
func roleCatalogClientRequests(t *testing.T) {
	for _, method := range []string{"ListRoles", "GetRole"} {
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
					listReq, getReq := &rolev1.ListRolesRequest{}, &rolev1.GetRoleRequest{RoleId: 10}
					listResp, getResp := &rolev1.ListRolesResponse{}, &rolev1.GetRoleResponse{}
					sdk := &mocks.RoleCatalogService{List: func(c context.Context, r *rolev1.ListRolesRequest) (*rolev1.ListRolesResponse, error) {
						if method != "ListRoles" || r != listReq {
							t.Fatal("wrong request")
						}
						err := inspect(c)
						if mode == "nil" {
							return nil, nil
						}
						return listResp, err
					}, Get: func(c context.Context, r *rolev1.GetRoleRequest) (*rolev1.GetRoleResponse, error) {
						if method != "GetRole" || r != getReq {
							t.Fatal("wrong request")
						}
						err := inspect(c)
						if mode == "nil" {
							return nil, nil
						}
						return getResp, err
					}}
					client, err := authservice.NewWithRoleCatalogService(newTransport(t), failByDefaultService(t), &mocks.ProfileService{}, &mocks.PasswordService{}, &mocks.SessionService{}, &mocks.SelfSessionService{}, &mocks.AdminSessionService{}, &mocks.UserDetailService{}, &mocks.UserStatusService{}, &mocks.UserProfileService{}, &mocks.UserCreationService{}, sdk)
					if err != nil {
						t.Fatal(err)
					}
					if _, err = client.ListRoles(nil, listReq); !errors.Is(err, dictionary.ErrGRPCAuthRequestContextRequired) {
						t.Fatal(err)
					}
					if _, err = client.GetRole(nil, getReq); !errors.Is(err, dictionary.ErrGRPCAuthRequestContextRequired) {
						t.Fatal(err)
					}
					if _, err = client.ListRoles(ctx, nil); !errors.Is(err, dictionary.ErrListRolesRequestRequired) {
						t.Fatal(err)
					}
					if _, err = client.GetRole(ctx, nil); !errors.Is(err, dictionary.ErrGetRoleRequestRequired) {
						t.Fatal(err)
					}
					if calls != 0 {
						t.Fatal("guard dispatch")
					}
					if method == "ListRoles" {
						out, e := client.ListRoles(ctx, listReq)
						err = e
						if mode == "success" && out != listResp {
							t.Fatal("result identity")
						}
					} else {
						out, e := client.GetRole(ctx, getReq)
						err = e
						if mode == "success" && out != getResp {
							t.Fatal("result identity")
						}
					}
					switch mode {
					case "success":
						if err != nil {
							t.Fatal(err)
						}
					case "nil":
						want := dictionary.ErrListRolesResponseRequired
						if method == "GetRole" {
							want = dictionary.ErrGetRoleResponseRequired
						}
						if !errors.Is(err, want) {
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
}
