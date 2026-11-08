package test_scenarios

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	applicationv1 "github.com/siti-nabila/api-contracts/pb/application/v1"
	paginatorv1 "github.com/siti-nabila/api-contracts/pb/paginator/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	gateway "github.com/siti-nabila/rest-orc/internal/features/users/gateway/authservice"
	"github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type applicationAccessClient struct {
	list   func(context.Context, *userv1.ListUserApplicationAccessRequest) (*userv1.ListUserApplicationAccessResponse, error)
	grant  func(context.Context, *applicationv1.GrantApplicationAccessRequest) (*applicationv1.GrantApplicationAccessResponse, error)
	revoke func(context.Context, *applicationv1.RevokeApplicationAccessRequest) (*applicationv1.RevokeApplicationAccessResponse, error)
}

func (s *applicationAccessClient) ListUserApplicationAccess(c context.Context, r *userv1.ListUserApplicationAccessRequest) (*userv1.ListUserApplicationAccessResponse, error) {
	return s.list(c, r)
}
func (s *applicationAccessClient) GrantApplicationAccess(c context.Context, r *applicationv1.GrantApplicationAccessRequest) (*applicationv1.GrantApplicationAccessResponse, error) {
	return s.grant(c, r)
}
func (s *applicationAccessClient) RevokeApplicationAccess(c context.Context, r *applicationv1.RevokeApplicationAccessRequest) (*applicationv1.RevokeApplicationAccessResponse, error) {
	return s.revoke(c, r)
}
func UserApplicationAccessScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "access history safe query metadata projection and malformed output", Run: applicationAccessListMapping}, {Name: "compound access outcomes and payload boundaries", Run: applicationAccessCommands}}
}
func applicationAccessListMapping(t *testing.T) {
	if _, e := gateway.NewUserApplicationAccess(nil); e == nil {
		t.Fatal("constructor")
	}
	ctx, md := assignmentParent()
	calls := 0
	r := fixtures.UserAccessHistory(23)
	var failure error
	q := domain.UserApplicationAccessListQuery{Page: 2, Limit: 30, ApplicationCode: "clm", Keyword: "a b", LastID: " x+/= ", Statuses: []string{"ACTIVE", "REVOKED"}}
	g, _ := gateway.NewUserApplicationAccess(&applicationAccessClient{list: func(c context.Context, in *userv1.ListUserApplicationAccessRequest) (*userv1.ListUserApplicationAccessResponse, error) {
		calls++
		checkAssignmentMetadata(t, c, md, "")
		if in.UserId != 23 || in.Query.Page != 2 || in.Query.Limit != 30 || in.Query.LastId != q.LastID || len(in.Query.Fields) != 0 || len(in.Query.Sort) != 1 || in.Query.Sort[0].Field != "application_id" || in.Query.Sort[0].Desc || !reflect.DeepEqual(in.Query.Search.Fields, []string{"application_code", "name"}) || in.Query.Search.Keyword != "a b" || in.Query.Search.Mode != paginatorv1.SearchMode_SEARCH_MODE_CONTAINS || !reflect.DeepEqual(in.Filter.Statuses, q.Statuses) || in.Filter.ApplicationCode != "clm" {
			t.Fatal(in)
		}
		in.Filter.Statuses[0] = "mutated"
		return r, failure
	}})
	if _, e := g.ListUserApplicationAccess(nil, 23, q); e == nil {
		t.Fatal("nil context")
	}
	if _, e := g.ListUserApplicationAccess(ctx, 0, q); e == nil {
		t.Fatal("bad target")
	}
	invalid := q
	invalid.Limit = 0
	if _, e := g.ListUserApplicationAccess(ctx, 23, invalid); e == nil || calls != 0 {
		t.Fatal("guard dispatch")
	}
	r.Meta = &paginatorv1.PageMeta{Page: 20, Limit: 30, Total: 1 << 32, TotalPages: 10, HasPrev: true}
	r.NextCursor = "opaque"
	got, e := g.ListUserApplicationAccess(ctx, 23, q)
	if e != nil || got.Total != 1<<32 || got.Version != "9007199254740993" || got.Page != 20 || !got.HasPrev || got.HasNext || got.NextCursor != "opaque" {
		t.Fatal(got, e)
	}
	b, _ := json.Marshal(got.Items[0])
	if string(b) != fixtures.UserAccessHistoryJSON {
		t.Fatal(string(b))
	}
	granted := uint64(17)
	revoked := uint64(18)
	reason := " legacy\té\n "
	r.Items[0].GrantedBy = &granted
	r.Items[0].RevokedBy = &revoked
	r.Items[0].RevokeReason = &reason
	r.Items[0].RevokedAt = timestamppb.New(fixtures.MeCreatedAt())
	got, e = g.ListUserApplicationAccess(ctx, 23, q)
	if e != nil || *got.Items[0].GrantedBy != 17 || *got.Items[0].RevokeReason != reason || got.Items[0].RevokedAt == nil {
		t.Fatal(got, e)
	}
	*got.Items[0].GrantedBy = 99
	*got.Items[0].RevokeReason = "changed"
	if granted != 17 || reason != " legacy\té\n " {
		t.Fatal("pointer alias")
	}
	for _, total := range []int64{0, 42, 9223372036854775807} {
		r.Items = nil
		r.Meta.Total = total
		r.Meta.TotalPages = 0
		got, e = g.ListUserApplicationAccess(ctx, 23, q)
		b, _ = json.Marshal(got.Items)
		if e != nil || string(b) != "[]" || got.Total != total {
			t.Fatal(got, e)
		}
	}
	mutations := []func(*userv1.ListUserApplicationAccessResponse){
		func(r *userv1.ListUserApplicationAccessResponse) { r.Meta = nil }, func(r *userv1.ListUserApplicationAccessResponse) { r.UserId = 99 }, func(r *userv1.ListUserApplicationAccessResponse) { r.UserId = 0 }, func(r *userv1.ListUserApplicationAccessResponse) { r.Version = 0 }, func(r *userv1.ListUserApplicationAccessResponse) { r.Version = 1 << 63 }, func(r *userv1.ListUserApplicationAccessResponse) { r.Meta.Page = 0 }, func(r *userv1.ListUserApplicationAccessResponse) { r.Meta.Limit = 0 }, func(r *userv1.ListUserApplicationAccessResponse) { r.Meta.Limit = 101 }, func(r *userv1.ListUserApplicationAccessResponse) { r.Meta.Total = -1 }, func(r *userv1.ListUserApplicationAccessResponse) { r.Meta.TotalPages = -1 }, func(r *userv1.ListUserApplicationAccessResponse) { r.Items = append(r.Items, nil) },
	}
	for _, mutate := range mutations {
		r = fixtures.UserAccessHistory(23)
		mutate(r)
		got, e = g.ListUserApplicationAccess(ctx, 23, q)
		if !errors.Is(e, common.ErrInternalServerError) || got.UserID != 0 {
			t.Fatal(got, e)
		}
	}
	for _, mutate := range []func(*userv1.UserApplicationAccessItem){
		func(i *userv1.UserApplicationAccessItem) { i.ApplicationId = 0 }, func(i *userv1.UserApplicationAccessItem) { i.ApplicationId = 1 << 31 }, func(i *userv1.UserApplicationAccessItem) { i.Status = "DISABLED" }, func(i *userv1.UserApplicationAccessItem) { i.ApplicationStatus = "REVOKED" }, func(i *userv1.UserApplicationAccessItem) { i.GrantedAt = nil }, func(i *userv1.UserApplicationAccessItem) { i.GrantedAt = &timestamppb.Timestamp{Nanos: -1} }, func(i *userv1.UserApplicationAccessItem) { i.RevokedAt = &timestamppb.Timestamp{Seconds: 253402300800} }, func(i *userv1.UserApplicationAccessItem) { v := uint64(0); i.GrantedBy = &v }, func(i *userv1.UserApplicationAccessItem) { v := uint64(1 << 31); i.RevokedBy = &v },
	} {
		r = fixtures.UserAccessHistory(23)
		mutate(r.Items[0])
		got, e = g.ListUserApplicationAccess(ctx, 23, q)
		if !errors.Is(e, common.ErrInternalServerError) || got.UserID != 0 {
			t.Fatal(got, e)
		}
	}
	r = nil
	if _, e = g.ListUserApplicationAccess(ctx, 23, q); !errors.Is(e, common.ErrInternalServerError) {
		t.Fatal(e)
	}
	for _, f := range []error{context.Canceled, context.DeadlineExceeded, common.ErrForbidden, common.ErrNotFound, common.ErrInternalServerError} {
		failure = f
		before := calls
		got, e = g.ListUserApplicationAccess(ctx, 23, q)
		if e != f || got.UserID != 0 || calls != before+1 {
			t.Fatal(got, e)
		}
	}
	if md.Get("x-request-id")[0] != "assignment" || q.Statuses[0] != "ACTIVE" || len(md.Get("idempotency-key")) != 2 {
		t.Fatal("parent mutated")
	}
}

