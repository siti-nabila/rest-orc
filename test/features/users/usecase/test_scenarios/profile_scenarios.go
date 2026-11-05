package test_scenarios

import (
	"context"
	"errors"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	usersusecase "github.com/siti-nabila/rest-orc/internal/features/users/usecase"
	"github.com/siti-nabila/rest-orc/test/features/users/usecase/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"testing"
)

func ProfileScenarios() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "profile guards before dispatch and local no-op", Run: profileGuards},
		{Name: "profile forwards normalized copy and authoritative result", Run: profileDispatch},
		{Name: "profile rejects incomplete or mismatched success", Run: profileInvalidResults},
	}
}
func profileGuards(t *testing.T) {
	name := "!"
	valid := "Jane"
	cases := []struct {
		name    string
		ctx     context.Context
		actor   domain.ProfileActor
		request domain.UpdateProfileRequest
		want    error
		invalid bool
	}{
		{name: "nil context", actor: domain.ProfileActor{UserID: 17}, invalid: true},
		{name: "zero actor", ctx: context.Background(), want: common.ErrUnauthorized},
		{name: "restricted no-op", ctx: context.Background(), actor: domain.ProfileActor{UserID: 17, MustChangePassword: true}, want: common.ErrForbidden},
		{name: "restricted invalid", ctx: context.Background(), actor: domain.ProfileActor{UserID: 17, MustChangePassword: true}, request: domain.UpdateProfileRequest{Name: &name}, want: common.ErrForbidden},
		{name: "restricted valid", ctx: context.Background(), actor: domain.ProfileActor{UserID: 17, MustChangePassword: true}, request: domain.UpdateProfileRequest{Name: &valid}, want: common.ErrForbidden},
		{name: "invalid field", ctx: context.Background(), actor: domain.ProfileActor{UserID: 17}, request: domain.UpdateProfileRequest{Name: &name}, invalid: true},
		{name: "local no-op", ctx: context.Background(), actor: domain.ProfileActor{UserID: 17}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			updater := &mocks.ProfileUpdater{UpdateFunc: func(context.Context, domain.UpdateProfileRequest) (domain.UpdateProfileResponse, error) {
				t.Fatal("unexpected mutation")
				return domain.UpdateProfileResponse{}, nil
			}}
			usecase, err := usersusecase.NewUpdateProfile(updater)
			if err != nil {
				t.Fatal(err)
			}
			got, err := usecase.Execute(tc.ctx, tc.actor, tc.request)
			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("error=%v, want %v", err, tc.want)
			}
			if tc.invalid && err == nil {
				t.Fatal("expected validation/guard error")
			}
			if tc.want == nil && !tc.invalid && (err != nil || got.Changed || got.Profile != nil || got.Version != nil) {
				t.Fatalf("no-op=%#v %v", got, err)
			}
			if updater.Calls != 0 {
				t.Fatalf("calls=%d", updater.Calls)
			}
		})
	}
	if _, err := usersusecase.NewUpdateProfile(nil); err == nil {
		t.Fatal("nil updater accepted")
	}
}
func profileDispatch(t *testing.T) {
	for _, failure := range []error{nil, errGateway} {
		for _, changed := range []bool{true, false} {
			ctx := context.WithValue(context.Background(), requestIDKey, "profile-17")
			name, country, phone, address := "  NASA  o'neil ", "62", "0812345678901", " X\r\nY "
			version := "9007199254740993"
			expected := domain.UpdateProfileResponse{Changed: changed, Version: &version, Profile: &domain.Profile{ID: 9, UserID: 17}}
			updater := &mocks.ProfileUpdater{UpdateFunc: func(actual context.Context, request domain.UpdateProfileRequest) (domain.UpdateProfileResponse, error) {
				if actual != ctx || *request.Name != "NASA O'Neil" || *request.CountryCode != "+62" || *request.Phone != "812345678901" || *request.Address != address {
					t.Fatalf("request/context=%#v %v", request, actual)
				}
				*request.Name = "modified by dependency"
				return expected, failure
			}}
			usecase, err := usersusecase.NewUpdateProfile(updater)
			if err != nil {
				t.Fatal(err)
			}
			got, err := usecase.Execute(ctx, domain.ProfileActor{UserID: 17}, domain.UpdateProfileRequest{Name: &name, CountryCode: &country, Phone: &phone, Address: &address})
			if !errors.Is(err, failure) || failure == nil && (got.Changed != changed || got.Version != expected.Version || got.Profile != expected.Profile) {
				t.Fatalf("result=%#v err=%v", got, err)
			}
			if updater.Calls != 1 || name != "  NASA  o'neil " || country != "62" || phone != "0812345678901" {
				t.Fatal("call count or caller data changed")
			}
		}
	}
}
func profileInvalidResults(t *testing.T) {
	version, zero, bad := "3", "0", "bad"
	for _, result := range []domain.UpdateProfileResponse{
		{}, {Version: &version}, {Version: &version, Profile: &domain.Profile{UserID: 17}},
		{Version: &version, Profile: &domain.Profile{ID: 9, UserID: 18}},
		{Profile: &domain.Profile{ID: 9, UserID: 17}},
		{Version: &zero, Profile: &domain.Profile{ID: 9, UserID: 17}},
		{Version: &bad, Profile: &domain.Profile{ID: 9, UserID: 17}},
	} {
		updater := &mocks.ProfileUpdater{UpdateFunc: func(context.Context, domain.UpdateProfileRequest) (domain.UpdateProfileResponse, error) {
			return result, nil
		}}
		usecase, err := usersusecase.NewUpdateProfile(updater)
		if err != nil {
			t.Fatal(err)
		}
		name := "Jane"
		got, err := usecase.Execute(context.Background(), domain.ProfileActor{UserID: 17}, domain.UpdateProfileRequest{Name: &name})
		if !errors.Is(err, common.ErrInternalServerError) || got.Profile != nil || updater.Calls != 1 {
			t.Fatalf("unsafe result=%#v err=%v calls=%d", got, err, updater.Calls)
		}
	}
}
