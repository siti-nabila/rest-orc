package test_scenarios

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	authhandler "github.com/siti-nabila/rest-orc/internal/features/auth/handler"
	"github.com/siti-nabila/rest-orc/test/shared/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

func registerErrorResponse(t *testing.T) {
	for _, httpStatus := range []int{400, 409, 429, 503} {
		t.Run(fmt.Sprint(httpStatus), func(t *testing.T) {
			calls := []string{}
			serviceError := testutils.ServiceFieldError(t, httpStatus)
			service := &mocks.Service{
				MeFunc: func(context.Context, *emptypb.Empty, ...grpc.CallOption) (*userv1.UserData, error) {
					t.Fatal("public register unexpectedly called Me")
					return nil, nil
				},
				RegisterFunc: func(ctx context.Context, req *userv1.AuthRequest, _ ...grpc.CallOption) (*userv1.UserTokenResponse, error) {
					calls = append(calls, "Register")
					testutils.AssertServiceContext(t, ctx)
					if req.GetEmail() != "used@example.com" || req.GetPassword() != "example-password" {
						t.Error("Register arguments differ from HTTP input")
					}
					return nil, fmt.Errorf("private wrapper: %w", serviceError)
				},
			}
			app := newAuthApp(t, service)
			request := httptest.NewRequest(http.MethodPost, apiV1Prefix+authhandler.Route+authhandler.RegisterRoute,
				strings.NewReader(`{"email":"used@example.com","password":"example-password"}`))
			request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			request.Header.Set(locale.HTTPHeader, "id-ID")
			response, err := app.Test(request)
			body := testutils.ResponseBody(t, response, err)
			testutils.AssertServiceErrorResponse(t, response, body, httpStatus)
			if !reflect.DeepEqual(calls, []string{"Register"}) {
				t.Errorf("calls = %v, want [Register] without retry", calls)
			}
		})
	}
}
