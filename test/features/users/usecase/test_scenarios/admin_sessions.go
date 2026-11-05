package test_scenarios

import (
	"context"
	"errors"
	"testing"

	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/internal/features/users/usecase"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

type adminGateway struct {
	calls  int
	target uint64
	ctx    context.Context
	result domain.AdminSessionsPage
	err    error
	single domain.RevokeUserSessionRequest
	all    domain.RevokeAllUserSessionsRequest
}

func (g *adminGateway) ListUserSessions(ctx context.Context, id uint64, q domain.AdminSessionsQuery) (domain.AdminSessionsPage, error) {
	g.calls++
	g.target = id
	g.ctx = ctx
	return g.result, g.err
}

func (g *adminGateway) RevokeUserSession(ctx context.Context, r domain.RevokeUserSessionRequest) (domain.RevokeUserSessionResponse, error) {
	g.calls++
	g.single = r
	g.ctx = ctx
	return domain.RevokeUserSessionResponse{}, g.err
}

func (g *adminGateway) RevokeAllUserSessions(ctx context.Context, r domain.RevokeAllUserSessionsRequest) (domain.RevokeAllUserSessionsResponse, error) {
	g.calls++
	g.all = r
	g.ctx = ctx
	return domain.RevokeAllUserSessionsResponse{Version: "9007199254740993"}, g.err
}

func AdminSessionScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "permission guards and request result isolation", Run: adminUsecases}}
}

func adminUsecases(t *testing.T) {
	if _, err := usecase.NewListUserSessions(nil); err == nil {
		t.Fatal("nil list")
	}
	if _, err := usecase.NewRevokeUserSession(nil); err == nil {
		t.Fatal("nil revoke")
	}
	if _, err := usecase.NewRevokeAllUserSessions(nil); err == nil {
		t.Fatal("nil all")
	}
	for _, a := range []domain.AdminSessionActor{{}, {UserID: 17}, {UserID: 17, Permissions: map[string]bool{"iam.session.*": true}}, {UserID: 17, Permissions: map[string]bool{domain.SessionReadPermission: false}}, {UserID: 17, MustChangePassword: true, Permissions: map[string]bool{domain.SessionReadPermission: true, domain.SessionRevokePermission: true}}} {
		g := &adminGateway{}
		l, _ := usecase.NewListUserSessions(g)
		s, _ := usecase.NewRevokeUserSession(g)
		all, _ := usecase.NewRevokeAllUserSessions(g)
		_, err := l.Execute(context.Background(), a, 0, domain.AdminSessionsQuery{})
		want := common.ErrForbidden
		if a.UserID == 0 {
			want = common.ErrUnauthorized
		}
		if !errors.Is(err, want) {
			t.Fatal(err)
		}
		if _, err = s.Execute(context.Background(), a, domain.RevokeUserSessionRequest{}); !errors.Is(err, want) {
			t.Fatal(err)
		}
		if _, err = all.Execute(context.Background(), a, domain.RevokeAllUserSessionsRequest{}); !errors.Is(err, want) {
			t.Fatal(err)
		}
		if g.calls != 0 {
			t.Fatal(g.calls)
		}
	}
	value := "past"
	g := &adminGateway{result: domain.AdminSessionsPage{Items: []domain.AdminSession{{UserID: 23, LastSeenAt: &value, RevokedAt: &value}}, Total: 2147483648}}
	l, _ := usecase.NewListUserSessions(g)
	s, _ := usecase.NewRevokeUserSession(g)
	all, _ := usecase.NewRevokeAllUserSessions(g)
	actor := domain.AdminSessionActor{UserID: 17, Permissions: map[string]bool{domain.SessionReadPermission: true, domain.SessionRevokePermission: true}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if _, err := l.Execute(nil, actor, 23, domain.AdminSessionsQuery{Page: 1, Limit: 20}); !errors.Is(err, common.ErrUnauthorized) {
		t.Fatal(err)
	}
	result, err := l.Execute(ctx, actor, 23, domain.AdminSessionsQuery{Page: 1, Limit: 20})
	if err != nil || g.target != 23 || g.ctx != ctx || g.calls != 1 || result.Total != 2147483648 {
		t.Fatal(result, err)
	}
	*result.Items[0].LastSeenAt = "changed"
	*result.Items[0].RevokedAt = "changed"
	result.Items[0].UserID = 99
	if value != "past" || g.result.Items[0].UserID != 23 {
		t.Fatal("aliased")
	}
	g.result.Items = append(g.result.Items, domain.AdminSession{UserID: 17})
	if got, err := l.Execute(ctx, actor, 23, domain.AdminSessionsQuery{Page: 1, Limit: 20}); !errors.Is(err, dictionary.ErrSessionResultInvalid) || len(got.Items) != 0 {
		t.Fatal(got, err)
	}
	single := domain.RevokeUserSessionRequest{UserID: 23, SessionID: "00000000-0000-0000-0000-000000000000", IdempotencyKey: "a, b"}
	if got, err := s.Execute(ctx, actor, single); err != nil || got.Changed || g.single != single {
		t.Fatal(got, err)
	}
	request := domain.RevokeAllUserSessionsRequest{UserID: 23, IdempotencyKey: "a, b", Version: 9223372036854775807}
	if got, err := all.Execute(ctx, actor, request); err != nil || got.Version != "9007199254740993" || g.all != request {
		t.Fatal(got, err)
	}
	sentinel := errors.New("downstream")
	g.err = sentinel
	if _, err := l.Execute(ctx, actor, 23, domain.AdminSessionsQuery{Page: 1, Limit: 20}); err != sentinel {
		t.Fatal(err)
	}
	if _, err := s.Execute(ctx, actor, single); err != sentinel {
		t.Fatal(err)
	}
	if _, err := all.Execute(ctx, actor, request); err != sentinel {
		t.Fatal(err)
	}
	before := g.calls
	if _, err := l.Execute(ctx, actor, 0, domain.AdminSessionsQuery{Page: 1, Limit: 20}); err == nil {
		t.Fatal("target")
	}
	if _, err := l.Execute(ctx, actor, 23, domain.AdminSessionsQuery{}); err == nil {
		t.Fatal("query")
	}
	if _, err := s.Execute(ctx, actor, domain.RevokeUserSessionRequest{}); err == nil {
		t.Fatal("single")
	}
	if _, err := all.Execute(ctx, actor, domain.RevokeAllUserSessionsRequest{}); err == nil {
		t.Fatal("all")
	}
	if g.calls != before {
		t.Fatal("dispatch invalid")
	}
}
