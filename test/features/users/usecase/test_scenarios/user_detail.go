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

type detailGateway struct {
	calls  int
	ctx    context.Context
	target uint64
	result domain.UserDetail
	err    error
}

func (g *detailGateway) GetUser(ctx context.Context, id uint64) (domain.UserDetail, error) {
	g.calls++
	g.ctx = ctx
	g.target = id
	return g.result, g.err
}
func UserDetailScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "detail actor permission guards target equality and isolated outputs", Run: detailUsecase}}
}
func detailUsecase(t *testing.T) {
	if _, err := usecase.NewGetUser(nil); err == nil {
		t.Fatal("nil gateway")
	}
	allowed := domain.UserDetailActor{UserID: 17, Permissions: map[string]bool{domain.UserReadPermission: true}}
	for _, tc := range []struct {
		actor domain.UserDetailActor
		ctx   context.Context
		want  error
	}{
		{allowed, nil, common.ErrUnauthorized}, {domain.UserDetailActor{}, context.Background(), common.ErrUnauthorized},
		{domain.UserDetailActor{UserID: 17}, context.Background(), common.ErrForbidden},
		{domain.UserDetailActor{UserID: 17, Permissions: map[string]bool{domain.UserReadPermission: false}}, context.Background(), common.ErrForbidden},
		{domain.UserDetailActor{UserID: 17, Permissions: map[string]bool{"iam.user.*": true, "iam.user": true, "iam.session.read": true}}, context.Background(), common.ErrForbidden},
		{domain.UserDetailActor{UserID: 17, MustChangePassword: true, Permissions: allowed.Permissions}, context.Background(), common.ErrForbidden},
	} {
		g := &detailGateway{}
		uc, _ := usecase.NewGetUser(g)
		got, err := uc.Execute(tc.ctx, tc.actor, 0)
		if !errors.Is(err, tc.want) || g.calls != 0 || got.ID != 0 {
			t.Fatal(got, err, g.calls)
		}
	}
	for _, target := range []uint64{17, 23} {
		g := &detailGateway{result: domain.UserDetail{ID: target, Version: "9007199254740993", Roles: []domain.UserDetailRole{{RoleID: 9, Name: "role"}}, ApplicationAccess: []domain.UserDetailApplicationAccess{{ApplicationID: 5, Name: "app"}}}}
		uc, _ := usecase.NewGetUser(g)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		got, err := uc.Execute(ctx, allowed, target)
		if err != nil || g.ctx != ctx || g.target != target || g.calls != 1 || got.Version != "9007199254740993" {
			t.Fatal(got, err)
		}
		got.Roles[0].Name = "changed"
		got.ApplicationAccess[0].Name = "changed"
		if g.result.Roles[0].Name != "role" || g.result.ApplicationAccess[0].Name != "app" {
			t.Fatal("aliased")
		}
		g.result.ID = 99
		if got, err = uc.Execute(ctx, allowed, target); !errors.Is(err, common.ErrInternalServerError) || got.ID != 0 || len(got.Roles) != 0 {
			t.Fatal(got, err)
		}
		failure := errors.New("dependency")
		g.err = failure
		if _, err = uc.Execute(ctx, allowed, target); err != failure {
			t.Fatal(err)
		}
		before := g.calls
		for _, id := range []uint64{0, 2147483648} {
			if _, err = uc.Execute(ctx, allowed, id); err == nil {
				t.Fatal("invalid target")
			}
		}
		if g.calls != before {
			t.Fatal("invalid dispatch")
		}
		g.err = nil
		g.result = domain.UserDetail{ID: target}
		got, err = uc.Execute(ctx, allowed, target)
		if err != nil || got.Roles == nil || got.ApplicationAccess == nil {
			t.Fatal(got, err)
		}
	}
}
