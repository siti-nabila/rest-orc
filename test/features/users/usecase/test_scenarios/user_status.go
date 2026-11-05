package test_scenarios

import (
	"context"
	"errors"
	"testing"

	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/internal/features/users/usecase"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

type statusGateway struct {
	calls   int
	ctx     context.Context
	request domain.ChangeUserStatusRequest
	result  domain.ChangeUserStatusResponse
	err     error
}

func (g *statusGateway) ChangeUserStatus(c context.Context, r domain.ChangeUserStatusRequest) (domain.ChangeUserStatusResponse, error) {
	g.calls++
	g.ctx = c
	g.request = r
	return g.result, g.err
}
func UserStatusScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "status exact authority guards one call normalized request no-op and errors", Run: statusUsecase}}
}
func statusUsecase(t *testing.T) {
	if _, err := usecase.NewChangeUserStatus(nil); err == nil {
		t.Fatal("nil gateway")
	}
	allowed := domain.UserStatusActor{UserID: 17, Permissions: map[string]bool{domain.UserStatusUpdatePermission: true}}
	for _, tc := range []struct {
		actor domain.UserStatusActor
		ctx   context.Context
		want  error
	}{
		{allowed, nil, common.ErrUnauthorized}, {domain.UserStatusActor{}, context.Background(), common.ErrUnauthorized},
		{domain.UserStatusActor{UserID: 17}, context.Background(), common.ErrForbidden},
		{domain.UserStatusActor{UserID: 17, Permissions: map[string]bool{domain.UserStatusUpdatePermission: false}}, context.Background(), common.ErrForbidden},
		{domain.UserStatusActor{UserID: 17, Permissions: map[string]bool{"iam.user.*": true, "iam.user": true, "iam.user.read": true, "iam.session.revoke": true}}, context.Background(), common.ErrForbidden},
		{domain.UserStatusActor{UserID: 17, MustChangePassword: true, Permissions: allowed.Permissions}, context.Background(), common.ErrForbidden},
	} {
		g := &statusGateway{}
		uc, _ := usecase.NewChangeUserStatus(g)
		got, err := uc.Execute(tc.ctx, tc.actor, domain.ChangeUserStatusRequest{})
		if !errors.Is(err, tc.want) || g.calls != 0 || got.UserID != 0 {
			t.Fatal(got, err, g.calls)
		}
	}
	for _, target := range []uint64{17, 23} {
		for _, state := range []string{"ACTIVE", "PENDING", "SUSPENDED", "DEACTIVATED"} {
			reason := "  approved  "
			r := domain.ChangeUserStatusRequest{UserID: target, Status: &state, Reason: &reason, Version: 9007199254740993, IdempotencyKey: "key"}
			g := &statusGateway{result: domain.ChangeUserStatusResponse{UserID: target, Status: state, Version: "1", Changed: false}}
			uc, _ := usecase.NewChangeUserStatus(g)
			ctx := context.Background()
			got, err := uc.Execute(ctx, allowed, r)
			if err != nil || g.calls != 1 || g.ctx != ctx || g.request.UserID != target || g.request.Version != r.Version || *g.request.Reason != "approved" || got.Version != "1" || got.Changed {
				t.Fatal(got, err, g)
			}
			*g.request.Status = "altered"
			*g.request.Reason = "altered"
			if state == "altered" || reason != "  approved  " {
				t.Fatal("aliased")
			}
			failure := errors.New("dependency")
			g.err = failure
			if got, err = uc.Execute(ctx, allowed, r); err != failure || got.UserID != 0 {
				t.Fatal(got, err)
			}
			g.err = nil
			g.result.UserID = 99
			if got, err = uc.Execute(ctx, allowed, r); !errors.Is(err, common.ErrInternalServerError) || got.UserID != 0 {
				t.Fatal(got, err)
			}
			before := g.calls
			r.UserID = 0
			if _, err = uc.Execute(ctx, allowed, r); err == nil || g.calls != before {
				t.Fatal("invalid dispatch")
			}
		}
	}
}
