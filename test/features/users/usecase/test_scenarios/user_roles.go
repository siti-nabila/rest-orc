package test_scenarios

import (
	"context"
	"reflect"
	"testing"

	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/internal/features/users/usecase"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

type assignmentContextKey struct{}

type assignmentGateway struct {
	list   func(context.Context, uint64, domain.UserRoleListQuery) (domain.ListUserRolesResponse, error)
	grant  func(context.Context, domain.UserRoleCommand) (domain.UserRoleCommandResponse, error)
	revoke func(context.Context, domain.UserRoleCommand) (domain.UserRoleCommandResponse, error)
}

func (g *assignmentGateway) ListUserRoles(c context.Context, id uint64, q domain.UserRoleListQuery) (domain.ListUserRolesResponse, error) {
	return g.list(c, id, q)
}
func (g *assignmentGateway) GrantUserRoles(c context.Context, r domain.UserRoleCommand) (domain.UserRoleCommandResponse, error) {
	return g.grant(c, r)
}
func (g *assignmentGateway) RevokeUserRoles(c context.Context, r domain.UserRoleCommand) (domain.UserRoleCommandResponse, error) {
	return g.revoke(c, r)
}
func UserRoleScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "assignment usecase permissions precedence payload calls and isolation", Run: assignmentUsecases}}
}
func assignmentUsecases(t *testing.T) {
	if _, e := usecase.NewListUserRoles(nil); e == nil {
		t.Fatal("nil list")
	}
	if _, e := usecase.NewGrantUserRoles(nil); e == nil {
		t.Fatal("nil grant")
	}
	if _, e := usecase.NewRevokeUserRoles(nil); e == nil {
		t.Fatal("nil revoke")
	}
	for _, method := range []string{"list", "grant", "revoke"} {
		t.Run(method, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), assignmentContextKey{}, "assignment")
			calls := 0
			var failure error
			target := uint64(23)
			ids := []uint64{10}
			reason := " historical\té "
			permission := domain.UserRoleReadPermission
			if method == "grant" {
				permission = domain.UserRoleGrantPermission
			}
			if method == "revoke" {
				permission = domain.UserRoleRevokePermission
			}
			g := &assignmentGateway{list: func(c context.Context, id uint64, q domain.UserRoleListQuery) (domain.ListUserRolesResponse, error) {
				calls++
				if method != "list" || c != ctx || id != 23 || q.Page != 1 || q.Limit != 20 {
					t.Fatal(id, q)
				}
				return domain.ListUserRolesResponse{UserID: target, Version: "9007199254740993", Items: []domain.UserRoleAssignment{{RevokeReason: &reason}}}, failure
			}}
			inspect := func(c context.Context, r domain.UserRoleCommand) (domain.UserRoleCommandResponse, error) {
				calls++
				if c != ctx || r.UserID != 23 || !reflect.DeepEqual(r.RoleIDs, []uint64{10, 11}) || r.Reason != "approved\r\nreason" || r.Version != 9007199254740993 || r.IdempotencyKey != "opaque, key" {
					t.Fatal(r)
				}
				r.RoleIDs[0] = 99
				return domain.UserRoleCommandResponse{UserID: target, Version: "9007199254740993", ChangedRoleIDs: ids, Changed: len(ids) > 0}, failure
			}
			g.grant = func(c context.Context, r domain.UserRoleCommand) (domain.UserRoleCommandResponse, error) {
				if method != "grant" {
					t.Fatal("wrong method")
				}
				return inspect(c, r)
			}
			g.revoke = func(c context.Context, r domain.UserRoleCommand) (domain.UserRoleCommandResponse, error) {
				if method != "revoke" {
					t.Fatal("wrong method")
				}
				return inspect(c, r)
			}
			list, _ := usecase.NewListUserRoles(g)
			grant, _ := usecase.NewGrantUserRoles(g)
			revoke, _ := usecase.NewRevokeUserRoles(g)
			actor := domain.UserRoleActor{UserID: 17, Permissions: map[string]bool{permission: true}}
			rReason := "  approved\r\nreason  "
			request := domain.UserRoleCommandRequest{UserID: "00023", UserRoleCommandBody: domain.UserRoleCommandBody{RoleIDs: []string{"11", "00010"}, Reason: &rReason}, Version: 9007199254740993, IdempotencyKey: "opaque, key"}
			listRequest := domain.ListUserRolesRequest{UserID: "00023"}
			call := func(c context.Context, a domain.UserRoleActor) error {
				if method == "list" {
					out, e := list.Execute(c, a, listRequest)
					if e == nil {
						if out.Version != "9007199254740993" {
							t.Fatal(out)
						}
						*out.Items[0].RevokeReason = "mutated"
						if reason != " historical\té " {
							t.Fatal("pointer alias")
						}
					} else if out.UserID != 0 {
						t.Fatal("partial output")
					}
					return e
				}
				uc := grant.Execute
				if method == "revoke" {
					uc = revoke.Execute
				}
				out, e := uc(c, a, request)
				if e == nil {
					if out.Version != "9007199254740993" || out.Changed != (len(ids) > 0) || out.ChangedRoleIDs == nil {
						t.Fatal(out)
					}
					if len(out.ChangedRoleIDs) > 0 {
						out.ChangedRoleIDs[0] = 99
						if ids[0] != 10 {
							t.Fatal("slice alias")
						}
					}
				} else if out.UserID != 0 {
					t.Fatal("partial output")
				}
				return e
			}
			if e := call(nil, actor); e != common.ErrUnauthorized || calls != 0 {
				t.Fatal(e)
			}
			if e := call(ctx, domain.UserRoleActor{}); e != common.ErrUnauthorized || calls != 0 {
				t.Fatal(e)
			}
			for _, perms := range []map[string]bool{nil, {}, {permission: false}, {"*": true}, {"iam.user.*": true}, {"iam.role.read": true}, {"admin": true}} {
				a := actor
				a.Permissions = perms
				request.UserID = "bad"
				listRequest.UserID = "bad"
				if e := call(ctx, a); e != common.ErrForbidden || calls != 0 {
					t.Fatal(e)
				}
			}
			a := actor
			a.MustChangePassword = true
			if e := call(ctx, a); e != common.ErrForbidden || calls != 0 {
				t.Fatal(e)
			}
			if e := call(ctx, actor); e == nil || calls != 0 {
				t.Fatal("invalid business input")
			}
			request.UserID = "00023"
			listRequest.UserID = "00023"
			for _, selected := range [][]uint64{{10}, {}} {
				ids = selected
				for repeat := 0; repeat < 2; repeat++ {
					before := calls
					if e := call(ctx, actor); e != nil || calls != before+1 {
						t.Fatal(e, calls)
					}
				}
			}
			target = 99
			if e := call(ctx, actor); e != common.ErrInternalServerError {
				t.Fatal(e)
			}
			target = 23
			for _, f := range []error{common.ErrForbidden, common.ErrNotFound, common.ErrInternalServerError, context.Canceled, context.DeadlineExceeded} {
				failure = f
				before := calls
				if e := call(ctx, actor); e != f || calls != before+1 {
					t.Fatal(e)
				}
			}
			if request.RoleIDs[0] != "11" || rReason != "  approved\r\nreason  " {
				t.Fatal("request mutated")
			}
		})
	}
}
