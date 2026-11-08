package test_scenarios

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	rolev1 "github.com/siti-nabila/api-contracts/pb/role/v1"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/roles/domain"
	gateway "github.com/siti-nabila/rest-orc/internal/features/roles/gateway/authservice"
	shared "github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type client struct {
	list func(context.Context, *rolev1.ListRolesRequest) (*rolev1.ListRolesResponse, error)
	get  func(context.Context, *rolev1.GetRoleRequest) (*rolev1.GetRoleResponse, error)
}

func (c *client) ListRoles(ctx context.Context, r *rolev1.ListRolesRequest) (*rolev1.ListRolesResponse, error) {
	return c.list(ctx, r)
}
func (c *client) GetRole(ctx context.Context, r *rolev1.GetRoleRequest) (*rolev1.GetRoleResponse, error) {
	return c.get(ctx, r)
}
func All() []testutils.Scenario {
	return []testutils.Scenario{{Name: "role list exact query metadata safe fields nullability counts and isolation", Run: list}, {Name: "role detail permissions identity and read metadata", Run: get}, {Name: "malformed role successes fail without partial output", Run: malformed}}
}
func list(t *testing.T) {
	if _, err := gateway.New(nil); err == nil {
		t.Fatal("nil client")
	}
	q := domain.ListQuery{Page: 2, Limit: 100, Keyword: "legal reviewer", ApplicationCode: "clm", Statuses: []string{"ACTIVE", "DISABLED"}, LastID: " opaque+/=% "}
	md := metadata.Pairs("authorization", "Bearer actor", "version", "old", "idempotency-key", "old", "x-request-id", "roles")
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	calls := 0
	response := shared.RoleList()
	var failure error
	c := &client{list: func(ctx context.Context, r *rolev1.ListRolesRequest) (*rolev1.ListRolesResponse, error) {
		calls++
		actual, _ := metadata.FromOutgoingContext(ctx)
		want := md.Copy()
		want.Delete("version")
		want.Delete("idempotency-key")
		if !reflect.DeepEqual(actual, want) || r.Query.Page != q.Page || r.Query.Limit != q.Limit || r.Query.LastId != q.LastID || len(r.Query.Fields) != 0 || r.Filter.ApplicationCode != q.ApplicationCode || !reflect.DeepEqual(r.Filter.Statuses, q.Statuses) {
			t.Fatal("payload/metadata")
		}
		for i, field := range []string{"created_at", "updated_at", "id"} {
			if len(r.Query.Sort) != 3 || r.Query.Sort[i].Field != field || !r.Query.Sort[i].Desc {
				t.Fatal("sort")
			}
		}
		if q.Keyword == "" {
			if r.Query.Search != nil {
				t.Fatal("nil search")
			}
		} else {
			if r.Query.Search == nil || r.Query.Search.Keyword != q.Keyword || int32(r.Query.Search.Mode) != 5 || !reflect.DeepEqual(r.Query.Search.Fields, []string{"keyword"}) {
				t.Fatal("logical search")
			}
			r.Query.Search.Fields[0] = "changed"
		}
		if len(r.Filter.Statuses) > 0 {
			r.Filter.Statuses[0] = "changed"
		}
		actual.Set("authorization", "changed")
		return response, failure
	}}
	g, _ := gateway.New(c)
	if _, err := g.List(nil, q); err == nil {
		t.Fatal("nil context")
	}
	invalid := q
	invalid.Page = 0
	if _, err := g.List(ctx, invalid); err == nil || calls != 0 {
		t.Fatal("invalid dispatch")
	}
	got, err := g.List(ctx, q)
	raw, _ := json.Marshal(got.Items[0])
	if err != nil || calls != 1 || string(raw) != shared.RoleJSON {
		t.Fatal(string(raw), err)
	}
	*got.Items[0].Description = "changed"
	*got.Items[0].CreatedAt = "changed"
	if *response.Items[0].Description != "Reviewer legal kontrak" || response.Items[0].CreatedAt.Nanos != 123456789 || q.Statuses[0] != "ACTIVE" || md.Get("authorization")[0] != "Bearer actor" {
		t.Fatal("alias")
	}
	response.Items[0].Description = nil
	response.Items[0].CreatedAt = nil
	response.Items[0].UpdatedAt = nil
	response.Items[0].Version = 9223372036854775807
	response.Items[0].RoleCode = 2147483647
	response.Items[0].Id = 2147483647
	response.Items[0].ApplicationId = 2147483647
	q.Keyword = ""
	q.Statuses = nil
	got, err = g.List(ctx, q)
	if err != nil || got.Items[0].Description != nil || got.Items[0].CreatedAt != nil || got.Items[0].Version != "9223372036854775807" || got.Items[0].RoleCode != 2147483647 {
		t.Fatal(got, err)
	}
	for _, total := range []int64{0, 4294967297, 9223372036854775807} {
		response = shared.RoleList()
		response.Items = nil
		response.Meta.Total = total
		response.Meta.Page = 2147483647
		response.Meta.TotalPages = 0
		response.Meta.HasPrev = true
		response.Meta.HasNext = true
		response.NextCursor = "next+/=%"
		got, err = g.List(ctx, q)
		if err != nil || got.Total != total || got.Page != 2147483647 || got.TotalPages != 0 || !got.HasPrev || !got.HasNext || got.NextCursor != "next+/=%" || got.Items == nil || len(got.Items) != 0 {
			t.Fatal(got, err)
		}
	}
	failure = errors.New("dependency")
	if got, err = g.List(ctx, q); err != failure || got.Items != nil {
		t.Fatal(got, err)
	}
}
func get(t *testing.T) {
	md := metadata.Pairs("authorization", "Bearer actor", "version", "old", "idempotency-key", "old")
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	calls := 0
	response := shared.RoleDetail()
	var failure error
	c := &client{get: func(c context.Context, r *rolev1.GetRoleRequest) (*rolev1.GetRoleResponse, error) {
		calls++
		actual, _ := metadata.FromOutgoingContext(c)
		if r.RoleId != 10 || len(actual.Get("version")) != 0 || len(actual.Get("idempotency-key")) != 0 || actual.Get("authorization")[0] != "Bearer actor" {
			t.Fatal("read inputs")
		}
		actual.Set("authorization", "changed")
		return response, failure
	}}
	g, _ := gateway.New(c)
	if _, err := g.Get(nil, 10); err == nil {
		t.Fatal("nil context")
	}
	for _, id := range []uint64{0, 2147483648} {
		if _, err := g.Get(ctx, id); err == nil || calls != 0 {
			t.Fatal("invalid dispatch")
		}
	}
	for _, system := range []bool{false, true} {
		response.Role.IsSystem = system
		if system {
			response.Role.ApplicationCode = "iam"
			response.Permissions[0].ApplicationCode = "iam"
			response.Role.Code = "iam.reader"
		}
		got, err := g.Get(ctx, 10)
		if err != nil || got.Role.IsSystem != system || len(got.Permissions) != 1 {
			t.Fatal(got, err)
		}
		*got.Role.Description = "changed"
		*got.Role.CreatedAt = "changed"
		got.Permissions[0].Description = "changed"
		if *response.Role.Description != "Reviewer legal kontrak" || response.Permissions[0].Description != "Read contracts" {
			t.Fatal("alias")
		}
	}
	response.Permissions = nil
	got, err := g.Get(ctx, 10)
	if err != nil || got.Permissions == nil {
		t.Fatal(got, err)
	}
	failure = errors.New("dependency")
	if got, err = g.Get(ctx, 10); err != failure || got.Role.ID != 0 {
		t.Fatal(got, err)
	}
	if md.Get("authorization")[0] != "Bearer actor" {
		t.Fatal("parent mutated")
	}
}
func malformed(t *testing.T) {
	for _, mode := range []string{"nil role", "zero id", "large id", "zero app", "large app", "zero role code", "negative role code", "zero version", "large version", "bad status", "created timestamp", "updated timestamp"} {
		t.Run(mode, func(t *testing.T) {
			r := shared.RoleSummary()
			switch mode {
			case "nil role":
				r = nil
			case "zero id":
				r.Id = 0
			case "large id":
				r.Id = 2147483648
			case "zero app":
				r.ApplicationId = 0
			case "large app":
				r.ApplicationId = 2147483648
			case "zero role code":
				r.RoleCode = 0
			case "negative role code":
				r.RoleCode = -1
			case "zero version":
				r.Version = 0
			case "large version":
				r.Version = 1 << 63
			case "bad status":
				r.Status = "REVOKED"
			case "created timestamp":
				r.CreatedAt = &timestamppb.Timestamp{Nanos: -1}
			case "updated timestamp":
				r.UpdatedAt = &timestamppb.Timestamp{Seconds: 253402300800}
			}
			lr, gr := shared.RoleList(), shared.RoleDetail()
			lr.Items = append(lr.Items, r)
			gr.Role = r
			c := &client{list: func(context.Context, *rolev1.ListRolesRequest) (*rolev1.ListRolesResponse, error) { return lr, nil }, get: func(context.Context, *rolev1.GetRoleRequest) (*rolev1.GetRoleResponse, error) { return gr, nil }}
			g, _ := gateway.New(c)
			if got, err := g.List(context.Background(), domain.ListQuery{Page: 1, Limit: 20}); !errors.Is(err, common.ErrInternalServerError) || got.Items != nil {
				t.Fatal(mode, got, err)
			}
			if got, err := g.Get(context.Background(), 10); !errors.Is(err, common.ErrInternalServerError) || got.Role.ID != 0 {
				t.Fatal(mode, got, err)
			}
		})
	}
	for _, mode := range []string{"nil response", "nil meta", "page", "limit zero", "limit large", "total", "total pages"} {
		t.Run(mode, func(t *testing.T) {
			r := shared.RoleList()
			switch mode {
			case "nil response":
				r = nil
			case "nil meta":
				r.Meta = nil
			case "page":
				r.Meta.Page = 0
			case "limit zero":
				r.Meta.Limit = 0
			case "limit large":
				r.Meta.Limit = 101
			case "total":
				r.Meta.Total = -1
			case "total pages":
				r.Meta.TotalPages = -1
			}
			g, _ := gateway.New(&client{list: func(context.Context, *rolev1.ListRolesRequest) (*rolev1.ListRolesResponse, error) { return r, nil }})
			if got, err := g.List(context.Background(), domain.ListQuery{Page: 1, Limit: 20}); err == nil || got.Items != nil {
				t.Fatal(got, err)
			}
		})
	}
	for _, mode := range []string{"nil response", "wrong id", "nil permission", "zero permission", "large permission", "zero permission app", "large permission app", "different app id", "different app code"} {
		t.Run(mode, func(t *testing.T) {
			r := shared.RoleDetail()
			switch mode {
			case "nil response":
				r = nil
			case "wrong id":
				r.Role.Id = 11
			case "nil permission":
				r.Permissions[0] = nil
			case "zero permission":
				r.Permissions[0].Id = 0
			case "large permission":
				r.Permissions[0].Id = 2147483648
			case "zero permission app":
				r.Permissions[0].ApplicationId = 0
			case "large permission app":
				r.Permissions[0].ApplicationId = 2147483648
			case "different app id":
				r.Permissions[0].ApplicationId = 6
			case "different app code":
				r.Permissions[0].ApplicationCode = "iam"
			}
			g, _ := gateway.New(&client{get: func(context.Context, *rolev1.GetRoleRequest) (*rolev1.GetRoleResponse, error) { return r, nil }})
			if got, err := g.Get(context.Background(), 10); !errors.Is(err, common.ErrInternalServerError) || got.Role.ID != 0 {
				t.Fatal(got, err)
			}
		})
	}
}
