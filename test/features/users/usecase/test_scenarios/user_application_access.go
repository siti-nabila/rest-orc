package test_scenarios

import (
	"context"
	apperrors "github.com/siti-nabila/api-contracts/pkg/dictionary/application"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	roleerrors "github.com/siti-nabila/api-contracts/pkg/dictionary/role"
	usererrors "github.com/siti-nabila/api-contracts/pkg/dictionary/user"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/internal/features/users/usecase"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"reflect"
	"testing"
)

type accessGateway struct {
	list   func(context.Context, uint64, domain.UserApplicationAccessListQuery) (domain.ListUserApplicationAccessResponse, error)
	grant  func(context.Context, domain.GrantApplicationAccessCommand) (domain.ApplicationAccessCommandResponse, error)
	revoke func(context.Context, domain.ApplicationAccessCommand) (domain.ApplicationAccessCommandResponse, error)
}

func (g *accessGateway) ListUserApplicationAccess(c context.Context, id uint64, q domain.UserApplicationAccessListQuery) (domain.ListUserApplicationAccessResponse, error) {
	return g.list(c, id, q)
}
func (g *accessGateway) GrantApplicationAccess(c context.Context, r domain.GrantApplicationAccessCommand) (domain.ApplicationAccessCommandResponse, error) {
	return g.grant(c, r)
}
func (g *accessGateway) RevokeApplicationAccess(c context.Context, r domain.ApplicationAccessCommand) (domain.ApplicationAccessCommandResponse, error) {
	return g.revoke(c, r)
}
func UserApplicationAccessScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "access usecase AND authority order ownership compound outcomes", Run: accessUsecases}}
}
func accessUsecases(t *testing.T) {
	if _, e := usecase.NewListUserApplicationAccess(nil); e == nil {
		t.Fatal("list guard")
	}
	if _, e := usecase.NewGrantApplicationAccess(nil); e == nil {
		t.Fatal("grant guard")
	}
	if _, e := usecase.NewRevokeApplicationAccess(nil); e == nil {
		t.Fatal("revoke guard")
	}
	for _, op := range []string{"list", "grant", "revoke"} {
		t.Run(op, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), assignmentContextKey{}, "access")
			calls := 0
			target := uint64(23)
			var failure error
			reason := " historical\té "
			status := "ACTIVE"
			ids := []uint64{10}
			changed := true
			inspect := func(c context.Context, r domain.ApplicationAccessCommand) (domain.ApplicationAccessCommandResponse, error) {
				calls++
				if c != ctx || r.UserID != 23 || r.ApplicationCode != "clm" || r.Reason != "approved\r\nreason" || r.Version != 9007199254740993 || r.IdempotencyKey != "opaque, key" {
					t.Fatal(r)
				}
				return domain.ApplicationAccessCommandResponse{UserID: target, ApplicationID: 5, AccessStatus: &status, ChangedRoleIDs: ids, Version: "9007199254740993", Changed: changed}, failure
			}
			g := &accessGateway{list: func(c context.Context, id uint64, q domain.UserApplicationAccessListQuery) (domain.ListUserApplicationAccessResponse, error) {
				calls++
				if op != "list" || c != ctx || id != 23 || q.Page != 1 || q.Limit != 20 {
					t.Fatal(id, q)
				}
				return domain.ListUserApplicationAccessResponse{UserID: target, Version: "9007199254740993", Items: []domain.UserApplicationAccess{{RevokeReason: &reason, RevokedAt: &reason, GrantedBy: &target, RevokedBy: &target}}}, failure
			}, grant: func(c context.Context, r domain.GrantApplicationAccessCommand) (domain.ApplicationAccessCommandResponse, error) {
				if op != "grant" || !reflect.DeepEqual(r.RoleIDs, []uint64{10, 11}) {
					t.Fatal(r)
				}
				r.RoleIDs[0] = 99
				return inspect(c, r.ApplicationAccessCommand)
			}, revoke: func(c context.Context, r domain.ApplicationAccessCommand) (domain.ApplicationAccessCommandResponse, error) {
				if op != "revoke" {
					t.Fatal("wrong method")
				}
				return inspect(c, r)
			}}
			list, _ := usecase.NewListUserApplicationAccess(g)
			grant, _ := usecase.NewGrantApplicationAccess(g)
			revoke, _ := usecase.NewRevokeApplicationAccess(g)
			perms := map[string]bool{domain.UserApplicationReadPermission: true}
			if op == "grant" {
				perms = map[string]bool{domain.UserApplicationGrantPermission: true, domain.UserRoleGrantPermission: true}
			}
			if op == "revoke" {
				perms = map[string]bool{domain.UserApplicationRevokePermission: true, domain.UserRoleRevokePermission: true}
			}
			actor := domain.UserApplicationAccessActor{UserID: 17, Permissions: perms}
			rawReason := "  approved\r\nreason  "
			request := domain.ApplicationAccessCommandRequest{UserID: "00023", ApplicationCode: "clm", Version: 9007199254740993, IdempotencyKey: "opaque, key"}
			selected := []string{"11", "00010"}
			listRequest := domain.ListUserApplicationAccessRequest{UserID: "00023"}
			call := func(c context.Context, a domain.UserApplicationAccessActor) error {
				if op == "list" {
					out, e := list.Execute(c, a, listRequest)
					if e == nil {
						if out.Version != "9007199254740993" {
							t.Fatal(out)
						}
						*out.Items[0].RevokeReason = "mutated"
						*out.Items[0].RevokedAt = "mutated"
						*out.Items[0].GrantedBy = 99
						*out.Items[0].RevokedBy = 99
						if reason != " historical\té " || target != 23 {
							t.Fatal("list alias")
						}
					} else if out.UserID != 0 {
						t.Fatal("partial")
					}
					return e
				}
				var out domain.ApplicationAccessCommandResponse
				var e error
				if op == "grant" {
					out, e = grant.Execute(c, a, domain.GrantApplicationAccessRequest{ApplicationAccessCommandRequest: request, GrantApplicationAccessBody: domain.GrantApplicationAccessBody{RoleIDs: selected, Reason: &rawReason}})
				} else {
					out, e = revoke.Execute(c, a, domain.RevokeApplicationAccessRequest{ApplicationAccessCommandRequest: request, RevokeApplicationAccessBody: domain.RevokeApplicationAccessBody{Reason: &rawReason}})
				}
				if e == nil {
					if out.Version != "9007199254740993" || out.Changed != changed || out.ChangedRoleIDs == nil {
						t.Fatal(out)
					}
					*out.AccessStatus = "mutated"
					if status != "ACTIVE" {
						t.Fatal("status alias")
					}
					if len(ids) > 0 {
						out.ChangedRoleIDs[0] = 99
						if ids[0] == 99 {
							t.Fatal("roles alias")
						}
					}
				} else if out.UserID != 0 {
					t.Fatal("partial")
				}
				return e
			}
			if e := call(nil, actor); e != common.ErrUnauthorized || calls != 0 {
				t.Fatal(e)
			}
			if e := call(ctx, domain.UserApplicationAccessActor{}); e != common.ErrUnauthorized || calls != 0 {
				t.Fatal(e)
			}
			denials := []map[string]bool{nil, {}, {"*": true}, {"iam.user.*": true}, {"admin": true}, {"iam.permission.read": true}, {"iam.role.read": true}}
			for k := range perms {
				denials = append(denials, map[string]bool{k: false})
				if len(perms) == 2 {
					denials = append(denials, map[string]bool{k: true})
				}
			}
			request.UserID = "bad"
			listRequest.UserID = "bad"
			for _, p := range denials {
				a := actor
				a.Permissions = p
				if e := call(ctx, a); e != common.ErrForbidden || calls != 0 {
					t.Fatal(e, p)
				}
			}
			a := actor
			a.MustChangePassword = true
			if e := call(ctx, a); e != common.ErrForbidden || calls != 0 {
				t.Fatal(e)
			}
			if e := call(ctx, actor); e == nil || calls != 0 {
				t.Fatal("business guard")
			}
			request.UserID = "00023"
			listRequest.UserID = "00023"
			many := make([]uint64, 25)
			for i := range many {
				many[i] = uint64(i + 1)
			}
			for _, roles := range [][]uint64{{10}, {}, many} {
				ids = roles
				for _, ch := range []bool{true, false} {
					if !ch && len(roles) > 0 || op == "grant" && len(roles) > 20 {
						continue
					}
					changed = ch
					for repeat := 0; repeat < 2; repeat++ {
						before := calls
						if e := call(ctx, actor); e != nil || calls != before+1 {
							t.Fatal(e, calls)
						}
					}
				}
			}
			target = 99
			if e := call(ctx, actor); e != common.ErrInternalServerError {
				t.Fatal(e)
			}
			target = 23
			for _, f := range []error{common.ErrForbidden, common.ErrNotFound, common.ErrInternalServerError, context.Canceled, context.DeadlineExceeded, apperrors.ErrApplicationAccessSystemProtected, apperrors.ErrApplicationInactive, roleerrors.ErrRoleDisabled, usererrors.ErrGrantTargetInactive, usererrors.ErrVersionConflict, usererrors.ErrIdempotencyConflict} {
				failure = f
				before := calls
				if e := call(ctx, actor); e != f || calls != before+1 {
					t.Fatal(e)
				}
			}
			if selected[0] != "11" || rawReason != "  approved\r\nreason  " {
				t.Fatal("input alias")
			}
		})
	}
}
