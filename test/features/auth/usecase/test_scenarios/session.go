package test_scenarios

import (
	"context"
	"errors"

	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	"github.com/siti-nabila/rest-orc/internal/features/auth/usecase"
	"github.com/siti-nabila/rest-orc/test/features/auth/usecase/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"testing"
)

func SessionScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "session usecases preserve proof context and errors without fallback", Run: sessionUsecases}}
}

func sessionUsecases(t *testing.T) {
	if _, err := usecase.NewRefresh(nil); err == nil {
		t.Fatal("nil refresh gateway accepted")
	}
	if _, err := usecase.NewLogout(nil); err == nil {
		t.Fatal("nil logout gateway accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	sentinel := errors.New("dependency sentinel")
	g := &mocks.SessionGateway{
		RefreshFunc: func(got context.Context, token string) (domain.SessionResult, error) {
			calls++
			if got != ctx || token != "opaque raw" {
				t.Fatal("refresh changed input/context")
			}
			return domain.SessionResult{Token: "result"}, sentinel
		},
		LogoutFunc: func(got context.Context, proof domain.LogoutCredential) (domain.LogoutResponse, error) {
			calls++
			if got != ctx || !proof.Valid() {
				t.Fatal("logout changed proof/context")
			}
			return domain.LogoutResponse{Changed: false}, sentinel
		},
	}
	r, _ := usecase.NewRefresh(g)
	l, _ := usecase.NewLogout(g)
	for _, input := range []struct {
		ctx   context.Context
		token string
	}{{nil, "raw"}, {ctx, ""}} {
		if _, err := r.Execute(input.ctx, input.token); err == nil {
			t.Fatal("invalid refresh accepted")
		}
	}
	for _, proof := range []domain.LogoutCredential{{}, {Authorization: "bad"}, {Authorization: "Bearer access", RefreshToken: "raw"}} {
		if _, err := l.Execute(ctx, proof); err == nil {
			t.Fatal("invalid logout accepted")
		}
	}
	if _, err := l.Execute(nil, domain.LogoutCredential{RefreshToken: "raw"}); err == nil {
		t.Fatal("nil context accepted")
	}
	if calls != 0 {
		t.Fatal("invalid input dispatched")
	}
	res, err := r.Execute(ctx, "opaque raw")
	if !errors.Is(err, sentinel) || res.Token != "result" {
		t.Fatal("refresh result/error changed")
	}
	for _, proof := range []domain.LogoutCredential{{Authorization: "Bearer access"}, {RefreshToken: "opaque raw"}} {
		res, err := l.Execute(ctx, proof)
		if !errors.Is(err, sentinel) || res.Changed {
			t.Fatal("logout result/error changed")
		}
	}
	if calls != 3 {
		t.Fatal("unexpected calls/retry")
	}
}
