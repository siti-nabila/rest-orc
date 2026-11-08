package test_scenarios

import (
	"context"
	"testing"

	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/applications/domain"
	"github.com/siti-nabila/rest-orc/internal/features/applications/usecase"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

type applicationGateway func(context.Context, domain.ApplicationListQuery) (domain.ApplicationPage[domain.Application], error)

func (f applicationGateway) ListApplications(c context.Context, q domain.ApplicationListQuery) (domain.ApplicationPage[domain.Application], error) {
	return f(c, q)
}

type contextKey struct{}

func All() []testutils.Scenario {
	return []testutils.Scenario{{Name: "application permission before business validation exact call context and result isolation", Run: read}}
}
func read(t *testing.T) {
	if _, e := usecase.NewList(nil); e == nil {
		t.Fatal("nil gateway")
	}
	ctx := context.WithValue(context.Background(), contextKey{}, "caller")
	calls := 0
	var failure error
	result := domain.ApplicationPage[domain.Application]{Items: []domain.Application{{ID: 5, Code: "clm", Name: " Legacy\té ", Status: "DISABLED"}}, Total: 1 << 32, Page: 99, Limit: 20, TotalPages: 2, HasPrev: true, NextCursor: "next"}
	uc, _ := usecase.NewList(applicationGateway(func(c context.Context, q domain.ApplicationListQuery) (domain.ApplicationPage[domain.Application], error) {
		calls++
		if c != ctx || q.Page != 1 || q.Limit != 20 || q.Keyword != "app name" {
			t.Fatal(c, q)
		}
		return result, failure
	}))
	actor := domain.ApplicationActor{UserID: 17, Permissions: map[string]bool{domain.ReadPermission: true}}
	input := domain.ApplicationListInput{"keyword": {" app\tname "}}
	if _, e := uc.Execute(nil, actor, input); e != common.ErrUnauthorized || calls != 0 {
		t.Fatal(e)
	}
	if _, e := uc.Execute(ctx, domain.ApplicationActor{}, input); e != common.ErrUnauthorized || calls != 0 {
		t.Fatal(e)
	}
	for _, perms := range []map[string]bool{nil, {}, {domain.ReadPermission: false}, {"*": true}, {"iam.permission.*": true}, {"iam.permission": true}, {"iam.user.read": true}, {"iam.role.read": true}, {"iam.user.application.grant": true}, {"iam.user.application.revoke": true}, {"admin": true}} {
		a := actor
		a.Permissions = perms
		out, e := uc.Execute(ctx, a, domain.ApplicationListInput{"page": {"bad"}})
		if e != common.ErrForbidden || out.Items != nil || calls != 0 {
			t.Fatal(perms, out, e)
		}
	}
	a := actor
	a.MustChangePassword = true
	if _, e := uc.Execute(ctx, a, input); e != common.ErrForbidden || calls != 0 {
		t.Fatal(e)
	}
	if _, e := uc.Execute(ctx, actor, domain.ApplicationListInput{"limit": {"0"}}); e == nil || calls != 0 {
		t.Fatal("business validation")
	}
	out, e := uc.Execute(ctx, actor, input)
	if e != nil || calls != 1 || out.Total != 1<<32 || out.Page != 99 || !out.HasPrev || out.NextCursor != "next" || out.Items[0].Status != "DISABLED" {
		t.Fatal(out, e)
	}
	out.Items[0].Code = "mutated"
	if result.Items[0].Code != "clm" {
		t.Fatal("result alias")
	}
	result.Items = nil
	out, e = uc.Execute(ctx, actor, input)
	if e != nil || out.Items == nil || len(out.Items) != 0 {
		t.Fatal(out, e)
	}
	for _, err := range []error{common.ErrForbidden, common.ErrNotFound, common.ErrInternalServerError, context.Canceled, context.DeadlineExceeded} {
		failure = err
		before := calls
		out, e = uc.Execute(ctx, actor, input)
		if e != err || out.Items != nil || calls != before+1 {
			t.Fatal(out, e)
		}
	}
}
