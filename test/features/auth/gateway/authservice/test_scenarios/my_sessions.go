package test_scenarios

import (
	"context"
	"errors"
	sessionv1 "github.com/siti-nabila/api-contracts/pb/session/v1"
	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	gateway "github.com/siti-nabila/rest-orc/internal/features/auth/gateway/authservice"
	"github.com/siti-nabila/rest-orc/test/features/auth/gateway/authservice/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/protobuf/types/known/timestamppb"
	"testing"
)

func MySessionsScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "session gateway maps safe historical data and validates dependency results", Run: mySessionsGateway}}
}
func mySessionsGateway(t *testing.T) {
	if _, e := gateway.NewMySessions(nil); e == nil {
		t.Fatal("nil client")
	}
	ctx := context.Background()
	q := domain.MySessionsQuery{Page: 1, Limit: 20, LastID: "%2F+ "}
	calls := 0
	result := fixtures.MySessionsResponse(17)
	var failure error
	nilResult := false
	changed, login := false, false
	sdk := &mocks.SelfSessionClient{List: func(c context.Context, r *sessionv1.ListMySessionsRequest) (*sessionv1.ListSessionsResponse, error) {
		calls++
		if c != ctx || r.Query.Page != 1 || r.Query.Limit != 20 || r.Query.LastId != q.LastID || r.Query.Fields != nil || r.Query.Search != nil || len(r.Query.Sort) != 2 || r.Query.Sort[0].Field != "created_at" || !r.Query.Sort[0].Desc || r.Query.Sort[1].Field != "id" || !r.Query.Sort[1].Desc {
			t.Fatal("query/context changed")
		}
		return result, failure
	}, Revoke: func(c context.Context, r *sessionv1.RevokeMySessionRequest) (*sessionv1.RevokeSessionResponse, error) {
		calls++
		if c != ctx || r.SessionId != fixtures.MySessionID {
			t.Fatal("target/context changed")
		}
		if failure != nil || nilResult {
			return nil, failure
		}
		return &sessionv1.RevokeSessionResponse{Changed: changed}, nil
	}, All: func(c context.Context) (*sessionv1.RevokeAllMySessionsResponse, error) {
		calls++
		if c != ctx {
			t.Fatal("context changed")
		}
		if failure != nil || nilResult {
			return nil, failure
		}
		return &sessionv1.RevokeAllMySessionsResponse{Changed: changed, LoginRequired: login}, nil
	}}
	g, _ := gateway.NewMySessions(sdk)
	for _, invoke := range []func() error{func() error { _, e := g.ListMySessions(nil, q); return e }, func() error { _, e := g.ListMySessions(ctx, domain.MySessionsQuery{}); return e }, func() error { _, e := g.RevokeMySession(nil, fixtures.MySessionID); return e }, func() error { _, e := g.RevokeMySession(ctx, "bad"); return e }, func() error { _, e := g.RevokeAllMySessions(nil); return e }} {
		if invoke() == nil {
			t.Fatal("guard failed")
		}
	}
	if calls != 0 {
		t.Fatal("guard dispatched")
	}
	result.Items[0].LastSeenAt = result.Items[0].CreatedAt
	result.Items[0].RevokedAt = result.Items[0].CreatedAt
	got, e := g.ListMySessions(ctx, q)
	if e != nil || got.Total != 2147483648 || !got.HasPrev || got.NextCursor != "opaque-next" || got.Items[0].CreatedAt != "2020-01-01T00:00:00.123456789Z" || got.Items[0].IsCurrent {
		t.Fatal("mapping", e)
	}
	got.Items[0].ID = "changed"
	*got.Items[0].LastSeenAt = "changed"
	if result.Items[0].Id != fixtures.MySessionID || result.Items[0].LastSeenAt.Nanos != 123456789 {
		t.Fatal("aliased mapping")
	}
	result.Items = nil
	got, e = g.ListMySessions(ctx, q)
	if e != nil || got.Items == nil || got.Total != 2147483648 {
		t.Fatal("empty page total")
	}
	for _, mutate := range []func(){func() { result = nil }, func() { result.Meta = nil }, func() { result.Meta.Page = 0 }, func() { result.Meta.Limit = 101 }, func() { result.Meta.Total = -1 }, func() { result.Meta.TotalPages = -1 }, func() { result.Items[0] = nil }, func() { result.Items[0].Id = "bad" }, func() { result.Items[0].UserId = 0 }, func() { result.Items[0].Status = "EXPIRED" }, func() { result.Items[0].CreatedAt = nil }, func() { result.Items[0].ExpiresAt = nil }, func() { result.Items[0].LastSeenAt = &timestamppb.Timestamp{Seconds: 1 << 62} }, func() { result.Items[0].RevokedAt = &timestamppb.Timestamp{Nanos: -1} }} {
		result = fixtures.MySessionsResponse(17)
		mutate()
		if r, e := g.ListMySessions(ctx, q); e == nil || r.Items != nil {
			t.Fatal("invalid result accepted")
		}
	}
	for _, ch := range []bool{false, true} {
		for _, lr := range []bool{false, true} {
			changed, login = ch, lr
			r, e := g.RevokeMySession(ctx, fixtures.MySessionID)
			s, f := g.RevokeAllMySessions(ctx)
			if e != nil || f != nil || r.Changed != ch || s.Changed != ch || s.LoginRequired != lr {
				t.Fatal("flags changed")
			}
		}
	}
	nilResult = true
	if _, e := g.RevokeMySession(ctx, fixtures.MySessionID); e == nil {
		t.Fatal("nil revoke")
	}
	if _, e := g.RevokeAllMySessions(ctx); e == nil {
		t.Fatal("nil all")
	}
	failure = errors.New("dependency")
	_, a := g.ListMySessions(ctx, q)
	_, b := g.RevokeMySession(ctx, fixtures.MySessionID)
	_, c := g.RevokeAllMySessions(ctx)
	if a != failure || b != failure || c != failure {
		t.Fatal("error identity")
	}
}
