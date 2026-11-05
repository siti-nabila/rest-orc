package test_scenarios

import (
	"context"
	"errors"
	profilev1 "github.com/siti-nabila/api-contracts/pb/profile/v1"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	usersgateway "github.com/siti-nabila/rest-orc/internal/features/users/gateway/authservice"
	"github.com/siti-nabila/rest-orc/test/features/users/gateway/authservice/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"reflect"
	"testing"
)

func ProfileScenarios() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "profile gateway exact mask for every subset and safe authoritative output", Run: profileMasks},
		{Name: "profile gateway rejects empty selection invalid successes and preserves error", Run: profileGatewayGuards},
	}
}
func profileMasks(t *testing.T) {
	for bits := 1; bits < 16; bits++ {
		for _, changed := range []bool{true, false} {
			request := domain.UpdateProfileRequest{}
			name, address, phone, country := "NASA", " A\r\nB ", "812345678901", "+62"
			want := []string{}
			if bits&1 != 0 {
				request.Name = &name
				want = append(want, "name")
			}
			if bits&2 != 0 {
				request.Address = &address
				want = append(want, "address")
			}
			if bits&4 != 0 {
				request.Phone = &phone
				want = append(want, "phone")
			}
			if bits&8 != 0 {
				request.CountryCode = &country
				want = append(want, "country_code")
			}
			ctx := context.WithValue(context.Background(), requestIDKey, "profile-17")
			client := &mocks.ProfileClient{UpdateFunc: func(actual context.Context, payload *profilev1.ProfileRequest) (*profilev1.ProfileResponse, error) {
				if actual != ctx || payload.FieldMask == nil || !reflect.DeepEqual(payload.FieldMask.Paths, want) {
					t.Fatalf("mask/context=%v %v", payload, actual)
				}
				if (payload.CountryCode != nil) != (request.CountryCode != nil) {
					t.Fatal("country presence lost")
				}
				if request.Name != nil && payload.Name != name || request.Name == nil && payload.Name != "" || request.Address != nil && payload.Address != address || request.Address == nil && payload.Address != "" || request.Phone != nil && payload.Phone != phone || request.Phone == nil && payload.Phone != "" {
					t.Fatalf("payload=%v", payload)
				}
				if payload.CountryCode != nil {
					if *payload.CountryCode != country {
						t.Fatal("country value")
					}
					*payload.CountryCode = "modified"
				}
				return &profilev1.ProfileResponse{Changed: changed, Version: 9007199254740993, Profile: &profilev1.Profile{Id: 9, UserId: 17, Name: "legacy NAME", Address: "Legacy\taddress", Phone: "legacy-phone", CountryCode: "legacy-country"}}, nil
			}}
			gateway, err := usersgateway.NewProfile(client)
			if err != nil {
				t.Fatal(err)
			}
			got, err := gateway.UpdateProfile(ctx, request)
			if err != nil || client.Calls != 1 || got.Changed != changed || got.Version == nil || *got.Version != "9007199254740993" || got.Profile == nil || got.Profile.Name != "legacy NAME" || got.Profile.Address != "Legacy\taddress" || got.Profile.Phone != "legacy-phone" || got.Profile.CountryCode != "legacy-country" || got.Profile.ID != 9 || got.Profile.UserID != 17 || country != "+62" {
				t.Fatalf("result=%#v err=%v calls=%d", got, err, client.Calls)
			}
		}
	}
}
func profileGatewayGuards(t *testing.T) {
	if _, err := usersgateway.NewProfile(nil); err == nil {
		t.Fatal("nil client accepted")
	}
	name := "Jane"
	for _, response := range []*profilev1.ProfileResponse{nil, {}, {Version: 1}, {Version: 1, Profile: &profilev1.Profile{UserId: 17}}, {Version: 1, Profile: &profilev1.Profile{Id: 9}}, {Profile: &profilev1.Profile{Id: 9, UserId: 17}}} {
		client := &mocks.ProfileClient{UpdateFunc: func(context.Context, *profilev1.ProfileRequest) (*profilev1.ProfileResponse, error) {
			return response, nil
		}}
		gateway, err := usersgateway.NewProfile(client)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := gateway.UpdateProfile(nil, domain.UpdateProfileRequest{Name: &name}); err == nil {
			t.Fatal("nil ctx accepted")
		}
		if _, err := gateway.UpdateProfile(context.Background(), domain.UpdateProfileRequest{}); !errors.Is(err, common.ErrBadRequest) {
			t.Fatal("empty mask accepted")
		}
		if client.Calls != 0 {
			t.Fatal("guards dispatched")
		}
		got, err := gateway.UpdateProfile(context.Background(), domain.UpdateProfileRequest{Name: &name})
		if !errors.Is(err, common.ErrInternalServerError) || got.Profile != nil || client.Calls != 1 {
			t.Fatalf("result=%#v %v", got, err)
		}
	}
	client := &mocks.ProfileClient{UpdateFunc: func(context.Context, *profilev1.ProfileRequest) (*profilev1.ProfileResponse, error) {
		return nil, errDependency
	}}
	gateway, err := usersgateway.NewProfile(client)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gateway.UpdateProfile(context.Background(), domain.UpdateProfileRequest{Name: &name}); !errors.Is(err, errDependency) || client.Calls != 1 {
		t.Fatalf("dependency error=%v", err)
	}
}
