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

type creationGateway func(context.Context, domain.CreateUserRequest) (domain.CreateUserResponse, error)

func (f creationGateway) CreateUser(c context.Context, r domain.CreateUserRequest) (domain.CreateUserResponse, error) {
	return f(c, r)
}
func UserCreationScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "creation authorization before validation one dispatch and isolated output", Run: creationUsecase}}
}
func creationUsecase(t *testing.T) {
	if _, err := usecase.NewCreateUser(nil); err == nil {
		t.Fatal("nil gateway")
	}
	allowed := domain.UserCreationActor{UserID: 17, Permissions: map[string]bool{domain.UserCreatePermission: true}}
	calls := 0
	ctx := context.Background()
	var failure error
	result := domain.CreateUserResponse{User: &domain.UserDetail{ID: 23, Roles: []domain.UserDetailRole{{Name: "member"}}, ApplicationAccess: []domain.UserDetailApplicationAccess{{Name: "app"}}}}
	g := creationGateway(func(c context.Context, r domain.CreateUserRequest) (domain.CreateUserResponse, error) {
		calls++
		if c != ctx || *r.Body.Name != "NASA O'Neil" || *r.Body.Phone != "628123456789" || *r.Body.Email != " User@Example.COM " || *r.Body.Status != "PENDING" {
			t.Fatal(r)
		}
		*r.Body.Password = "SDK mutation"
		return result, failure
	})
	uc, _ := usecase.NewCreateUser(g)
	for _, tc := range []struct {
		a    domain.UserCreationActor
		c    context.Context
		want error
	}{
		{allowed, nil, common.ErrUnauthorized}, {domain.UserCreationActor{}, ctx, common.ErrUnauthorized},
		{domain.UserCreationActor{UserID: 17}, ctx, common.ErrForbidden},
		{domain.UserCreationActor{UserID: 17, Permissions: map[string]bool{domain.UserCreatePermission: false}}, ctx, common.ErrForbidden},
		{domain.UserCreationActor{UserID: 17, MustChangePassword: true, Permissions: allowed.Permissions}, ctx, common.ErrForbidden},
		{domain.UserCreationActor{UserID: 17, Permissions: map[string]bool{"iam.user.*": true, "iam.user": true, "iam.user.read": true, "iam.user.update": true, "iam.user.status.update": true, "iam.session.revoke": true}}, ctx, common.ErrForbidden},
	} {
		got, err := uc.Execute(tc.c, tc.a, domain.CreateUserRequest{})
		if !errors.Is(err, tc.want) || got.User != nil || calls != 0 {
			t.Fatal(got, err)
		}
	}
	if _, err := uc.Execute(ctx, allowed, domain.CreateUserRequest{IdempotencyKey: "key"}); err == nil || calls != 0 {
		t.Fatal("empty create no-op")
	}
	var body domain.CreateUserBody
	_ = json.Unmarshal([]byte(`{"email":" User@Example.COM ","password":"simple","name":"  NASA  o'neil  ","phone":"62628123456789","country_code":"62"}`), &body)
	r := domain.CreateUserRequest{IdempotencyKey: "key", Body: body}
	got, err := uc.Execute(ctx, allowed, r)
	if err != nil || calls != 1 || got.User.ID != 23 || *body.Password != "simple" {
		t.Fatal(got, err)
	}
	got.User.ID = 99
	got.User.Roles[0].Name = "changed"
	got.User.ApplicationAccess[0].Name = "changed"
	if result.User.ID != 23 || result.User.Roles[0].Name != "member" || result.User.ApplicationAccess[0].Name != "app" {
		t.Fatal("result alias")
	}
	failure = errors.New("dependency")
	if got, err = uc.Execute(ctx, allowed, r); err != failure || got.User != nil || calls != 2 {
		t.Fatal(got, err)
	}
	failure = nil
	result.User = nil
	if got, err = uc.Execute(ctx, allowed, r); !errors.Is(err, common.ErrInternalServerError) || got.User != nil || calls != 3 {
		t.Fatal(got, err)
	}
}
