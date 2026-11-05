package test_scenarios

import (
	"context"
	"errors"

	authv1 "github.com/siti-nabila/api-contracts/pb/auth/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	authgateway "github.com/siti-nabila/rest-orc/internal/features/auth/gateway/authservice"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/test/features/auth/gateway/authservice/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
	"reflect"
	"testing"
)

func SessionScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "session gateway selects credential mode and preserves parent metadata", Run: sessionGateway}, {Name: "session gateway guards invalid dependency output", Run: sessionGatewayGuards}}
}
func sessionGateway(t *testing.T) {
	if _, err := authgateway.NewSession(nil); err == nil {
		t.Fatal("nil client accepted")
	}
	md := metadata.Pairs("authorization", "Bearer unrelated", "x-request-id", "request", "locale", "id-ID")
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	calls := 0
	check := func(actual context.Context, want string) {
		calls++
		got, _ := metadata.FromOutgoingContext(actual)
		if got.Get("x-request-id")[0] != "request" || len(got.Get("authorization")) != map[bool]int{true: 1, false: 0}[want != ""] {
			t.Fatal("metadata differs")
		}
		if want != "" && got.Get("authorization")[0] != want {
			t.Fatal("wrong bearer")
		}
	}
	client := &mocks.SessionClient{RefreshFunc: func(c context.Context, r *authv1.RefreshSessionRequest) (*userv1.UserTokenResponse, error) {
		check(c, "")
		if r.RefreshToken != "opaque raw" {
			t.Fatal("raw cookie changed")
		}
		return fixtures.LoginTokenResponse(), nil
	}, LogoutFunc: func(c context.Context, r *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
		want := "Bearer access"
		if r.RefreshToken != nil {
			want = ""
			if *r.RefreshToken != "opaque raw" {
				t.Fatal("refresh changed")
			}
		}
		check(c, want)
		return &authv1.LogoutResponse{}, nil
	}}
	g, _ := authgateway.NewSession(client)
	r, err := g.Refresh(ctx, "opaque raw")
	if err != nil || r != fixtures.LoginResult() {
		t.Fatal("result mapping changed")
	}
	for _, p := range []domain.LogoutCredential{{Authorization: "Bearer access"}, {RefreshToken: "opaque raw"}} {
		res, err := g.Logout(ctx, p)
		if err != nil || res.Changed {
			t.Fatal("false result lost")
		}
	}
	original, _ := metadata.FromOutgoingContext(ctx)
	if !reflect.DeepEqual(md, original) || calls != 3 {
		t.Fatal("parent mutation or retry")
	}
}
func sessionGatewayGuards(t *testing.T) {
	cause := errors.New("dependency cause")
	calls := 0
	var result *userv1.UserTokenResponse
	var failure error
	c := &mocks.SessionClient{RefreshFunc: func(context.Context, *authv1.RefreshSessionRequest) (*userv1.UserTokenResponse, error) {
		calls++
		return result, failure
	}, LogoutFunc: func(context.Context, *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
		calls++
		return nil, failure
	}}
	g, _ := authgateway.NewSession(c)
	ctx := context.Background()
	if _, err := g.Refresh(nil, "raw"); err == nil {
		t.Fatal("nil context accepted")
	}
	if _, err := g.Refresh(ctx, ""); err == nil {
		t.Fatal("empty refresh accepted")
	}
	if _, err := g.Logout(nil, domain.LogoutCredential{RefreshToken: "raw"}); err == nil {
		t.Fatal("nil context accepted")
	}
	if _, err := g.Logout(ctx, domain.LogoutCredential{}); err == nil {
		t.Fatal("invalid logout accepted")
	}
	if calls != 0 {
		t.Fatal("invalid dispatched")
	}
	for _, mutate := range []func(*userv1.UserTokenResponse){func(r *userv1.UserTokenResponse) { r.Token = "" }, func(r *userv1.UserTokenResponse) { r.RefreshToken = "" }, func(r *userv1.UserTokenResponse) { r.SessionId = "" }, func(r *userv1.UserTokenResponse) { r.AccessExpiresAt = nil }, func(r *userv1.UserTokenResponse) { r.SessionExpiresAt = &timestamppb.Timestamp{Seconds: 1 << 62} }, func(r *userv1.UserTokenResponse) {
		r.AccessExpiresAt = timestamppb.New(fixtures.SessionNow().AddDate(1, 0, 0))
	}} {
		result = fixtures.LoginTokenResponse()
		mutate(result)
		if _, err := g.Refresh(ctx, "raw"); !errors.Is(err, dictionary.ErrSessionResultInvalid) {
			t.Fatal("invalid session accepted")
		}
	}
	result = nil
	if _, err := g.Refresh(ctx, "raw"); err == nil {
		t.Fatal("nil accepted")
	}
	if _, err := g.Logout(ctx, domain.LogoutCredential{RefreshToken: "raw"}); err == nil {
		t.Fatal("nil logout accepted")
	}
	failure = cause
	_, err := g.Refresh(ctx, "raw")
	if err != cause {
		t.Fatal("error identity lost")
	}
	_, err = g.Logout(ctx, domain.LogoutCredential{Authorization: "Bearer access"})
	if err != cause {
		t.Fatal("error identity lost")
	}
}
