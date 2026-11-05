package test_scenarios

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/siti-nabila/api-contracts/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	authusecase "github.com/siti-nabila/rest-orc/internal/features/auth/usecase"
	pkgdictionary "github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/test/features/auth/usecase/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

type contextKey string

const requestIDKey contextKey = "request-id"

var errGateway = errors.New("auth gateway failed")

func All() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "auth usecases reject invalid input before gateway", Run: authRejectsInvalid},
		{Name: "auth usecases forward exact request context and gateway results", Run: authForwardsValid},
		{Name: "auth usecases preserve constructor and nil context guards", Run: authGuards},
	}
}

type authExecute func(context.Context, domain.AuthRequest) (any, error)

func newAuthExecute(t *testing.T, mode string, gateway *mocks.Gateway) authExecute {
	t.Helper()
	if mode == "register" {
		uc, err := authusecase.NewRegister(gateway)
		if err != nil {
			t.Fatal(err)
		}
		return func(ctx context.Context, req domain.AuthRequest) (any, error) { return uc.Execute(ctx, req) }
	}
	uc, err := authusecase.NewLogin(gateway)
	if err != nil {
		t.Fatal(err)
	}
	return func(ctx context.Context, req domain.AuthRequest) (any, error) { return uc.Execute(ctx, req) }
}

func authRejectsInvalid(t *testing.T) {
	for _, mode := range []string{"register", "login"} {
		requests := []domain.AuthRequest{
			{},
			{Email: "bad", Password: "short"},
			{Email: "user+tag@example.com", Password: "valid-password"},
			{Email: "a@b.co", Password: strings.Repeat("a", 51)},
		}
		if mode == "register" {
			requests = append(requests, domain.AuthRequest{Email: "a@b.co", Password: "ééé"})
		}
		t.Run(mode, func(t *testing.T) {
			for _, req := range requests {
				gateway := &mocks.Gateway{}
				got, err := newAuthExecute(t, mode, gateway)(context.Background(), req)
				if fields, ok := errors.AsType[dictionary.FieldErrors](err); !ok || fields.Empty() {
					t.Errorf("invalid input error type = %T, want nonempty FieldErrors", err)
				}
				if !reflect.ValueOf(got).IsZero() {
					t.Error("invalid input returned a nonzero response")
				}
				assertAuthCalls(t, gateway, 0, 0)
			}
		})
	}
}

func authForwardsValid(t *testing.T) {
	for _, mode := range []string{"register", "login"} {
		passwords := []string{"aaaaaa", strings.Repeat("~", 50)}
		if mode == "login" {
			passwords = append(passwords, "ééé", "      ", "ab\x00\t\r\n")
		}
		for _, password := range passwords {
			for _, dependencyError := range []error{nil, errGateway} {
				t.Run(mode, func(t *testing.T) {
					ctx := context.WithValue(context.Background(), requestIDKey, "auth-request")
					req := domain.AuthRequest{Email: " User.Name@Example.COM ", Password: password}
					var want any = domain.RegisterResponse{RegistrationID: 17, RegistrationStatus: "PENDING"}
					if mode == "login" {
						want = domain.SessionResult{Token: "gateway-result"}
					}
					call := func(actual context.Context, input domain.AuthRequest) (any, error) {
						if actual != ctx || input != req {
							t.Error("gateway received different context or credential")
						}
						return want, dependencyError
					}
					gateway := &mocks.Gateway{}
					registerCalls, loginCalls := 0, 0
					if mode == "register" {
						gateway.RegisterFunc = func(c context.Context, r domain.AuthRequest) (domain.RegisterResponse, error) {
							v, e := call(c, r)
							return v.(domain.RegisterResponse), e
						}
						registerCalls = 1
					} else {
						gateway.LoginFunc = func(c context.Context, r domain.AuthRequest) (domain.SessionResult, error) {
							v, e := call(c, r)
							return v.(domain.SessionResult), e
						}
						loginCalls = 1
					}
					got, err := newAuthExecute(t, mode, gateway)(ctx, req)
					if got != want || err != dependencyError {
						t.Error("gateway result or error identity changed")
					}
					assertAuthCalls(t, gateway, registerCalls, loginCalls)
				})
			}
		}
	}
}

func authGuards(t *testing.T) {
	register, err := authusecase.NewRegister(nil)
	if register != nil || !errors.Is(err, pkgdictionary.ErrUsersUseCaseGatewayRequired) {
		t.Error("register constructor did not preserve nil dependency error")
	}
	login, err := authusecase.NewLogin(nil)
	if login != nil || !errors.Is(err, pkgdictionary.ErrUsersUseCaseGatewayRequired) {
		t.Error("login constructor did not preserve nil dependency error")
	}
	for _, mode := range []string{"register", "login"} {
		gateway := &mocks.Gateway{}
		got, err := newAuthExecute(t, mode, gateway)(nil, domain.AuthRequest{})
		if !reflect.ValueOf(got).IsZero() || !errors.Is(err, pkgdictionary.ErrUsersUseCaseContextRequired) {
			t.Error("nil context guard must run before input validation")
		}
		assertAuthCalls(t, gateway, 0, 0)
	}
}

func assertAuthCalls(t *testing.T, gateway *mocks.Gateway, register, login int) {
	t.Helper()
	if gateway.RegisterCalls != register || gateway.LoginCalls != login {
		t.Errorf("gateway calls register/login = %d/%d, want %d/%d",
			gateway.RegisterCalls, gateway.LoginCalls, register, login)
	}
}
