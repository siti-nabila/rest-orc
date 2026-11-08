package test_scenarios

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	paginatorv1 "github.com/siti-nabila/api-contracts/pb/paginator/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	gateway "github.com/siti-nabila/rest-orc/internal/features/users/gateway/authservice"
	"github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type assignmentClient struct {
	list   func(context.Context, *userv1.ListUserRolesRequest) (*userv1.ListUserRolesResponse, error)
	grant  func(context.Context, *userv1.GrantUserRolesRequest) (*userv1.GrantUserRolesResponse, error)
	revoke func(context.Context, *userv1.RevokeUserRolesRequest) (*userv1.RevokeUserRolesResponse, error)
}

func (s *assignmentClient) ListUserRoles(c context.Context, r *userv1.ListUserRolesRequest) (*userv1.ListUserRolesResponse, error) {
	return s.list(c, r)
}
func (s *assignmentClient) GrantUserRoles(c context.Context, r *userv1.GrantUserRolesRequest) (*userv1.GrantUserRolesResponse, error) {
	return s.grant(c, r)
}
func (s *assignmentClient) RevokeUserRoles(c context.Context, r *userv1.RevokeUserRolesRequest) (*userv1.RevokeUserRolesResponse, error) {
	return s.revoke(c, r)
}
func UserRoleScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "assignment list projection precision nulls metadata and malformed outputs", Run: assignmentListMapping}, {Name: "assignment commands exact payload owned data no-op replay and malformed outputs", Run: assignmentCommandMapping}}
}
func assignmentParent() (context.Context, metadata.MD) {
	md := metadata.Pairs("authorization", "Bearer actor", "version", "old", "idempotency-key", "old", "idempotency-key", "old2", "x-request-id", "assignment")
	return metadata.NewOutgoingContext(context.Background(), md), md
}
func checkAssignmentMetadata(t *testing.T, c context.Context, md metadata.MD, key string) {
	t.Helper()
	got, _ := metadata.FromOutgoingContext(c)
	want := md.Copy()
	want.Delete("version")
	want.Delete("idempotency-key")
	if key != "" {
		want.Set("idempotency-key", key)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatal(got, want)
	}
	got.Set("x-request-id", "mutated")
}
func assignmentListMapping(t *testing.T) {
	if _, e := gateway.NewUserRoleManagement(nil); e == nil {
		t.Fatal("constructor")
	}
	ctx, md := assignmentParent()
	calls := 0
	r := fixtures.UserRoleList(23)
	var failure error
	q := domain.UserRoleListQuery{Page: 2, Limit: 30, ApplicationCode: "clm", Keyword: "a b", LastID: " x+/= ", Statuses: []string{"ACTIVE", "REVOKED"}}
	g, _ := gateway.NewUserRoleManagement(&assignmentClient{list: func(c context.Context, in *userv1.ListUserRolesRequest) (*userv1.ListUserRolesResponse, error) {
		calls++
		checkAssignmentMetadata(t, c, md, "")
		if in.UserId != 23 || in.Query.Page != 2 || in.Query.Limit != 30 || in.Query.LastId != q.LastID || len(in.Query.Fields) != 0 || len(in.Query.Sort) != 1 || in.Query.Sort[0].Field != "id" || in.Query.Sort[0].Desc || !reflect.DeepEqual(in.Query.Search.Fields, []string{"keyword"}) || in.Query.Search.Keyword != "a b" || in.Query.Search.Mode != paginatorv1.SearchMode_SEARCH_MODE_FULL_TEXT_TRIGRAM || !reflect.DeepEqual(in.Filter.Statuses, q.Statuses) || in.Filter.ApplicationCode != "clm" {
			t.Fatal(in)
		}
		in.Filter.Statuses[0] = "mutated"
		return r, failure
	}})
	if _, e := g.ListUserRoles(nil, 23, q); e == nil {
		t.Fatal("nil context")
	}
	if _, e := g.ListUserRoles(ctx, 0, q); e == nil {
		t.Fatal("bad target")
	}
	invalid := q
	invalid.Limit = 0
	if _, e := g.ListUserRoles(ctx, 23, invalid); e == nil || calls != 0 {
		t.Fatal("guard dispatch")
	}
	r.Meta = &paginatorv1.PageMeta{Page: 20, Limit: 30, Total: 1 << 32, TotalPages: 10, HasPrev: true}
	r.NextCursor = "opaque"
	got, e := g.ListUserRoles(ctx, 23, q)
	if e != nil || got.Total != 1<<32 || got.Version != "9007199254740993" || got.Page != 20 || !got.HasPrev || got.HasNext || got.NextCursor != "opaque" {
		t.Fatal(got, e)
	}
	b, _ := json.Marshal(got.Items[0])
	if string(b) != fixtures.UserRoleAssignmentJSON {
		t.Fatal(string(b))
	}
	granted := uint64(17)
	revoked := uint64(18)
	reason := " legacy\té\n "
	access := "REVOKED"
	r.Items[0].GrantedBy = &granted
	r.Items[0].RevokedBy = &revoked
	r.Items[0].RevokeReason = &reason
	r.Items[0].ApplicationAccessStatus = &access
	r.Items[0].RevokedAt = timestamppb.New(fixtures.MeCreatedAt())
	got, e = g.ListUserRoles(ctx, 23, q)
	if e != nil || *got.Items[0].GrantedBy != 17 || *got.Items[0].RevokeReason != reason || got.Items[0].RevokedAt == nil {
		t.Fatal(got, e)
	}
	*got.Items[0].GrantedBy = 99
	*got.Items[0].RevokeReason = "changed"
	*got.Items[0].ApplicationAccessStatus = "ACTIVE"
	if granted != 17 || reason != " legacy\té\n " || access != "REVOKED" {
		t.Fatal("pointer alias")
	}
	for _, total := range []int64{0, 42} {
		r.Items = nil
		r.Meta.Total = total
		r.Meta.TotalPages = 0
		got, e = g.ListUserRoles(ctx, 23, q)
		b, _ = json.Marshal(got.Items)
		if e != nil || string(b) != "[]" || got.Total != total {
			t.Fatal(got, e)
		}
	}
	mutations := []func(*userv1.ListUserRolesResponse){
		func(r *userv1.ListUserRolesResponse) { r.Meta = nil }, func(r *userv1.ListUserRolesResponse) { r.UserId = 99 }, func(r *userv1.ListUserRolesResponse) { r.UserId = 0 }, func(r *userv1.ListUserRolesResponse) { r.Version = 0 }, func(r *userv1.ListUserRolesResponse) { r.Version = 1 << 63 }, func(r *userv1.ListUserRolesResponse) { r.Meta.Page = 0 }, func(r *userv1.ListUserRolesResponse) { r.Meta.Limit = 0 }, func(r *userv1.ListUserRolesResponse) { r.Meta.Limit = 101 }, func(r *userv1.ListUserRolesResponse) { r.Meta.Total = -1 }, func(r *userv1.ListUserRolesResponse) { r.Meta.TotalPages = -1 }, func(r *userv1.ListUserRolesResponse) { r.Items = append(r.Items, nil) },
	}
	for _, mutate := range mutations {
		r = fixtures.UserRoleList(23)
		mutate(r)
		got, e = g.ListUserRoles(ctx, 23, q)
		if !errors.Is(e, common.ErrInternalServerError) || got.UserID != 0 {
			t.Fatal(got, e)
		}
	}
	for _, mutate := range []func(*userv1.UserRoleAssignmentItem){
		func(i *userv1.UserRoleAssignmentItem) { i.Id = 0 }, func(i *userv1.UserRoleAssignmentItem) { i.Id = 1 << 31 }, func(i *userv1.UserRoleAssignmentItem) { i.RoleId = 0 }, func(i *userv1.UserRoleAssignmentItem) { i.RoleId = 1 << 31 }, func(i *userv1.UserRoleAssignmentItem) { i.ApplicationId = 0 }, func(i *userv1.UserRoleAssignmentItem) { i.ApplicationId = 1 << 31 }, func(i *userv1.UserRoleAssignmentItem) { i.RoleCode = 0 }, func(i *userv1.UserRoleAssignmentItem) { i.Status = "DISABLED" }, func(i *userv1.UserRoleAssignmentItem) { i.RoleStatus = "REVOKED" }, func(i *userv1.UserRoleAssignmentItem) { i.ApplicationStatus = "REVOKED" }, func(i *userv1.UserRoleAssignmentItem) { s := "DISABLED"; i.ApplicationAccessStatus = &s }, func(i *userv1.UserRoleAssignmentItem) { v := uint64(0); i.GrantedBy = &v }, func(i *userv1.UserRoleAssignmentItem) { v := uint64(1 << 31); i.RevokedBy = &v }, func(i *userv1.UserRoleAssignmentItem) { i.GrantedAt = &timestamppb.Timestamp{Nanos: -1} }, func(i *userv1.UserRoleAssignmentItem) { i.RevokedAt = &timestamppb.Timestamp{Seconds: 253402300800} },
	} {
		r = fixtures.UserRoleList(23)
		mutate(r.Items[0])
		got, e = g.ListUserRoles(ctx, 23, q)
		if !errors.Is(e, common.ErrInternalServerError) || got.UserID != 0 {
			t.Fatal(got, e)
		}
	}
	r = nil
	if _, e = g.ListUserRoles(ctx, 23, q); !errors.Is(e, common.ErrInternalServerError) {
		t.Fatal(e)
	}
	for _, f := range []error{context.Canceled, context.DeadlineExceeded, common.ErrForbidden, common.ErrNotFound, common.ErrInternalServerError} {
		failure = f
		before := calls
		got, e = g.ListUserRoles(ctx, 23, q)
		if e != f || got.UserID != 0 || calls != before+1 {
			t.Fatal(got, e)
		}
	}
	if md.Get("x-request-id")[0] != "assignment" || q.Statuses[0] != "ACTIVE" || len(md.Get("idempotency-key")) != 2 {
		t.Fatal("parent mutated")
	}
}
func assignmentCommandMapping(t *testing.T) {
	for _, method := range []string{"grant", "revoke"} {
		t.Run(method, func(t *testing.T) {
			ctx, md := assignmentParent()
			input := domain.UserRoleCommand{UserID: 23, RoleIDs: []uint64{10, 11}, Reason: "approved\r\nreason", Version: 9007199254740993, IdempotencyKey: "opaque, key"}
			calls := 0
			result := &userv1.GrantUserRolesResponse{UserId: 23, Version: 9007199254740993}
			var failure error
			inspect := func(c context.Context, id uint64, ids []uint64, reason string, version *uint64) {
				calls++
				checkAssignmentMetadata(t, c, md, input.IdempotencyKey)
				if id != 23 || !reflect.DeepEqual(ids, input.RoleIDs) || reason != input.Reason || version == nil || *version != input.Version {
					t.Fatal(id, ids, reason, version)
				}
				ids[0] = 99
				*version = 1
			}
			sdk := &assignmentClient{grant: func(c context.Context, r *userv1.GrantUserRolesRequest) (*userv1.GrantUserRolesResponse, error) {
				if method != "grant" {
					t.Fatal("wrong method")
				}
				inspect(c, r.UserId, r.RoleIds, r.Reason, r.ExpectedUserVersion)
				return result, failure
			}, revoke: func(c context.Context, r *userv1.RevokeUserRolesRequest) (*userv1.RevokeUserRolesResponse, error) {
				if method != "revoke" {
					t.Fatal("wrong method")
				}
				inspect(c, r.UserId, r.RoleIds, r.Reason, r.ExpectedUserVersion)
				if result == nil {
					return nil, failure
				}
				return &userv1.RevokeUserRolesResponse{UserId: result.UserId, Version: result.Version, ChangedRoleIds: result.ChangedRoleIds, Changed: result.Changed}, failure
			}}
			g, _ := gateway.NewUserRoleManagement(sdk)
			call := g.GrantUserRoles
			if method == "revoke" {
				call = g.RevokeUserRoles
			}
			if _, e := call(nil, input); e == nil {
				t.Fatal("nil context")
			}
			bad := input
			bad.RoleIDs = nil
			if _, e := call(ctx, bad); e == nil || calls != 0 {
				t.Fatal("guard dispatch")
			}
			for _, ids := range [][]uint64{nil, {}, {10}, {10, 11}} {
				result = &userv1.GrantUserRolesResponse{UserId: 23, Version: 9223372036854775807, ChangedRoleIds: ids, Changed: len(ids) > 0}
				for repeat := 0; repeat < 2; repeat++ {
					before := calls
					out, e := call(ctx, input)
					if e != nil || calls != before+1 || out.Version != "9223372036854775807" || out.Changed != (len(ids) > 0) || len(out.ChangedRoleIDs) != len(ids) || out.ChangedRoleIDs == nil {
						t.Fatal(out, e)
					}
					if len(out.ChangedRoleIDs) > 0 {
						out.ChangedRoleIDs[0] = 99
						if ids[0] != 10 {
							t.Fatal("result alias")
						}
					}
				}
			}
			for _, mutate := range []func(*userv1.GrantUserRolesResponse){func(r *userv1.GrantUserRolesResponse) { r.UserId = 0 }, func(r *userv1.GrantUserRolesResponse) { r.UserId = 99 }, func(r *userv1.GrantUserRolesResponse) { r.Version = 0 }, func(r *userv1.GrantUserRolesResponse) { r.Version = 1 << 63 }, func(r *userv1.GrantUserRolesResponse) { r.ChangedRoleIds = []uint64{10} }, func(r *userv1.GrantUserRolesResponse) { r.Changed = true }, func(r *userv1.GrantUserRolesResponse) { r.Changed = true; r.ChangedRoleIds = []uint64{0} }, func(r *userv1.GrantUserRolesResponse) { r.Changed = true; r.ChangedRoleIds = []uint64{10, 10} }, func(r *userv1.GrantUserRolesResponse) { r.Changed = true; r.ChangedRoleIds = []uint64{11, 10} }, func(r *userv1.GrantUserRolesResponse) { r.Changed = true; r.ChangedRoleIds = []uint64{12} }, func(r *userv1.GrantUserRolesResponse) { r.Changed = true; r.ChangedRoleIds = []uint64{1 << 31} }} {
				result = &userv1.GrantUserRolesResponse{UserId: 23, Version: 1}
				mutate(result)
				out, e := call(ctx, input)
				if !errors.Is(e, common.ErrInternalServerError) || out.UserID != 0 {
					t.Fatal(out, e)
				}
			}
			result = nil
			if _, e := call(ctx, input); !errors.Is(e, common.ErrInternalServerError) {
				t.Fatal(e)
			}
			for _, f := range []error{common.ErrForbidden, common.ErrNotFound, common.ErrInternalServerError, context.Canceled, context.DeadlineExceeded} {
				failure = f
				before := calls
				out, e := call(ctx, input)
				if e != f || calls != before+1 || out.UserID != 0 {
					t.Fatal(out, e)
				}
			}
			if input.RoleIDs[0] != 10 || input.Version != 9007199254740993 || md.Get("x-request-id")[0] != "assignment" {
				t.Fatal("input mutated")
			}
		})
	}
}
