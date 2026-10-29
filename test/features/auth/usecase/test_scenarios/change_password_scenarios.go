package test_scenarios

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/siti-nabila/api-contracts/pkg/dictionary"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	authusecase "github.com/siti-nabila/rest-orc/internal/features/auth/usecase"
	pkgdictionary "github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/test/features/auth/usecase/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func PasswordScenarios() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "password usecase guards identity context and invalid input before mutation", Run: passwordUseCaseGuards},
		{Name: "password usecase forwards raw credentials context flags and dependency cause", Run: passwordUseCaseDispatch},
	}
}

func passwordUseCaseGuards(t *testing.T) {
	if got, err := authusecase.NewChangePassword(nil); got != nil || !errors.Is(err, pkgdictionary.ErrAuthUseCaseGatewayRequired) {
		t.Fatal("nil gateway guard differs")
	}
	valid := domain.ChangePasswordRequest{CurrentPassword: "legacy password", NewPassword: "new-password"}
	for _, tc := range []struct {
		name    string
		ctx     context.Context
		id      uint64
		request domain.ChangePasswordRequest
		want    error
	}{
		{name: "nil context", id: 17, request: valid, want: pkgdictionary.ErrAuthUseCaseContextRequired},
		{name: "zero identity", ctx: context.Background(), request: valid, want: common.ErrUnauthorized},
		{name: "empty credentials", ctx: context.Background(), id: 17},
		{name: "equal credentials", ctx: context.Background(), id: 17, request: domain.ChangePasswordRequest{CurrentPassword: "same-password", NewPassword: "same-password"}},
		{name: "new password character", ctx: context.Background(), id: 17, request: domain.ChangePasswordRequest{CurrentPassword: "legacy password", NewPassword: "new password"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gateway := &mocks.PasswordGateway{}
			usecase, err := authusecase.NewChangePassword(gateway)
			if err != nil {
				t.Fatal(err)
			}
			got, err := usecase.Execute(tc.ctx, tc.id, tc.request)
			if tc.want != nil {
				if !errors.Is(err, tc.want) {
					t.Fatal("guard identity differs")
				}
			} else if fields, ok := errors.AsType[dictionary.FieldErrors](err); !ok || fields.Empty() {
				t.Fatal("missing field-validation error")
			}
			if got != (domain.ChangePasswordResponse{}) || gateway.Calls != 0 {
				t.Fatal("rejected input caused mutation or success")
			}
		})
	}
}

func passwordUseCaseDispatch(t *testing.T) {
	for _, current := range []string{" PaSs ", "ééé", "ab\x00\t\r\n"} {
		for _, changed := range []bool{false, true} {
			for _, loginRequired := range []bool{false, true} {
				for _, failure := range []error{nil, fmt.Errorf("password dependency: %w", errGateway)} {
					ctx := context.WithValue(context.Background(), requestIDKey, "password-request")
					request := domain.ChangePasswordRequest{CurrentPassword: current, NewPassword: "new-password"}
					expected := domain.ChangePasswordResponse{Changed: changed, LoginRequired: loginRequired}
					gateway := &mocks.PasswordGateway{ChangeFunc: func(actual context.Context, input domain.ChangePasswordRequest) (domain.ChangePasswordResponse, error) {
						if actual != ctx || input != request {
							t.Error("raw credential or context changed")
						}
						return expected, failure
					}}
					usecase, err := authusecase.NewChangePassword(gateway)
					if err != nil {
						t.Fatal(err)
					}
					got, err := usecase.Execute(ctx, 17, request)
					if got != expected || err != failure || gateway.Calls != 1 {
						t.Fatal("dependency result, error identity, or call count changed")
					}
					if failure != nil && !errors.Is(err, errGateway) {
						t.Fatal("dependency cause lost")
					}
				}
			}
		}
	}
}