func applicationAccessCommands(t *testing.T) {
	for _, op := range []string{"grant", "revoke"} {
		t.Run(op, func(t *testing.T) {
			ctx, md := assignmentParent()
			calls := 0
			input := domain.GrantApplicationAccessCommand{ApplicationAccessCommand: domain.ApplicationAccessCommand{UserID: 23, ApplicationCode: "clm", Reason: "approved\r\nreason", Version: 9007199254740993, IdempotencyKey: "opaque, key"}, RoleIDs: []uint64{10, 11}}
			state := "ACTIVE"
			if op == "revoke" {
				state = "REVOKED"
			}
			result := &applicationv1.GrantApplicationAccessResponse{UserId: 23, ApplicationId: 5, AccessStatus: &state, Version: 9223372036854775807}
			var failure error
			inspect := func(c context.Context, id uint64, code, reason string, version *uint64) {
				calls++
				checkAssignmentMetadata(t, c, md, input.IdempotencyKey)
				if id != 23 || code != "clm" || reason != input.Reason || version == nil || *version != input.Version {
					t.Fatal(id, code, reason, version)
				}
				*version = 1
			}
			sdk := &applicationAccessClient{grant: func(c context.Context, r *applicationv1.GrantApplicationAccessRequest) (*applicationv1.GrantApplicationAccessResponse, error) {
				if op != "grant" || !reflect.DeepEqual(r.RoleIds, input.RoleIDs) {
					t.Fatal(r)
				}
				inspect(c, r.UserId, r.ApplicationCode, r.Reason, r.ExpectedUserVersion)
				r.RoleIds[0] = 99
				return result, failure
			}, revoke: func(c context.Context, r *applicationv1.RevokeApplicationAccessRequest) (*applicationv1.RevokeApplicationAccessResponse, error) {
				if op != "revoke" {
					t.Fatal("wrong RPC")
				}
				inspect(c, r.UserId, r.ApplicationCode, r.Reason, r.ExpectedUserVersion)
				if result == nil {
					return nil, failure
				}
				return &applicationv1.RevokeApplicationAccessResponse{UserId: result.UserId, ApplicationId: result.ApplicationId, AccessStatus: result.AccessStatus, ChangedRoleIds: result.ChangedRoleIds, Version: result.Version, Changed: result.Changed}, failure
			}}
			g, _ := gateway.NewUserApplicationAccess(sdk)
			call := func(c context.Context, r domain.GrantApplicationAccessCommand) (domain.ApplicationAccessCommandResponse, error) {
				if op == "grant" {
					return g.GrantApplicationAccess(c, r)
				}
				return g.RevokeApplicationAccess(c, r.ApplicationAccessCommand)
			}
			if _, e := call(nil, input); e == nil {
				t.Fatal("nil context")
			}
			bad := input
			bad.ApplicationCode = "!"
			if _, e := call(ctx, bad); e == nil || calls != 0 {
				t.Fatal("invalid dispatch")
			}
			many := make([]uint64, 25)
			for i := range many {
				many[i] = uint64(i + 1)
			}
			lists := [][]uint64{nil, {}, {10}, {10, 11}}
			if op == "revoke" {
				lists = append(lists, many)
			}
			for _, ids := range lists {
				for _, changed := range []bool{false, true} {
					if !changed && len(ids) > 0 {
						continue
					}
					result = &applicationv1.GrantApplicationAccessResponse{UserId: 23, ApplicationId: 5, AccessStatus: &state, Version: 9223372036854775807, ChangedRoleIds: ids, Changed: changed}
					for repeat := 0; repeat < 2; repeat++ {
						before := calls
						out, e := call(ctx, input)
						if e != nil || calls != before+1 || out.UserID != 23 || out.ApplicationID != 5 || out.Changed != changed || out.Version != "9223372036854775807" || out.ChangedRoleIDs == nil || len(out.ChangedRoleIDs) != len(ids) || out.AccessStatus == nil || *out.AccessStatus != state {
							t.Fatal(out, e)
						}
						*out.AccessStatus = "modified"
						if len(ids) > 0 {
							out.ChangedRoleIDs[0] = 99
							if ids[0] == 99 {
								t.Fatal("result alias")
							}
						}
						if state != result.GetAccessStatus() {
							t.Fatal("status alias")
						}
					}
				}
			}
			if op == "revoke" {
				result = &applicationv1.GrantApplicationAccessResponse{UserId: 23, ApplicationId: 5, Version: 1, ChangedRoleIds: many, Changed: true}
				out, e := call(ctx, input)
				if e != nil || out.AccessStatus != nil || len(out.ChangedRoleIDs) != 25 {
					t.Fatal(out, e)
				}
			}
			for _, mutate := range []func(*applicationv1.GrantApplicationAccessResponse){
				func(r *applicationv1.GrantApplicationAccessResponse) { r.UserId = 0 }, func(r *applicationv1.GrantApplicationAccessResponse) { r.UserId = 99 }, func(r *applicationv1.GrantApplicationAccessResponse) { r.ApplicationId = 0 }, func(r *applicationv1.GrantApplicationAccessResponse) { r.ApplicationId = 1 << 31 }, func(r *applicationv1.GrantApplicationAccessResponse) { r.Version = 0 }, func(r *applicationv1.GrantApplicationAccessResponse) { r.Version = 1 << 63 }, func(r *applicationv1.GrantApplicationAccessResponse) { s := "DISABLED"; r.AccessStatus = &s }, func(r *applicationv1.GrantApplicationAccessResponse) {
					r.ChangedRoleIds = []uint64{10}
					r.Changed = false
				}, func(r *applicationv1.GrantApplicationAccessResponse) { r.ChangedRoleIds = []uint64{0} }, func(r *applicationv1.GrantApplicationAccessResponse) { r.ChangedRoleIds = []uint64{1 << 31} }, func(r *applicationv1.GrantApplicationAccessResponse) { r.ChangedRoleIds = []uint64{10, 10} }, func(r *applicationv1.GrantApplicationAccessResponse) { r.ChangedRoleIds = []uint64{11, 10} },
			} {
				result = &applicationv1.GrantApplicationAccessResponse{UserId: 23, ApplicationId: 5, AccessStatus: &state, Version: 1, Changed: true}
				mutate(result)
				out, e := call(ctx, input)
				if !errors.Is(e, common.ErrInternalServerError) || out.UserID != 0 {
					t.Fatal(out, e)
				}
			}
			if op == "grant" {
				for _, status := range []*string{nil, new(string)} {
					result = &applicationv1.GrantApplicationAccessResponse{UserId: 23, ApplicationId: 5, Version: 1, AccessStatus: status}
					if _, e := call(ctx, input); e == nil {
						t.Fatal("invalid grant status")
					}
				}
				result = &applicationv1.GrantApplicationAccessResponse{UserId: 23, ApplicationId: 5, Version: 1, AccessStatus: &state, Changed: true, ChangedRoleIds: []uint64{12}}
				if _, e := call(ctx, input); e == nil {
					t.Fatal("unrequested grant role")
				}
			}
			result = nil
			if _, e := call(ctx, input); !errors.Is(e, common.ErrInternalServerError) {
				t.Fatal(e)
			}
			for _, f := range []error{context.Canceled, context.DeadlineExceeded, common.ErrForbidden, common.ErrNotFound, common.ErrInternalServerError} {
				failure = f
				before := calls
				out, e := call(ctx, input)
				if e != f || calls != before+1 || out.UserID != 0 {
					t.Fatal(out, e)
				}
			}
			if input.RoleIDs[0] != 10 || input.Version != 9007199254740993 || md.Get("x-request-id")[0] != "assignment" {
				t.Fatal("input alias")
			}
		})
	}
	// Empty search is absent, and full projection does not suppress safe history fields.
	g, _ := gateway.NewUserApplicationAccess(&applicationAccessClient{list: func(c context.Context, r *userv1.ListUserApplicationAccessRequest) (*userv1.ListUserApplicationAccessResponse, error) {
		if r.Query.Search != nil || len(r.Query.Fields) != 0 {
			t.Fatal(r)
		}
		out := fixtures.UserAccessHistory(23)
		out.Version = 9223372036854775807
		out.Items[0].Status = "ACTIVE"
		out.Items[0].IsSystem = true
		return out, nil
	}})
	q, _ := domain.ParseUserApplicationAccessListQuery(nil)
	out, e := g.ListUserApplicationAccess(context.Background(), 23, q)
	if e != nil || out.Version != "9223372036854775807" || !out.Items[0].IsSystem {
		t.Fatal(out, e)
	}
}
