package test_scenarios

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	applicationv1 "github.com/siti-nabila/api-contracts/pb/application/v1"
	paginatorv1 "github.com/siti-nabila/api-contracts/pb/paginator/v1"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/applications/domain"
	gateway "github.com/siti-nabila/rest-orc/internal/features/applications/gateway/authservice"
	"github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc/metadata"
)

type applicationClient func(context.Context, *applicationv1.ListApplicationsRequest) (*applicationv1.ListApplicationsResponse, error)

func (f applicationClient) ListApplications(c context.Context, r *applicationv1.ListApplicationsRequest) (*applicationv1.ListApplicationsResponse, error) {
	return f(c, r)
}
func All() []testutils.Scenario {
	return []testutils.Scenario{{Name: "application full page payload metadata text totals and ownership", Run: mapping}, {Name: "application guards malformed successes no partial output and error identity", Run: failures}}
}
func mapping(t *testing.T) {
	if _, e := gateway.New(nil); e == nil {
		t.Fatal("nil client")
	}
	for _, search := range []string{"", "a b"} {
		for _, states := range [][]string{nil, {"ACTIVE", "DISABLED"}} {
			t.Run(search+"/"+jsonText(states), func(t *testing.T) {
				q := domain.ApplicationListQuery{Page: 2, Limit: 30, Keyword: search, LastID: " opaque+/=% ", Statuses: states}
				md := metadata.Pairs("authorization", "Bearer actor", "x-request-id", "app", "version", "old", "idempotency-key", "old", "idempotency-key", "old2")
				ctx := metadata.NewOutgoingContext(context.Background(), md)
				calls := 0
				r := fixtures.ApplicationList()
				r.Meta = &paginatorv1.PageMeta{Page: 99, Limit: 30, Total: 9223372036854775807, TotalPages: 42, HasPrev: true, HasNext: false}
				r.NextCursor = "next+/="
				g, _ := gateway.New(applicationClient(func(c context.Context, in *applicationv1.ListApplicationsRequest) (*applicationv1.ListApplicationsResponse, error) {
					calls++
					got, _ := metadata.FromOutgoingContext(c)
					want := md.Copy()
					want.Delete("version")
					want.Delete("idempotency-key")
					if !reflect.DeepEqual(got, want) || in.Query.Page != 2 || in.Query.Limit != 30 || in.Query.LastId != q.LastID || len(in.Query.Fields) != 0 || len(in.Query.Sort) != 1 || in.Query.Sort[0].Field != "id" || !in.Query.Sort[0].Desc || !reflect.DeepEqual(in.Filter.Statuses, states) {
						t.Fatal(in, got)
					}
					if search == "" {
						if in.Query.Search != nil {
							t.Fatal("invented search")
						}
					} else if in.Query.Search.Keyword != search || !reflect.DeepEqual(in.Query.Search.Fields, []string{"keyword"}) || in.Query.Search.Mode != paginatorv1.SearchMode_SEARCH_MODE_FULL_TEXT_TRIGRAM {
						t.Fatal(in)
					}
					if len(in.Filter.Statuses) > 0 {
						in.Filter.Statuses[0] = "mutated"
					}
					got.Set("authorization", "mutated")
					return r, nil
				}))
				out, e := g.ListApplications(ctx, q)
				if e != nil || calls != 1 || out.Total != 9223372036854775807 || out.Page != 99 || out.TotalPages != 42 || !out.HasPrev || out.HasNext || out.NextCursor != "next+/=" {
					t.Fatal(out, e)
				}
				b, _ := json.Marshal(out.Items[0])
				if string(b) != fixtures.ApplicationJSON {
					t.Fatal(string(b))
				}
				out.Items[0].Code = "mutated"
				if r.Items[0].Code != "clm" {
					t.Fatal("result alias")
				}
				r.Items[0].Id = 2147483647
				r.Items[0].IsSystem = true
				r.Items[0].Status = "ACTIVE"
				r.Items[0].Code = "iam"
				out, e = g.ListApplications(ctx, q)
				if e != nil || !out.Items[0].IsSystem || out.Items[0].Code != "iam" {
					t.Fatal(out, e)
				}
				for _, total := range []int64{0, 42} {
					r.Items = nil
					r.Meta.Total = total
					r.Meta.TotalPages = 0
					out, e = g.ListApplications(ctx, q)
					if e != nil || out.Items == nil || len(out.Items) != 0 || out.Total != total || out.Page != 99 {
						t.Fatal(out, e)
					}
				}
				if md.Get("authorization")[0] != "Bearer actor" || len(md.Get("idempotency-key")) != 2 || len(states) > 0 && states[0] != "ACTIVE" {
					t.Fatal("input mutation")
				}
			})
		}
	}
}
func jsonText(v any) string { b, _ := json.Marshal(v); return string(b) }
func failures(t *testing.T) {
	calls := 0
	r := fixtures.ApplicationList()
	var failure error
	g, _ := gateway.New(applicationClient(func(c context.Context, _ *applicationv1.ListApplicationsRequest) (*applicationv1.ListApplicationsResponse, error) {
		calls++
		return r, failure
	}))
	q, _ := domain.ParseApplicationListQuery(nil)
	ctx := context.Background()
	if _, e := g.ListApplications(nil, q); e == nil {
		t.Fatal("nil context")
	}
	bad := q
	bad.Page = 0
	if _, e := g.ListApplications(ctx, bad); e == nil || calls != 0 {
		t.Fatal("guard dispatch")
	}
	for _, mutate := range []func(*applicationv1.ListApplicationsResponse){func(r *applicationv1.ListApplicationsResponse) { r.Meta = nil }, func(r *applicationv1.ListApplicationsResponse) { r.Meta.Page = 0 }, func(r *applicationv1.ListApplicationsResponse) { r.Meta.Limit = 0 }, func(r *applicationv1.ListApplicationsResponse) { r.Meta.Limit = 101 }, func(r *applicationv1.ListApplicationsResponse) { r.Meta.Total = -1 }, func(r *applicationv1.ListApplicationsResponse) { r.Meta.TotalPages = -1 }, func(r *applicationv1.ListApplicationsResponse) { r.Items = append(r.Items, nil) }, func(r *applicationv1.ListApplicationsResponse) {
		r.Items = append(r.Items, &applicationv1.ApplicationItem{Id: 0, Status: "ACTIVE"})
	}, func(r *applicationv1.ListApplicationsResponse) { r.Items[0].Id = 2147483648 }, func(r *applicationv1.ListApplicationsResponse) { r.Items[0].Status = "REVOKED" }} {
		r = fixtures.ApplicationList()
		mutate(r)
		before := calls
		out, e := g.ListApplications(ctx, q)
		if !errors.Is(e, common.ErrInternalServerError) || out.Items != nil || calls != before+1 {
			t.Fatal(out, e)
		}
	}
	r = nil
	if _, e := g.ListApplications(ctx, q); !errors.Is(e, common.ErrInternalServerError) {
		t.Fatal(e)
	}
	for _, e := range []error{common.ErrForbidden, common.ErrNotFound, common.ErrInternalServerError, context.Canceled, context.DeadlineExceeded} {
		failure = e
		before := calls
		out, got := g.ListApplications(ctx, q)
		if got != e || out.Items != nil || calls != before+1 {
			t.Fatal(out, got)
		}
	}
}
