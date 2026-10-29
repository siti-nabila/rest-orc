package testutils

import (
	"net/http"
	"strconv"
	"testing"

	authdictionary "github.com/siti-nabila/api-contracts/pkg/dictionary/auth"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
)

func PasswordFieldError(t *testing.T, httpStatus int) error {
	t.Helper()
	result, err := status.New(codes.Internal, "private-password-backend-sentinel").WithDetails(
		&errdetails.ErrorInfo{Reason: authdictionary.ErrPasswordMismatch.Key(), Metadata: map[string]string{
			"http_status": strconv.Itoa(httpStatus), "code": "PASSWORD_REJECTED",
		}},
		&errdetails.BadRequest{FieldViolations: []*errdetails.BadRequest_FieldViolation{
			{Field: "current_password", Description: "Password saat ini tidak sesuai."},
		}},
		&errdetails.RetryInfo{RetryDelay: &durationpb.Duration{Seconds: 1, Nanos: 200000000}},
	)
	if err != nil {
		t.Fatal(err)
	}
	return result.Err()
}

func AssertPasswordErrorResponse(t *testing.T, response *http.Response, body string, wantStatus int) {
	t.Helper()
	if response.StatusCode != wantStatus || response.Header.Get("Retry-After") != "2" || body != `{"code":"PASSWORD_REJECTED","errors":{"current_password":["Password saat ini tidak sesuai."]},"data":[]}` {
		t.Fatal("downstream password status, public fields, or retry guidance changed")
	}
}
