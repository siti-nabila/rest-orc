package test_scenarios

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/internal/features/users/usecase"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

type profileUpdateGateway struct {
	calls   int
	ctx     context.Context
	request domain.UpdateUserProfileRequest
	result  domain.UpdateUserProfileResponse
	err     error
}

func (g *profileUpdateGateway) UpdateUserProfile(c context.Context, r domain.UpdateUserProfileRequest) (domain.UpdateUserProfileResponse, error) {
	g.calls++
	g.ctx = c
	g.request = r
	return g.result, g.err
}
func UserProfileScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "target profile guards local no-op command and result independence", Run: targetProfileUsecase}}
}
func targetProfileUsecase(t *testing.T) {
	if _, err := usecase.NewUpdateUserProfile(nil); err == nil {
		t.Fatal("nil gateway")
	}
	allowed := domain.UserProfileActor{UserID: 17, Permissions: map[string]bool{domain.UserUpdatePermission: true}}
	for _, tc := range []struct {
		a    domain.UserProfileActor
		c    context.Context
		want error
	}{
		{allowed, nil, common.ErrUnauthorized}, {domain.UserProfileActor{}, context.Background(), common.ErrUnauthorized},
		{domain.UserProfileActor{UserID: 17}, context.Background(), common.ErrForbidden},
		{domain.UserProfileActor{UserID: 17, Permissions: map[string]bool{domain.UserUpdatePermission: false}}, context.Background(), common.ErrForbidden},
		{domain.UserProfileActor{UserID: 17, Permissions: map[string]bool{"iam.user.*": true, "iam.user": true, "iam.user.read": true, "iam.user.status.update": true, "iam.session.revoke": true}}, context.Background(), common.ErrForbidden},
		{domain.UserProfileActor{UserID: 17, MustChangePassword: true, Permissions: allowed.Permissions}, context.Background(), common.ErrForbidden},
	} {
		for _, selected := range []bool{true, false} {
			r := domain.UpdateUserProfileRequest{}
			bad := "!"
			if selected {
				r.Profile.Name = &bad
			}
			g := &profileUpdateGateway{}
			uc, _ := usecase.NewUpdateUserProfile(g)
			got, err := uc.Execute(tc.c, tc.a, r)
			if !errors.Is(err, tc.want) || g.calls != 0 || got.User != nil {
				t.Fatal(got, err, g.calls)
			}
		}
	}
	for _, target := range []uint64{17, 23, 2147483647} {
		g := &profileUpdateGateway{}
		uc, _ := usecase.NewUpdateUserProfile(g)
		r := domain.UpdateUserProfileRequest{UserID: target, Version: 9007199254740993}
		ctx := context.Background()
		got, err := uc.Execute(ctx, allowed, r)
		raw, _ := json.Marshal(got)
		if err != nil || g.calls != 0 || string(raw) != `{"changed":false}` {
			t.Fatal(string(raw), err, g.calls)
		}
		name := "  NASA  o'neil  "
		r.Profile.Name = &name
		g.result = domain.UpdateUserProfileResponse{Changed: false, User: &domain.UserDetail{ID: target, Version: "1", Roles: []domain.UserDetailRole{{RoleID: 9, Name: "role"}}, ApplicationAccess: []domain.UserDetailApplicationAccess{{ApplicationID: 5, Name: "app"}}}}
		got, err = uc.Execute(ctx, allowed, r)
		if err != nil || g.calls != 1 || g.ctx != ctx || g.request.UserID != target || g.request.Version != r.Version || *g.request.Profile.Name != "NASA O'Neil" || got.Changed || got.User.Version != "1" {
			t.Fatal(got, err)
		}
		*g.request.Profile.Name = "changed"
		got.User.Name = "changed"
		got.User.Roles[0].Name = "changed"
		got.User.ApplicationAccess[0].Name = "changed"
		if name != "  NASA  o'neil  " || g.result.User.Name != "" || g.result.User.Roles[0].Name != "role" || g.result.User.ApplicationAccess[0].Name != "app" {
			t.Fatal("aliased")
		}
		failure := errors.New("dependency")
		g.err = failure
		if got, err = uc.Execute(ctx, allowed, r); err != failure || got.User != nil || got.Changed {
			t.Fatal(got, err)
		}
		g.err = nil
		for _, user := range []*domain.UserDetail{nil, {ID: 99}} {
			g.result.User = user
			if got, err = uc.Execute(ctx, allowed, r); !errors.Is(err, common.ErrInternalServerError) || got.User != nil {
				t.Fatal(got, err)
			}
		}
		for _, mode := range []string{"id", "version", "key", "body"} {
			r := domain.UpdateUserProfileRequest{UserID: target, Version: 1}
			switch mode {
			case "id":
				r.UserID = 0
			case "version":
				r.Version = 0
			case "key":
				empty := ""
				r.IdempotencyKey = &empty
			case "body":
				bad := ""
				r.Profile.Name = &bad
			}
			before := g.calls
			if _, err = uc.Execute(ctx, allowed, r); err == nil || g.calls != before {
				t.Fatal("invalid dispatch", mode)
			}
		}
	}
}
