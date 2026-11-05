package test_scenarios

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/gofiber/fiber/v3"
	authdictionary "github.com/siti-nabila/api-contracts/pkg/dictionary/auth"
	commondictionary "github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	appresponse "github.com/siti-nabila/rest-orc/internal/response"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/protoadapt"
)

func explicitDownstreamStatus(t *testing.T) {
	for _, httpStatus := range []int{400, 409, 429, 503, 599} {
		for _, fields := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/fields=%t", httpStatus, fields), func(t *testing.T) {
				err := downstreamError(t, codes.PermissionDenied, authdictionary.ErrNotFound.Key(),
					map[string]string{"http_status": fmt.Sprint(httpStatus), "code": "SERVICE_CODE"}, fields)
				// Wrapping must preserve details without exposing the wrapping message.
				response := errorResponse(t, fmt.Errorf("private dependency: %w", err), "id-ID")
				var want any = map[string]string{"description": authdictionary.ErrNotFound.Message(locale.Indonesian)}
				if fields {
					want = emailMessages()
				}
				assertErrorResponse(t, response, httpStatus, "SERVICE_CODE", want)
			})
		}
	}
}

func invalidDownstreamStatus(t *testing.T) {
	for _, httpStatus := range []string{"", "bad", "399", "600", "0", "409.0", "99999999999999999999999999"} {
		for _, fields := range []bool{false, true} {
			for _, registered := range []bool{false, true} {
				t.Run(fmt.Sprintf("%q/fields=%t/registered=%t", httpStatus, fields, registered), func(t *testing.T) {
					reason := "unregistered.error"
					wantStatus := 503
					wantCode := commondictionary.ErrServiceUnavailable.Code()
					message := commondictionary.ErrServiceUnavailable.Message(locale.English)
					if registered {
						reason = authdictionary.ErrNotFound.Key()
						wantStatus = 404
						wantCode = authdictionary.ErrNotFound.Code()
						message = authdictionary.ErrNotFound.Message(locale.English)
					}
					err := downstreamError(t, codes.Unavailable, reason, map[string]string{"http_status": httpStatus}, fields)
					var want any = map[string]string{"description": message}
					if fields {
						want = emailMessages()
					}
					assertErrorResponse(t, errorResponse(t, err, "en"), wantStatus, wantCode, want)
				})
			}
		}
	}
}

func downstreamCodeAndMessage(t *testing.T) {
	cases := []struct {
		name, reason, publicCode, wantCode string
		wantStatus                         int
		fields                             bool
	}{
		{"unknown with public code", "unknown.reason", "EMAIL_EXISTS", "EMAIL_EXISTS", 409, false},
		{"unknown without public code", "unknown.reason", "", "IS", 409, false},
		{"unknown with fields", "unknown.reason", "EMAIL_EXISTS", "EMAIL_EXISTS", 409, true},
		{"registered code fallback", authdictionary.ErrDataExists.Key(), "", "EX", 409, false},
		{"public code overrides registry", authdictionary.ErrDataExists.Key(), "PUBLIC", "PUBLIC", 409, false},
		{"internal description", commondictionary.ErrInternalServerError.Key(), "IS", "IS", 500, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := downstreamError(t, codes.Internal, tc.reason,
				map[string]string{"http_status": fmt.Sprint(tc.wantStatus), "code": tc.publicCode, "secret": "hidden"}, tc.fields)
			var want any = map[string]string{"description": commondictionary.ErrInternalServerError.Message(locale.English)}
			if tc.reason == authdictionary.ErrDataExists.Key() {
				want = map[string]string{"description": authdictionary.ErrDataExists.Message(locale.English)}
			}
			if tc.fields {
				want = emailMessages()
			}
			assertErrorResponse(t, errorResponse(t, err, "en"), tc.wantStatus, tc.wantCode, want)
		})
	}
}

