package test_scenarios

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"testing"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	gateway "github.com/siti-nabila/rest-orc/internal/features/users/gateway/authservice"
	"github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type detailClient func(context.Context, *userv1.GetUserRequest) (*userv1.GetUserResponse, error)

func (f detailClient) GetUser(c context.Context, r *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	return f(c, r)
}
func UserDetailScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "detail gateway explicit mapping metadata precision and fail closed", Run: detailGatewayMapping}}
}
func detailGatewayMapping(t *testing.T) {
	if _, err := gateway.NewUserDetail(nil); err == nil {
		t.Fatal("nil client")
	}
	md := metadata.Pairs("authorization", "Bearer actor", "version", "88", "idempotency-key", "parent", "x-request-id", "detail")
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	calls := 0
	result := fixtures.UserDetailResponse(23)
	var failure error
	sdk := detailClient(func(c context.Context, r *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
		calls++
		if r.Id != 23 {
			t.Fatal(r)
		}
		actual, _ := metadata.FromOutgoingContext(c)
		want := md.Copy()
		want.Delete("version")
		want.Delete("idempotency-key")
		if !reflect.DeepEqual(actual, want) {
			t.Fatal(actual, want)
		}
		return result, failure
	})
	g, _ := gateway.NewUserDetail(sdk)
	if _, err := g.GetUser(nil, 23); err == nil {
		t.Fatal("nil context")
	}
	for _, id := range []uint64{0, 2147483648} {
		if _, err := g.GetUser(ctx, id); err == nil {
			t.Fatal("invalid ID")
		}
	}
	if calls != 0 {
		t.Fatal("invalid dispatch")
	}
	got, err := g.GetUser(ctx, 23)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(got)
	var expected struct{ Data json.RawMessage }
	if json.Unmarshal([]byte(fixtures.UserDetailBody), &expected) != nil || string(raw) != string(expected.Data) {
		t.Fatal(string(raw), string(expected.Data))
	}
	got.Roles[0].Name = "changed"
	got.ApplicationAccess[0].Name = "changed"
	if result.User.Roles[0].Name != "Reader" || result.User.ApplicationAccess[0].Name != "CLM" {
		t.Fatal("aliased")
	}
	original, _ := metadata.FromOutgoingContext(ctx)
	if !reflect.DeepEqual(original, md) || len(md.Get("version")) != 1 {
		t.Fatal("parent modified")
	}
	for _, state := range []string{"ACTIVE", "PENDING", "SUSPENDED", "DEACTIVATED"} {
		for _, restricted := range []bool{false, true} {
			for _, v := range []uint64{1, 9007199254740993, 9223372036854775807} {
				result = fixtures.UserDetailResponse(23)
				result.User.Status = state
				result.User.MustChangePassword = restricted
				result.User.Version = v
				got, err = g.GetUser(ctx, 23)
				if err != nil || got.Status != state || got.MustChangePassword != restricted || got.Version != strconv.FormatUint(v, 10) || len(got.Roles) != 2 {
					t.Fatal(got, err)
				}
			}
		}
	}
	result = fixtures.UserDetailResponse(23)
	result.User.Roles = nil
	result.User.ApplicationAccess = nil
	got, err = g.GetUser(ctx, 23)
	if err != nil || got.Roles == nil || got.ApplicationAccess == nil {
		t.Fatal(got, err)
	}
	for _, tc := range []struct {
		name   string
		mutate func()
	}{
		{"nil response", func() { result = nil }}, {"nil user", func() { result.User = nil }},
		{"zero ID", func() { result.User.Id = 0 }}, {"large ID", func() { result.User.Id = 2147483648 }},
		{"zero version", func() { result.User.Version = 0 }}, {"version overflow", func() { result.User.Version = 1 << 63 }},
		{"no timestamp", func() { result.User.CreatedAt = nil }}, {"invalid timestamp", func() { result.User.CreatedAt = &timestamppb.Timestamp{Nanos: -1} }},
		{"invalid status", func() { result.User.Status = "UNKNOWN" }},
		{"nil role", func() { result.User.Roles[0] = nil }}, {"zero role ID", func() { result.User.Roles[0].RoleId = 0 }}, {"large role ID", func() { result.User.Roles[0].RoleId = 2147483648 }},
		{"nil app", func() { result.User.ApplicationAccess[0] = nil }}, {"zero app ID", func() { result.User.ApplicationAccess[0].ApplicationId = 0 }}, {"large app ID", func() { result.User.ApplicationAccess[0].ApplicationId = 2147483648 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result = fixtures.UserDetailResponse(23)
			tc.mutate()
			got, err := g.GetUser(ctx, 23)
			if !errors.Is(err, common.ErrInternalServerError) || got.ID != 0 || got.Roles != nil {
				t.Fatal(got, err)
			}
		})
	}
	failure = errors.New("dependency")
	if _, err = g.GetUser(ctx, 23); err != failure {
		t.Fatal(err)
	}
}
