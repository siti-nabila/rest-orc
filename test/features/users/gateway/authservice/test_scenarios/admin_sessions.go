package test_scenarios

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"testing"

	sessionv1 "github.com/siti-nabila/api-contracts/pb/session/v1"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	gateway "github.com/siti-nabila/rest-orc/internal/features/users/gateway/authservice"
	"github.com/siti-nabila/rest-orc/test/features/users/gateway/authservice/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func AdminSessionScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "session gateway maps safe historical data and validates dependency results", Run: adminSessionsGateway}}
}

func adminSessionsGateway(t *testing.T) {
	if _, e := gateway.NewAdminSessions(nil); e == nil {
		t.Fatal("nil client")
	}
	parentMD := metadata.Pairs("authorization", "Bearer actor", "idempotency-key", "inherited", "version", "88", "x-request-id", "request")
	ctx := metadata.NewOutgoingContext(context.Background(), parentMD)
	single := domain.RevokeUserSessionRequest{UserID: 23, SessionID: fixtures.MySessionID, IdempotencyKey: "opaque, key"}
	all := domain.RevokeAllUserSessionsRequest{UserID: 23, IdempotencyKey: single.IdempotencyKey, Version: 9223372036854775807}
	check := func(c context.Context, key string) {
		t.Helper()
		md, _ := metadata.FromOutgoingContext(c)
		want := parentMD.Copy()
		want.Delete("idempotency-key")
		want.Delete("version")
		if key != "" {
			want.Set("idempotency-key", key)
		}
		if !reflect.DeepEqual(md, want) {
			t.Fatal(md, want)
		}
		if !reflect.DeepEqual(parentMD.Get("idempotency-key"), []string{"inherited"}) {
			t.Fatal("parent mutated")
		}
	}
	q := domain.AdminSessionsQuery{Page: 1, Limit: 20, LastID: "%2F+ "}
	calls := 0
	result := fixtures.MySessionsResponse(23)
	var failure error
	nilResult := false
	changed := false
	version := uint64(9007199254740993)
	sdk := &mocks.AdminSessionClient{List: func(c context.Context, r *sessionv1.ListUserSessionsRequest) (*sessionv1.ListSessionsResponse, error) {
		calls++
		check(c, "")
		if r.UserId != 23 || r.Query.Page != 1 || r.Query.Limit != 20 || r.Query.LastId != q.LastID || r.Query.Fields != nil || r.Query.Search != nil || len(r.Query.Sort) != 2 || r.Query.Sort[0].Field != "created_at" || !r.Query.Sort[0].Desc || r.Query.Sort[1].Field != "id" || !r.Query.Sort[1].Desc {
			t.Fatal("query/context changed")
		}
		return result, failure
	}, Revoke: func(c context.Context, r *sessionv1.RevokeUserSessionRequest) (*sessionv1.RevokeSessionResponse, error) {
		calls++
		check(c, single.IdempotencyKey)
		if r.UserId != 23 || r.SessionId != fixtures.MySessionID {
			t.Fatal("target/context changed")
		}
		if failure != nil || nilResult {
			return nil, failure
		}
		return &sessionv1.RevokeSessionResponse{Changed: changed}, nil
	}, All: func(c context.Context, r *sessionv1.RevokeAllUserSessionsRequest) (*sessionv1.RevokeAllUserSessionsResponse, error) {
		calls++
		check(c, all.IdempotencyKey)
		if r.UserId != 23 || r.ExpectedUserVersion == nil || *r.ExpectedUserVersion != all.Version {
			t.Fatal("context changed")
		}
		if failure != nil || nilResult {
			return nil, failure
		}
		return &sessionv1.RevokeAllUserSessionsResponse{Changed: changed, Version: version}, nil
	}}
	g, _ := gateway.NewAdminSessions(sdk)
	for _, invoke := range []func() error{func() error { _, e := g.ListUserSessions(nil, 23, q); return e }, func() error { _, e := g.ListUserSessions(ctx, 23, domain.AdminSessionsQuery{}); return e }, func() error { _, e := g.RevokeUserSession(nil, single); return e }, func() error { _, e := g.RevokeUserSession(ctx, domain.RevokeUserSessionRequest{}); return e }, func() error { _, e := g.RevokeAllUserSessions(nil, all); return e }} {
		if invoke() == nil {
			t.Fatal("guard failed")
		}
	}
	if calls != 0 {
		t.Fatal("guard dispatched")
	}
	result.Items[0].LastSeenAt = result.Items[0].CreatedAt
	result.Items[0].RevokedAt = result.Items[0].CreatedAt
	got, e := g.ListUserSessions(ctx, 23, q)
	if e != nil || got.Total != 2147483648 || !got.HasPrev || got.NextCursor != "opaque-next" || got.Items[0].CreatedAt != "2020-01-01T00:00:00.123456789Z" || got.Items[0].IsCurrent {
		t.Fatal("mapping", e)
	}
	got.Items[0].ID = "changed"
	*got.Items[0].LastSeenAt = "changed"
	if result.Items[0].Id != fixtures.MySessionID || result.Items[0].LastSeenAt.Nanos != 123456789 {
		t.Fatal("aliased mapping")
	}
	result.Items = nil
	got, e = g.ListUserSessions(ctx, 23, q)
	if e != nil || got.Items == nil || got.Total != 2147483648 {
		t.Fatal("empty page total")
	}
	for _, mutate := range []func(){func() { result = nil }, func() { result.Meta = nil }, func() { result.Meta.Page = 0 }, func() { result.Meta.Limit = 101 }, func() { result.Meta.Total = -1 }, func() { result.Meta.TotalPages = -1 }, func() { result.Items[0] = nil }, func() { result.Items[0].Id = "bad" }, func() { result.Items[0].UserId = 0 }, func() { result.Items[0].UserId = 2147483648 }, func() { result.Items[0].Status = "EXPIRED" }, func() { result.Items[0].CreatedAt = nil }, func() { result.Items[0].ExpiresAt = nil }, func() { result.Items[0].LastSeenAt = &timestamppb.Timestamp{Seconds: 1 << 62} }, func() { result.Items[0].RevokedAt = &timestamppb.Timestamp{Nanos: -1} }} {
		result = fixtures.MySessionsResponse(23)
		mutate()
		if r, e := g.ListUserSessions(ctx, 23, q); e == nil || r.Items != nil {
			t.Fatal("invalid result accepted")
		}
	}
	for _, ch := range []bool{false, true} {
		for _, v := range []uint64{1, 9007199254740993, 9223372036854775807} {
			changed, version = ch, v
			r, e := g.RevokeUserSession(ctx, single)
			s, f := g.RevokeAllUserSessions(ctx, all)
			if e != nil || f != nil || r.Changed != ch || s.Changed != ch || s.Version != strconv.FormatUint(v, 10) {
				t.Fatal("flags changed")
			}
		}
	}
	for _, v := range []uint64{0, 1 << 63} {
		version = v
		if _, e := g.RevokeAllUserSessions(ctx, all); e == nil {
			t.Fatal("invalid version")
		}
	}
	version = 1
	nilResult = true
	if _, e := g.RevokeUserSession(ctx, single); e == nil {
		t.Fatal("nil revoke")
	}
	if _, e := g.RevokeAllUserSessions(ctx, all); e == nil {
		t.Fatal("nil all")
	}
	failure = errors.New("dependency")
	_, a := g.ListUserSessions(ctx, 23, q)
	_, b := g.RevokeUserSession(ctx, single)
	_, c := g.RevokeAllUserSessions(ctx, all)
	if a != failure || b != failure || c != failure {
		t.Fatal("error identity")
	}
}