func grpcFallbackStatuses(t *testing.T) {
	cases := []struct {
		code    codes.Code
		status  int
		public  string
		message string
	}{
		{codes.InvalidArgument, 400, "BR", "Bad request."},
		{codes.Unauthenticated, 401, "UA", commondictionary.ErrUnauthorized.Message(locale.English)},
		{codes.PermissionDenied, 403, commondictionary.ErrForbidden.Code(), commondictionary.ErrForbidden.Message(locale.English)},
		{codes.NotFound, 404, "NF", "Data not found."},
		{codes.AlreadyExists, 409, "IS", "Internal server error."},
		{codes.Aborted, 409, "IS", "Internal server error."},
		{codes.ResourceExhausted, 429, "RL", commondictionary.ErrRateLimited.Message(locale.English)},
		{codes.FailedPrecondition, 400, "BR", "Bad request."},
		{codes.Unavailable, 503, commondictionary.ErrServiceUnavailable.Code(), commondictionary.ErrServiceUnavailable.Message(locale.English)},
		{codes.DeadlineExceeded, 504, "TO", "Request timeout."},
		{codes.Canceled, 499, "RC", commondictionary.ErrRequestCanceled.Message(locale.English)},
		{codes.Unknown, 500, "IS", "Internal server error."},
		{codes.Internal, 500, "IS", "Internal server error."},
		{codes.DataLoss, 500, "IS", "Internal server error."},
		{codes.Unimplemented, 500, "IS", "Internal server error."},
		{codes.Code(100), 500, "IS", "Internal server error."},
	}
	for _, tc := range cases {
		for _, fields := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/fields=%t", tc.code, fields), func(t *testing.T) {
				err := status.Error(tc.code, "private transport credentials")
				if fields {
					err = downstreamError(t, tc.code, "", nil, true)
				}
				var want any = map[string]string{"description": tc.message}
				if fields {
					want = emailMessages()
				}
				assertErrorResponse(t, errorResponse(t, err, "en"), tc.status, tc.public, want)
			})
		}
	}
}

func contextErrors(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
		code   string
		text   string
	}{
		{context.DeadlineExceeded, 504, "TO", "Request timeout."},
		{context.Canceled, 499, "RC", "Request canceled."},
	} {
		for _, wrapped := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/wrapped=%t", tc.code, wrapped), func(t *testing.T) {
				err := tc.err
				if wrapped {
					err = fmt.Errorf("private wrapper: %w", err)
				}
				assertErrorResponse(t, errorResponse(t, err, "en"), tc.status, tc.code, map[string]string{"description": tc.text})
			})
		}
	}
	t.Run("structured status precedes native cancellation", func(t *testing.T) {
		structured := downstreamError(t, codes.Canceled, "unknown.reason", map[string]string{"http_status": "503", "code": "PUBLIC"}, false)
		err := errors.Join(context.Canceled, structured)
		assertErrorResponse(t, errorResponse(t, err, "en"), 503, "PUBLIC", map[string]string{
			"description": commondictionary.ErrRequestCanceled.Message(locale.English),
		})
	})
}

func downstreamError(t *testing.T, code codes.Code, reason string, metadata map[string]string, fields bool) error {
	t.Helper()
	details := []protoadapt.MessageV1{&errdetails.ErrorInfo{Reason: reason, Metadata: metadata}}
	if fields {
		details = append(details, &errdetails.BadRequest{FieldViolations: []*errdetails.BadRequest_FieldViolation{
			{Field: "email", Description: "Email sudah digunakan."},
			{Field: "email", Description: "Pilih email lain."},
		}})
	}
	return grpcError(t, code, details...)
}

func grpcError(t *testing.T, code codes.Code, details ...protoadapt.MessageV1) error {
	t.Helper()
	s, err := status.New(code, "private database password").WithDetails(details...)
	if err != nil {
		t.Fatalf("WithDetails: %v", err)
	}
	return s.Err()
}

func emailMessages() map[string][]string {
	return map[string][]string{"email": {"Email sudah digunakan.", "Pilih email lain."}}
}

func errorResponse(t *testing.T, err error, language string) *http.Response {
	t.Helper()
	app := fiber.New()
	writer := newWriter()
	app.Get("/error", func(ctx fiber.Ctx) error {
		return writer.Write(ctx, appresponse.Result{Err: err})
	})
	return performRequest(t, app, "/error", language)
}

func assertErrorResponse(t *testing.T, response *http.Response, wantStatus int, wantCode string, wantErrors any) {
	t.Helper()
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close response: %v", err)
		}
	}()
	if response.StatusCode != wantStatus {
		t.Errorf("status = %d, want %d", response.StatusCode, wantStatus)
	}
	var got map[string]any
	if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	wantJSON, err := json.Marshal(map[string]any{"code": wantCode, "errors": wantErrors, "data": []any{}})
	if err != nil {
		t.Fatalf("encode expected body: %v", err)
	}
	var want map[string]any
	if err := json.Unmarshal(wantJSON, &want); err != nil {
		t.Fatalf("decode expected body: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("body = %#v, want %#v", got, want)
	}
}
