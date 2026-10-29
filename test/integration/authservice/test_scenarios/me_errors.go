package test_scenarios

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"github.com/gofiber/fiber/v3"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	authdictionary "github.com/siti-nabila/api-contracts/pkg/dictionary/auth"
	commondictionary "github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	sharedfixtures "github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func meHTTPErrors(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		rpc                        codes.Code
		reason, code, description  string
		httpStatus                 int
		explicit, invalidTimestamp bool
	}{
		{name: "invalid user", rpc: codes.Unauthenticated, reason: commondictionary.ErrUnauthorized.Key(), code: commondictionary.ErrUnauthorized.Code(), description: commondictionary.ErrUnauthorized.Message(locale.Parse("id")), httpStatus: 401},
		{name: "invalid service credential", rpc: codes.Unauthenticated, reason: authdictionary.ErrServiceAuthenticationFailed.Key(), code: authdictionary.ErrServiceAuthenticationFailed.Code(), description: authdictionary.ErrServiceAuthenticationFailed.Message(locale.Parse("id")), httpStatus: 401},
		{name: "service authority denied", rpc: codes.PermissionDenied, reason: commondictionary.ErrForbidden.Key(), code: commondictionary.ErrForbidden.Code(), description: commondictionary.ErrForbidden.Message(locale.Parse("id")), httpStatus: 403},
		{name: "snapshot dependency failed", rpc: codes.Unavailable, reason: commondictionary.ErrServiceUnavailable.Key(), code: commondictionary.ErrServiceUnavailable.Code(), description: commondictionary.ErrServiceUnavailable.Message(locale.Parse("id")), httpStatus: 503},
		{name: "explicit conflict and retry", rpc: codes.Internal, code: "ME_CONFLICT", httpStatus: 409, explicit: true},
		{name: "unknown private failure", rpc: codes.Unknown, code: commondictionary.ErrInternalServerError.Code(), description: commondictionary.ErrInternalServerError.Message(locale.Parse("id")), httpStatus: 500},
		{name: "invalid snapshot timestamp", invalidTimestamp: true, code: commondictionary.ErrInternalServerError.Code(), description: commondictionary.ErrInternalServerError.Message(locale.Parse("id")), httpStatus: 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				s := status.New(tc.rpc, "private-backend-sentinel")
				var err error
				if tc.reason != "" {
					s, err = s.WithDetails(&errdetails.ErrorInfo{Reason: tc.reason})
					if err != nil {
						t.Fatal(err)
					}
				}
				fields := any(map[string]string{"description": tc.description})
				if tc.explicit {
					s, err = s.WithDetails(
						&errdetails.ErrorInfo{Metadata: map[string]string{"http_status": strconv.Itoa(tc.httpStatus), "code": tc.code}},
						&errdetails.BadRequest{FieldViolations: []*errdetails.BadRequest_FieldViolation{{Field: "profile", Description: "Snapshot tidak tersedia."}}},
						&errdetails.RetryInfo{RetryDelay: &durationpb.Duration{Seconds: 1, Nanos: 200000000}},
					)
					if err != nil {
						t.Fatal(err)
					}
					fields = map[string][]string{"profile": {"Snapshot tidak tersedia."}}
				}
				var sequence atomic.Int32
				cfg := secureConfig(t)
				h := newTransport(t, cfg, defaultServerTLS(t), func(context.Context, any, *grpc.UnaryServerInfo, grpc.UnaryHandler) (any, error) {
					if sequence.Add(1) == 1 {
						return sharedfixtures.MeUserData(), nil
					}
					if tc.invalidTimestamp {
						return &userv1.UserData{CreatedAt: &timestamppb.Timestamp{Seconds: 253402300800}}, nil
					}
					return nil, s.Err()
				})
				defer h.close(t)
				app := sharedfixtures.NewApp(t, h.client)
				first, body := secureHTTPRequest(t, app, http.MethodGet, "/api/v1/me", "", true)
				readMeData(t, first, body)
				request := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
				request.Header.Set(fiber.HeaderAuthorization, userBearer)
				request.Header.Set(locale.HTTPHeader, "id-ID")
				response, body := meHTTPRequest(t, app, request)
				want, _ := json.Marshal(map[string]any{"code": tc.code, "errors": fields, "data": []any{}})
				var gotJSON, wantJSON any
				if err := json.Unmarshal([]byte(body), &gotJSON); err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal(want, &wantJSON); err != nil {
					t.Fatal(err)
				}
				if response.StatusCode != tc.httpStatus || !reflect.DeepEqual(gotJSON, wantJSON) {
					t.Errorf("error response=%d/%s, want %d/%s", response.StatusCode, body, tc.httpStatus, want)
				}
				wantRetry := ""
				if tc.explicit {
					wantRetry = "2"
				}
				if response.Header.Get(fiber.HeaderRetryAfter) != wantRetry {
					t.Error("retry guidance changed")
				}
				calls, _ := h.snapshot()
				if len(calls) != 2 {
					t.Fatalf("RPC calls=%d, want one per HTTP request", len(calls))
				}
				for _, c := range calls {
					if c.method != userv1.UserService_Me_FullMethodName {
						t.Error("unexpected aggregation RPC")
					}
				}
				assertMetadata(t, calls[1].metadata, locale.MetadataKey, "id-ID")
				assertNoSecrets(t, body, cfg.ServiceToken)
			})
		})
	}
}

func meHTTPRequest(t *testing.T, app *fiber.App, request *http.Request) (*http.Response, string) {
	t.Helper()
	// Leave timeout ownership to the request/client context under synctest.
	response, err := app.Test(request, fiber.TestConfig{})
	return response, testutils.ResponseBody(t, response, err)
}
