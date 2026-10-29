package test_scenarios

import (
	"fmt"
	"math"
	"testing"

	"github.com/gofiber/fiber/v3"
	commondictionary "github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/protoadapt"
	"google.golang.org/protobuf/types/known/durationpb"
)

func retryDelayHeaders(t *testing.T) {
	cases := []struct {
		name  string
		info  *errdetails.RetryInfo
		value string
	}{
		{"absent", nil, ""},
		{"missing duration", &errdetails.RetryInfo{}, ""},
		{"zero", retryDelay(0, 0), "0"},
		{"whole seconds", retryDelay(2, 0), "2"},
		{"round up", retryDelay(1, 200000000), "2"},
		{"smallest fraction", retryDelay(0, 1), "1"},
		{"large valid seconds", retryDelay(315576000000, 0), "315576000000"},
		{"large valid fraction", retryDelay(315576000000, 999999999), "315576000001"},
		{"negative seconds", retryDelay(-1, 0), ""},
		{"negative fraction", retryDelay(0, -1), ""},
		{"invalid signs", retryDelay(1, -1), ""},
		{"invalid nanos", retryDelay(0, 1000000000), ""},
		{"outside protobuf range", retryDelay(315576000001, 0), ""},
		{"overflow input", retryDelay(math.MaxInt64, 1), ""},
	}
	for _, tc := range cases {
		for _, fields := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/fields=%t", tc.name, fields), func(t *testing.T) {
				details := []protoadapt.MessageV1{&errdetails.ErrorInfo{
					Reason:   commondictionary.ErrRateLimited.Key(),
					Metadata: map[string]string{"http_status": "429", "code": "RL"},
				}}
				if tc.info != nil {
					details = append(details, tc.info)
				}
				var want any = map[string]string{"description": "Too many requests. Try again later."}
				if fields {
					details = append(details, &errdetails.BadRequest{FieldViolations: []*errdetails.BadRequest_FieldViolation{
						{Field: "email", Description: "Try again later."},
					}})
					want = map[string][]string{"email": {"Try again later."}}
				}
				response := errorResponse(t, grpcError(t, codes.Internal, details...), "en")
				if got := response.Header.Get(fiber.HeaderRetryAfter); got != tc.value {
					t.Errorf("Retry-After = %q, want %q", got, tc.value)
				}
				assertErrorResponse(t, response, 429, "RL", want)
			})
		}
	}
}

func retryDelay(seconds int64, nanos int32) *errdetails.RetryInfo {
	return &errdetails.RetryInfo{RetryDelay: &durationpb.Duration{Seconds: seconds, Nanos: nanos}}
}
