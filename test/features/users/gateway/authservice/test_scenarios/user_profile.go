package test_scenarios

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	gateway "github.com/siti-nabila/rest-orc/internal/features/users/gateway/authservice"
	shared "github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type targetProfileClient func(context.Context, *userv1.UpdateUserRequest) (*userv1.UpdateUserResponse, error)

func (f targetProfileClient) UpdateUser(c context.Context, r *userv1.UpdateUserRequest) (*userv1.UpdateUserResponse, error) {
	return f(c, r)
}
func UserProfileScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "target profile gateway mask optional values metadata safe snapshot and isolation", Run: targetProfileGateway}}
}
func targetProfileGateway(t *testing.T) {
	if _, err := gateway.NewUserProfile(nil); err == nil {
		t.Fatal("nil client")
	}
	for mask := 1; mask < 16; mask++ {
		for _, keyed := range []bool{true, false} {
			name, address, phone, country, key := "NASA O'Neil", " X\r\nY ", "628123456789", "+62", "opaque, key"
			r := domain.UpdateUserProfileRequest{UserID: 23, Version: 9007199254740993}
			wantMask := []string{}
			if mask&1 != 0 {
				r.Profile.Name = &name
				wantMask = append(wantMask, "name")
			}
			if mask&2 != 0 {
				r.Profile.Address = &address
				wantMask = append(wantMask, "address")
			}
			if mask&4 != 0 {
				r.Profile.Phone = &phone
				wantMask = append(wantMask, "phone")
			}
			if mask&8 != 0 {
				r.Profile.CountryCode = &country
				wantMask = append(wantMask, "country_code")
			}
			if keyed {
				r.IdempotencyKey = &key
			}
			md := metadata.Pairs("authorization", "Bearer actor", "version", "old", "idempotency-key", "old", "idempotency-key", "old2", "x-request-id", "target-profile")
			ctx := metadata.NewOutgoingContext(context.Background(), md)
			calls := 0
			result := &userv1.UpdateUserResponse{User: shared.UserDetailResponse(23).User, Changed: false}
			var failure error
			sdk := targetProfileClient(func(c context.Context, p *userv1.UpdateUserRequest) (*userv1.UpdateUserResponse, error) {
				calls++
				actual, _ := metadata.FromOutgoingContext(c)
				want := md.Copy()
				want.Delete("version")
				want.Delete("idempotency-key")
				if keyed {
					want.Set("idempotency-key", key)
				}
				if !reflect.DeepEqual(actual, want) || p.UserId != 23 || p.ExpectedUserVersion == nil || *p.ExpectedUserVersion != r.Version || p.FieldMask == nil || !reflect.DeepEqual(p.FieldMask.Paths, wantMask) {
					t.Fatal(p, actual, want)
				}
				for _, pair := range []struct{ got, want *string }{{p.Name, r.Profile.Name}, {p.Address, r.Profile.Address}, {p.Phone, r.Profile.Phone}, {p.CountryCode, r.Profile.CountryCode}} {
					if !reflect.DeepEqual(pair.got, pair.want) || pair.got != nil && pair.got == pair.want {
						t.Fatal("presence/value/pointer", p)
					}
					if pair.got != nil {
						*pair.got = "SDK mutation"
					}
				}
				p.FieldMask.Paths[0] = "SDK mutation"
				*p.ExpectedUserVersion = 1
				actual.Set("x-request-id", "SDK mutation")
				return result, failure
			})
			g, _ := gateway.NewUserProfile(sdk)
			if _, err := g.UpdateUserProfile(nil, r); err == nil {
				t.Fatal("nil context")
			}
			for _, mode := range []string{"target", "version", "key", "empty"} {
				invalid := r
				switch mode {
				case "target":
					invalid.UserID = 0
				case "version":
					invalid.Version = 1 << 63
				case "key":
					empty := ""
					invalid.IdempotencyKey = &empty
				case "empty":
					invalid.Profile = domain.UpdateProfileRequest{}
				}
				if _, err := g.UpdateUserProfile(ctx, invalid); err == nil || calls != 0 {
					t.Fatal("invalid dispatch", mode)
				}
			}
			got, err := g.UpdateUserProfile(ctx, r)
			if err != nil || calls != 1 || got.User == nil || got.Changed || got.User.Version != "9007199254740993" {
				t.Fatal(got, err)
			}
			raw, _ := json.Marshal(got.User)
			var expected struct{ Data json.RawMessage }
			_ = json.Unmarshal([]byte(shared.UserDetailBody), &expected)
			if string(raw) != string(expected.Data) {
				t.Fatal(string(raw), string(expected.Data))
			}
			got.User.Roles[0].Name = "changed"
			got.User.ApplicationAccess[0].Name = "changed"
			if result.User.Roles[0].Name != "Reader" || result.User.ApplicationAccess[0].Name != "CLM" {
				t.Fatal("response alias")
			}
			if name != "NASA O'Neil" || address != " X\r\nY " || phone != "628123456789" || country != "+62" || key != "opaque, key" || r.Version != 9007199254740993 || md.Get("x-request-id")[0] != "target-profile" {
				t.Fatal("caller mutated")
			}
			for _, s := range []string{"PENDING", "ACTIVE", "SUSPENDED", "DEACTIVATED"} {
				result.User = shared.UserDetailResponse(23).User
				result.User.Status = s
				result.User.MustChangePassword = true
				result.User.Version = 9223372036854775807
				result.User.Roles = nil
				result.User.ApplicationAccess = nil
				got, err = g.UpdateUserProfile(ctx, r)
				if err != nil || got.User.Status != s || !got.User.MustChangePassword || got.User.Version != "9223372036854775807" || got.User.Roles == nil || got.User.ApplicationAccess == nil {
					t.Fatal(got, err)
				}
			}
			for _, mode := range []string{"nil", "nil user", "wrong target", "zero ID", "large ID", "zero version", "overflow", "status", "missing timestamp", "timestamp", "nil role", "role ID", "nil app", "app ID"} {
				result = &userv1.UpdateUserResponse{User: shared.UserDetailResponse(23).User, Changed: true}
				switch mode {
				case "nil":
					result = nil
				case "nil user":
					result.User = nil
				case "wrong target":
					result.User.Id = 99
				case "zero ID":
					result.User.Id = 0
				case "large ID":
					result.User.Id = 2147483648
				case "zero version":
					result.User.Version = 0
				case "overflow":
					result.User.Version = 1 << 63
				case "status":
					result.User.Status = "unknown"
				case "missing timestamp":
					result.User.CreatedAt = nil
				case "timestamp":
					result.User.CreatedAt = &timestamppb.Timestamp{Nanos: -1}
				case "nil role":
					result.User.Roles[0] = nil
				case "role ID":
					result.User.Roles[0].RoleId = 0
				case "nil app":
					result.User.ApplicationAccess[0] = nil
				case "app ID":
					result.User.ApplicationAccess[0].ApplicationId = 2147483648
				}
				got, err = g.UpdateUserProfile(ctx, r)
				if !errors.Is(err, common.ErrInternalServerError) || got.User != nil || got.Changed {
					t.Fatal(mode, got, err)
				}
			}
			failure = errors.New("dependency")
			if got, err = g.UpdateUserProfile(ctx, r); err != failure || got.User != nil || got.Changed {
				t.Fatal(got, err)
			}
		}
	}
}
