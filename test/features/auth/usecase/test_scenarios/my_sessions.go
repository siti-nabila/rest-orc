package test_scenarios

import (
	"context"
	"errors"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	uc "github.com/siti-nabila/rest-orc/internal/features/auth/usecase"
	"github.com/siti-nabila/rest-orc/test/features/auth/usecase/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"testing"
)

func MySessionsScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "self-session guards and one-call authority", Run: mySessionsUseCases}}
}
func mySessionsUseCases(t *testing.T) {
	if _, e := uc.NewListMySessions(nil); e == nil {
		t.Fatal("nil list gateway")
	}
	if _, e := uc.NewRevokeMySession(nil); e == nil {
		t.Fatal("nil revoke gateway")
	}
	if _, e := uc.NewRevokeAllMySessions(nil); e == nil {
		t.Fatal("nil all gateway")
	}
	ctx := context.Background()
	q := domain.MySessionsQuery{Page: 1, Limit: 20}
	actor := domain.SessionActor{UserID: 17}
	id := "00000000-0000-0000-0000-000000000000"
	calls := 0
	failure := errors.New("dependency")
	var upstreamError error
	owner := uint64(17)
	stamp := "past"
	source := []domain.MySession{{UserID: owner, LastSeenAt: &stamp, RevokedAt: &stamp}}
	g := &mocks.MySessionsGateway{
		List: func(c context.Context, in domain.MySessionsQuery) (domain.MySessionsPage, error) {
			calls++
			if c != ctx || in != q {
				t.Fatal("context/query changed")
			}
			source[0].UserID = owner
			return domain.MySessionsPage{Items: source, Total: 2147483648}, upstreamError
		},
		Revoke: func(c context.Context, in string) (domain.RevokeMySessionResponse, error) {
			calls++
			if c != ctx || in != id {
				t.Fatal("target changed")
			}
			return domain.RevokeMySessionResponse{}, upstreamError
		},
		All: func(c context.Context) (domain.RevokeAllMySessionsResponse, error) {
			calls++
			if c != ctx {
				t.Fatal("context changed")
			}
			return domain.RevokeAllMySessionsResponse{}, upstreamError
		},
	}
	list, _ := uc.NewListMySessions(g)
	one, _ := uc.NewRevokeMySession(g)
	all, _ := uc.NewRevokeAllMySessions(g)
	for _, tc := range []struct {
		ctx   context.Context
		actor domain.SessionActor
		want  error
	}{{nil, actor, common.ErrUnauthorized}, {ctx, domain.SessionActor{}, common.ErrUnauthorized}, {ctx, domain.SessionActor{UserID: 17, MustChangePassword: true}, common.ErrForbidden}} {
		_, a := list.Execute(tc.ctx, tc.actor, q)
		_, b := one.Execute(tc.ctx, tc.actor, id)
		_, c := all.Execute(tc.ctx, tc.actor)
		if !errors.Is(a, tc.want) || !errors.Is(b, tc.want) || !errors.Is(c, tc.want) || calls != 0 {
			t.Fatal("guard dispatched")
		}
	}
	if _, e := list.Execute(ctx, actor, domain.MySessionsQuery{}); e == nil {
		t.Fatal("invalid query")
	}
	if _, e := one.Execute(ctx, actor, "bad"); e == nil || calls != 0 {
		t.Fatal("invalid target")
	}
	result, e := list.Execute(ctx, actor, q)
	if e != nil || result.Total != 2147483648 || calls != 1 {
		t.Fatal("list dispatch")
	}
	result.Items[0].UserID = 99
	*result.Items[0].LastSeenAt = "changed"
	*result.Items[0].RevokedAt = "changed"
	if source[0].UserID != 17 || stamp != "past" {
		t.Fatal("aliased collection")
	}
	owner = 99
	if r, e := list.Execute(ctx, actor, q); e == nil || r.Items != nil {
		t.Fatal("owner mismatch leaked")
	}
	owner = 17
	for _, err := range []error{nil, failure} {
		upstreamError = err
		before := calls
		_, a := list.Execute(ctx, actor, q)
		r, b := one.Execute(ctx, actor, id)
		s, c := all.Execute(ctx, actor)
		if a != err || b != err || c != err || calls != before+3 || r.Changed || s.Changed || s.LoginRequired {
			t.Fatal("result/error/calls changed")
		}
	}
}
