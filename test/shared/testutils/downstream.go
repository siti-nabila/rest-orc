package testutils

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/gofiber/fiber/v3"
	authdictionary "github.com/siti-nabila/api-contracts/pkg/dictionary/auth"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
)

func ServiceFieldError(t *testing.T, httpStatus int) error {
	t.Helper()
	s, err := status.New(codes.Internal, "private database password").WithDetails(
		&errdetails.ErrorInfo{Reason: authdictionary.ErrDataExists.Key(), Metadata: map[string]string{
			"http_status": fmt.Sprint(httpStatus), "code": "EMAIL_USED", "private": "hidden",
		}},
		&errdetails.BadRequest{FieldViolations: []*errdetails.BadRequest_FieldViolation{
			{Field: "email", Description: "Email sudah digunakan."},
		}},
		&errdetails.RetryInfo{RetryDelay: &durationpb.Duration{Seconds: 1, Nanos: 200000000}},
	)
	if err != nil {
		t.Fatalf("WithDetails: %v", err)
	}
	return s.Err()
}

func AssertServiceContext(t *testing.T, ctx context.Context) {
	t.Helper()
	md, _ := metadata.FromOutgoingContext(ctx)
	if !reflect.DeepEqual(md.Get(locale.MetadataKey), []string{"id-ID"}) {
		t.Errorf("locale metadata = %v, want [id-ID]", md.Get(locale.MetadataKey))
	}
	if _, ok := ctx.Deadline(); !ok {
		t.Error("service context is missing the client deadline")
	}
}

func AssertServiceErrorResponse(t *testing.T, response *http.Response, body string, wantStatus int) {
	t.Helper()
	if response.StatusCode != wantStatus {
		t.Errorf("status = %d, want %d", response.StatusCode, wantStatus)
	}
	if response.Header.Get(fiber.HeaderRetryAfter) != "2" {
		t.Errorf("Retry-After = %q, want 2", response.Header.Get(fiber.HeaderRetryAfter))
	}
	want := `{"code":"EMAIL_USED","errors":{"email":["Email sudah digunakan."]},"data":[]}`
	if body != want {
		t.Errorf("body = %s, want %s", body, want)
	}
}
