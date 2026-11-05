package test_scenarios

import (
	"context"
	"crypto/tls"
	"github.com/gofiber/fiber/v3"
	profilev1 "github.com/siti-nabila/api-contracts/pb/profile/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	usersfixtures "github.com/siti-nabila/rest-orc/test/integration/users/fixtures"
	sharedfixtures "github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

func profileWireOutcomes(t *testing.T) {
	for _, tc := range []struct {
		name, raw                      string
		status                         int
		restricted, missing, unchanged bool
		failure                        error
		mutation                       bool
	}{
		{name: "valid non-admin", raw: `{"name":"  NASA  o'neil ","address":"Jakarta","phone":"0812345678901","country_code":"62","user_id":999,"field_mask":{"paths":["roles"]},"namespace":"iam"}`, status: 200, mutation: true},
		{name: "backend unchanged", raw: `{"name":"NASA O'Neil"}`, status: 200, mutation: true, unchanged: true},
		{name: "no effective fields", raw: `{"name":null,"user_id":999,"field_mask":["address"]}`, status: 200},
		{name: "restricted no-op", raw: `{}`, status: 403, restricted: true},
		{name: "local validation", raw: `{"name":"a1"}`, status: 400},
		{name: "missing bearer", raw: `{"name":"NASA"}`, status: 401, missing: true},
		{name: "backend method denied", raw: `{"name":"NASA"}`, status: 403, mutation: true, failure: status.Error(codes.PermissionDenied, "private-backend-sentinel")},
		{name: "backend dependency unavailable", raw: `{"name":"NASA"}`, status: 503, mutation: true, failure: status.Error(codes.Unavailable, "private-backend-sentinel")},
		{name: "backend explicit conflict", raw: `{"name":"NASA"}`, status: 409, mutation: true, failure: testutils.ServiceFieldError(t, 409)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				cfg := secureConfig(t)
				h := newTransport(t, cfg, defaultServerTLS(t), func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
					if info.FullMethod == userv1.UserService_Me_FullMethodName {
						user := sharedfixtures.MeUserData()
						user.Fullname = "NASA O'Neil"
						user.MustChangePassword = tc.restricted
						return user, nil
					}
					if tc.failure != nil {
						return nil, tc.failure
					}
					result, err := handler(ctx, req)
					if err != nil {
						return nil, err
					}
					result.(*profilev1.ProfileResponse).Changed = !tc.unchanged
					return result, nil
				})
				defer h.close(t)
				parent, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				original := metadata.Pairs("x-request-id", "profile-wire")
				parent = metadata.NewOutgoingContext(parent, original)
				app := sharedfixtures.NewApp(t, h.client, func(ctx fiber.Ctx) error { ctx.SetContext(parent); return ctx.Next() })
				response, body := secureHTTPRequest(t, app, http.MethodPatch, "/api/v1/me/profile", tc.raw, !tc.missing)
				methods := []string{}
				if !tc.missing {
					methods = append(methods, userv1.UserService_Me_FullMethodName)
				}
				if tc.mutation {
					methods = append(methods, profilev1.ProfileService_UpdateProfile_FullMethodName)
				}
				calls, dials := h.snapshot()
				assertHTTPCalls(t, calls, methods, cfg.ServiceToken, !tc.missing)
				wantDials := 1
				if tc.missing {
					wantDials = 0
				}
				if response.StatusCode != tc.status || dials != wantDials {
					t.Fatalf("response=%d %s dials=%d", response.StatusCode, body, dials)
				}
				deadline, _ := parent.Deadline()
				for _, call := range calls {
					if call.tls != tls.VersionTLS13 || call.deadline.IsZero() || call.deadline.After(deadline) {
						t.Fatal("TLS or earlier parent deadline lost")
					}
					assertMetadata(t, call.metadata, "x-request-id", "profile-wire")
				}
				md, _ := metadata.FromOutgoingContext(parent)
				if !reflect.DeepEqual(md, original) {
					t.Fatal("parent metadata mutated")
				}
				if tc.status == 200 {
					want := usersfixtures.ProfileNoOpBody
					if tc.mutation {
						want = usersfixtures.ProfileSuccessBody
					}
					if tc.unchanged {
						want = strings.Replace(want, `"changed":true`, `"changed":false`, 1)
					}
					if body != want {
						t.Fatalf("body=%s want %s", body, want)
					}
				}
				if tc.name == "valid non-admin" {
					country := "+62"
					want := &profilev1.ProfileRequest{Name: "NASA O'Neil", Address: "Jakarta", Phone: "812345678901", CountryCode: &country, FieldMask: &fieldmaskpb.FieldMask{Paths: []string{"name", "address", "phone", "country_code"}}}
					if !proto.Equal(calls[1].request.(*profilev1.ProfileRequest), want) {
						t.Fatalf("wire payload=%v", calls[1].request)
					}
				}
				if tc.status == 409 {
					testutils.AssertServiceErrorResponse(t, response, body, 409)
				}
				assertNoSecrets(t, body, cfg.ServiceToken)
			})
		})
	}
}
