package test_scenarios

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gofiber/fiber/v3"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	usershandler "github.com/siti-nabila/rest-orc/internal/features/users/handler"
	"github.com/siti-nabila/rest-orc/test/integration/users/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

func protectedErrorResponses(t *testing.T) {
	for _, tc := range []struct {
		name, query string
		meError     bool
		status      int
		calls       []string
	}{
		{"business error", "?page=2", false, 503, []string{"Me", "ListUsers"}},
		{"authentication error", "?page=2", true, 503, []string{"Me"}},
		{"local validation", "?page=invalid", false, 400, []string{"Me"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := []string{}
			serviceError := testutils.ServiceFieldError(t, 503)
			service := &mocks.Service{
				MeFunc: func(ctx context.Context, _ *emptypb.Empty, _ ...grpc.CallOption) (*userv1.UserData, error) {
					calls = append(calls, "Me")
					assertAuthorizationMetadata(t, ctx)
					testutils.AssertServiceContext(t, ctx)
					if tc.meError {
						return nil, serviceError
					}
					return fixtures.MeResponse(), nil
				},
				ListUsersFunc: func(ctx context.Context, req *userv1.ListUsersRequest, _ ...grpc.CallOption) (*userv1.ListUsersResponse, error) {
					calls = append(calls, "ListUsers")
					assertAuthorizationMetadata(t, ctx)
					testutils.AssertServiceContext(t, ctx)
					if req.GetQuery().GetPage() != 2 {
						t.Error("ListUsers page differs from HTTP query")
					}
					return nil, serviceError
				},
			}
			app := newUsersApp(t, service)
			request := httptest.NewRequest(http.MethodGet, apiV1Prefix+usershandler.Route+tc.query, nil)
			request.Header.Set(fiber.HeaderAuthorization, authorizationValue)
			request.Header.Set(locale.HTTPHeader, "id-ID")
			response, err := app.Test(request)
			body := testutils.ResponseBody(t, response, err)
			if tc.name == "local validation" {
				if response.StatusCode != 400 || response.Header.Get(fiber.HeaderRetryAfter) != "" {
					t.Errorf("local validation response = %d/%v, want 400 without Retry-After", response.StatusCode, response.Header)
				}
				if body != `{"code":"BR","errors":{"description":"Permintaan tidak valid."},"data":[]}` {
					t.Errorf("local validation body = %s", body)
				}
			} else {
				testutils.AssertServiceErrorResponse(t, response, body, tc.status)
			}
			if !reflect.DeepEqual(calls, tc.calls) {
				t.Errorf("calls = %v, want %v without retry", calls, tc.calls)
			}
		})
	}
}
