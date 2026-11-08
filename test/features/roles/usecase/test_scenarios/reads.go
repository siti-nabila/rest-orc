package test_scenarios

import (
	"context"
	"errors"
	"testing"

	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/roles/domain"
	"github.com/siti-nabila/rest-orc/internal/features/roles/usecase"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

type gateway struct {
	calls  int
	ctx    context.Context
	q      domain.ListQuery
	id     uint64
	page   domain.Page[domain.Role]
	detail domain.Detail
	err    error
}

func (g *gateway) List(c context.Context, q domain.ListQuery) (domain.Page[domain.Role], error) {
	g.calls++
	g.ctx = c
	g.q = q
	return g.page, g.err
}
func (g *gateway) Get(c context.Context, id uint64) (domain.Detail, error) {
	g.calls++
	g.ctx = c
	g.id = id
	return g.detail, g.err
}
func All() []testutils.Scenario {
	return []testutils.Scenario{{Name: "roles guards before business validation single dispatch and independent output", Run: reads}}
}
func reads(t *testing.T) {
	if _, err := usecase.NewList(nil); err == nil {
		t.Fatal("nil list gateway")
	}
	if _, err := usecase.NewGet(nil); err == nil {
		t.Fatal("nil get gateway")
	}
	ctx := context.Background()
	allowed := domain.Actor{UserID: 17, Permissions: map[string]bool{domain.ReadPermission: true}}
	for _, tc := range []struct {
		ctx  context.Context
		a    domain.Actor
		want error
	}{
		{nil, allowed, common.ErrUnauthorized}, {ctx, domain.Actor{}, common.ErrUnauthorized}, {ctx, domain.Actor{UserID: 17}, common.ErrForbidden},
		{ctx, domain.Actor{UserID: 17, MustChangePassword: true, Permissions: allowed.Permissions}, common.ErrForbidden},
		{ctx, domain.Actor{UserID: 17, Permissions: map[string]bool{domain.ReadPermission: false}}, common.ErrForbidden},
		{ctx, domain.Actor{UserID: 17, Permissions: map[string]bool{"iam.role.*": true, "iam.role": true, "iam.user.read": true, "iam.role.create": true, "iam.role.update": true}}, common.ErrForbidden},
	} {
		g := &gateway{}
		list, _ := usecase.NewList(g)
		get, _ := usecase.NewGet(g)
		if got, err := list.Execute(tc.ctx, tc.a, domain.ListInput{"page": {"bad"}}); !errors.Is(err, tc.want) || got.Items != nil {
			t.Fatal(got, err)
		}
		if got, err := get.Execute(tc.ctx, tc.a, "bad"); !errors.Is(err, tc.want) || got.Role.ID != 0 || g.calls != 0 {
			t.Fatal(got, err)
		}
	}
	desc, at := " legacy ", "time"
	role := domain.Role{ID: 10, Description: &desc, CreatedAt: &at, UpdatedAt: &at, Version: "9007199254740993"}
	g := &gateway{page: domain.Page[domain.Role]{Items: []domain.Role{role}, Total: 4294967297}, detail: domain.Detail{Role: role, Permissions: []domain.Permission{{ID: 7, Description: "permission"}}}}
	list, _ := usecase.NewList(g)
	get, _ := usecase.NewGet(g)
	if _, err := list.Execute(ctx, allowed, domain.ListInput{"page": {"0"}}); err == nil || g.calls != 0 {
		t.Fatal("invalid list dispatched")
	}
	if _, err := get.Execute(ctx, allowed, "0"); err == nil || g.calls != 0 {
		t.Fatal("invalid get dispatched")
	}
	page, err := list.Execute(ctx, allowed, domain.ListInput{"keyword": {"  legal\t reviewer "}, "statuses": {"DISABLED,ACTIVE"}})
	if err != nil || g.calls != 1 || g.ctx != ctx || g.q.Keyword != "legal reviewer" || g.q.Statuses[0] != "ACTIVE" || page.Total != 4294967297 {
		t.Fatal(page, err)
	}
	*page.Items[0].Description = "changed"
	*page.Items[0].CreatedAt = "changed"
	*page.Items[0].UpdatedAt = "changed"
	page.Items[0].ID = 99
	detail, err := get.Execute(ctx, allowed, "00010")
	if err != nil || g.calls != 2 || g.id != 10 || g.ctx != ctx || detail.Role.Version != "9007199254740993" {
		t.Fatal(detail, err)
	}
	*detail.Role.Description = "changed"
	*detail.Role.CreatedAt = "changed"
	detail.Permissions[0].Description = "changed"
	if desc != " legacy " || at != "time" || g.page.Items[0].ID != 10 || g.detail.Permissions[0].Description != "permission" {
		t.Fatal("aliased")
	}
	g.err = errors.New("dependency")
	if p, err := list.Execute(ctx, allowed, nil); err != g.err || p.Items != nil {
		t.Fatal(p, err)
	}
	if d, err := get.Execute(ctx, allowed, "10"); err != g.err || d.Role.ID != 0 {
		t.Fatal(d, err)
	}
	g.err = nil
	g.detail.Role.ID = 11
	if _, err := get.Execute(ctx, allowed, "10"); !errors.Is(err, common.ErrInternalServerError) {
		t.Fatal("wrong target")
	}
	g.page.Items = nil
	p, err := list.Execute(ctx, allowed, nil)
	if err != nil || p.Items == nil {
		t.Fatal("empty items")
	}
	g.detail = domain.Detail{Role: role}
	d, err := get.Execute(ctx, allowed, "10")
	if err != nil || d.Permissions == nil {
		t.Fatal("empty permissions")
	}
}
