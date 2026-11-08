package test_scenarios

import (
	"context"
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

type creationClient func(context.Context, *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error)

func (f creationClient) CreateUser(c context.Context, r *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
	return f(c, r)
}
func UserCreationScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "creation exact SDK payload snapshot validation and isolation", Run: creationMapping}}
}
func creationMapping(t *testing.T) {
	if _, err := gateway.NewUserCreation(nil); err == nil {
		t.Fatal("nil client")
	}
	email, password, name, phone, country, initial, address := " User@Example.COM ", "simple", "NASA O'Neil", "0628123456789", "+62", "PENDING", " X\r\nY "
	req := domain.CreateUserRequest{IdempotencyKey: "opaque, key", Body: domain.CreateUserBody{Email: &email, Password: &password, Name: &name, Phone: &phone, CountryCode: &country, Status: &initial}}
	md := metadata.Pairs("version", "old", "idempotency-key", "old", "idempotency-key", "older", "authorization", "Bearer caller", "x-request-id", "create")
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	calls := 0
	var failure error
	fresh := func() *userv1.CreateUserResponse {
		u := shared.UserDetailResponse(23).User
		u.Email = "user@example.com"
		u.Status = "PENDING"
		return &userv1.CreateUserResponse{User: u}
	}
	result := fresh()
	sdk := creationClient(func(c context.Context, p *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
		calls++
		actual, _ := metadata.FromOutgoingContext(c)
		want := md.Copy()
		want.Delete("version")
		want.Set("idempotency-key", req.IdempotencyKey)
		if !reflect.DeepEqual(actual, want) || p.Email != email || p.Password != password || p.Name != name || p.Phone != phone || p.CountryCode == nil || *p.CountryCode != country || p.CountryCode == &country || p.Status == nil || *p.Status != "PENDING" || p.Status == &initial {
			t.Fatal("payload metadata")
		}
		wantAddress := ""
		if req.Body.Address != nil {
			wantAddress = *req.Body.Address
		}
		if p.Address != wantAddress {
			t.Fatal("address")
		}
		*p.CountryCode = "mutation"
		*p.Status = "ACTIVE"
		actual.Set("authorization", "mutation")
		return result, failure
	})
	g, _ := gateway.NewUserCreation(sdk)
	if _, err := g.CreateUser(nil, req); err == nil {
		t.Fatal("nil context")
	}
	for _, mode := range []string{"key", "country", "empty country", "status", "email", "password", "name", "phone"} {
		invalid := req
		empty := ""
		bad := "ACTIVE"
		switch mode {
		case "key":
			invalid.IdempotencyKey = ""
		case "country":
			invalid.Body.CountryCode = nil
		case "empty country":
			invalid.Body.CountryCode = &empty
		case "status":
			invalid.Body.Status = &bad
		case "email":
			invalid.Body.Email = nil
		case "password":
			invalid.Body.Password = nil
		case "name":
			invalid.Body.Name = nil
		case "phone":
			invalid.Body.Phone = nil
		}
		if _, err := g.CreateUser(ctx, invalid); err == nil || calls != 0 {
			t.Fatal(mode)
		}
	}
	for _, selected := range []bool{false, true} {
		if selected {
			req.Body.Address = &address
		}
		got, err := g.CreateUser(ctx, req)
		if err != nil || got.User.ID != 23 || got.User.Version != "9007199254740993" || got.User.MustChangePassword {
			t.Fatal(got, err)
		}
		got.User.Roles[0].Name = "changed"
		got.User.ApplicationAccess[0].Name = "changed"
		if result.User.Roles[0].Name != "Reader" || result.User.ApplicationAccess[0].Name != "CLM" {
			t.Fatal("alias")
		}
	}
	for _, version := range []uint64{1, 9007199254740993, 9223372036854775807} {
		result = fresh()
		result.User.Version = version
		result.User.Roles = nil
		result.User.ApplicationAccess = nil
		got, err := g.CreateUser(ctx, req)
		if err != nil || got.User.Roles == nil || got.User.ApplicationAccess == nil {
			t.Fatal(got, err)
		}
	}
	for _, mode := range []string{"nil", "user", "zero id", "large id", "zero version", "overflow", "timestamp", "missing timestamp", "email", "canonical email", "status", "nil role", "zero role", "large role", "nil app", "zero app", "large app"} {
		result = fresh()
		switch mode {
		case "nil":
			result = nil
		case "user":
			result.User = nil
		case "zero id":
			result.User.Id = 0
		case "large id":
			result.User.Id = 2147483648
		case "zero version":
			result.User.Version = 0
		case "overflow":
			result.User.Version = 1 << 63
		case "timestamp":
			result.User.CreatedAt = &timestamppb.Timestamp{Nanos: -1}
		case "missing timestamp":
			result.User.CreatedAt = nil
		case "email":
			result.User.Email = "other@example.com"
		case "canonical email":
			result.User.Email = email
		case "status":
			result.User.Status = "ACTIVE"
		case "nil role":
			result.User.Roles[0] = nil
		case "zero role":
			result.User.Roles[0].RoleId = 0
		case "large role":
			result.User.Roles[0].RoleId = 2147483648
		case "nil app":
			result.User.ApplicationAccess[0] = nil
		case "zero app":
			result.User.ApplicationAccess[0].ApplicationId = 0
		case "large app":
			result.User.ApplicationAccess[0].ApplicationId = 2147483648
		}
		got, err := g.CreateUser(ctx, req)
		if !errors.Is(err, common.ErrInternalServerError) || got.User != nil {
			t.Fatal(mode, got, err)
		}
	}
	failure = errors.New("dependency")
	got, err := g.CreateUser(ctx, req)
	if err != failure || got.User != nil {
		t.Fatal(got, err)
	}
	if email != " User@Example.COM " || password != "simple" || country != "+62" || initial != "PENDING" || phone != "0628123456789" || md.Get("authorization")[0] != "Bearer caller" || len(md.Get("idempotency-key")) != 2 {
		t.Fatal("caller mutation")
	}
}
