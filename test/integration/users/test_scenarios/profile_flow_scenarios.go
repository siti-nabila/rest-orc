package test_scenarios

import (
	"context"
	"encoding/json"
	"github.com/gofiber/fiber/v3"
	profilev1 "github.com/siti-nabila/api-contracts/pb/profile/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	authservice "github.com/siti-nabila/rest-orc/internal/client/authservice"
	"github.com/siti-nabila/rest-orc/test/integration/users/fixtures"
	profilemocks "github.com/siti-nabila/rest-orc/test/integration/users/mocks"
	sharedfixtures "github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func ProfileScenarios() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "profile HTTP normalization presence eligibility exact results and ordered calls", Run: profileHTTPInputs},
		{Name: "profile HTTP localized errors aggregate without leaking input", Run: profileHTTPLocales},
		{Name: "profile HTTP invalid or mismatched dependency result fails closed", Run: profileHTTPInvalidResults},
		{Name: "profile HTTP preserves downstream status retry and authentication failures", Run: profileHTTPFailures},
	}
}
func profileHTTPInputs(t *testing.T) {
	country := "+62"
	for _, tc := range []struct {
		name, raw                      string
		status                         int
		restricted, missing, unchanged bool
		want                           *profilev1.ProfileRequest
	}{
		{name: "all fields and forged claims", raw: `{"name":"  NASA  o'neil ","phone":"0812345678901","country_code":"62","address":"Jakarta","user_id":999,"id":888,"field_mask":{"paths":["permissions"]},"roles":["admin"],"permissions":{"all":true},"namespace":"iam","version":"99","service_token":"forged"}`, status: 200, want: &profilev1.ProfileRequest{Name: "NASA O'Neil", Address: "Jakarta", Phone: "812345678901", CountryCode: &country, FieldMask: &fieldmaskpb.FieldMask{Paths: []string{"name", "address", "phone", "country_code"}}}},
		{name: "mixed subset", raw: `{"name":"NASA O'Neil","phone":null,"extra":"ignored"}`, status: 200, want: &profilev1.ProfileRequest{Name: "NASA O'Neil", FieldMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}}}},
		{name: "phone without country does not infer Me country", raw: `{"phone":"0812345678901"}`, status: 200, want: &profilev1.ProfileRequest{Phone: "0812345678901", FieldMask: &fieldmaskpb.FieldMask{Paths: []string{"phone"}}}},
		{name: "country only", raw: `{"country_code":"62"}`, status: 200, want: &profilev1.ProfileRequest{CountryCode: &country, FieldMask: &fieldmaskpb.FieldMask{Paths: []string{"country_code"}}}},
		{name: "address exact", raw: `{"address":" A\r\nB "}`, status: 200, want: &profilev1.ProfileRequest{Address: " A\r\nB ", FieldMask: &fieldmaskpb.FieldMask{Paths: []string{"address"}}}},
		{name: "same value still dispatches", raw: `{"name":"NASA O'Neil"}`, status: 200, unchanged: true, want: &profilev1.ProfileRequest{Name: "NASA O'Neil", FieldMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}}}},
		{name: "empty object", raw: `{}`, status: 200}, {name: "JSON null", raw: `null`, status: 200}, {name: "unknown only", raw: `{"field_mask":{"paths":["name"]},"user_id":999}`, status: 200}, {name: "all null", raw: `{"name":null,"phone":null,"country_code":null,"address":null}`, status: 200},
		{name: "empty name", raw: `{"name":""}`, status: 400}, {name: "empty phone", raw: `{"phone":""}`, status: 400}, {name: "empty country", raw: `{"country_code":""}`, status: 400}, {name: "empty address", raw: `{"address":""}`, status: 400},
		{name: "phone length after normalization", raw: `{"phone":"081234567890","country_code":"+62"}`, status: 400},
		{name: "name number", raw: `{"name":17}`, status: 400}, {name: "phone boolean", raw: `{"phone":false}`, status: 400}, {name: "country array", raw: `{"country_code":[]}`, status: 400}, {name: "address object", raw: `{"address":{}}`, status: 400}, {name: "malformed", raw: `{`, status: 400}, {name: "missing body", status: 400},
		{name: "restricted no-op", raw: `{}`, status: 403, restricted: true}, {name: "restricted invalid", raw: `{"name":""}`, status: 403, restricted: true}, {name: "restricted selected", raw: `{"name":"NASA"}`, status: 403, restricted: true},
		{name: "missing bearer", raw: `{"name":"NASA"}`, status: 401, missing: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := []string{}
			service := &mocks.Service{MeFunc: func(ctx context.Context, _ *emptypb.Empty, _ ...grpc.CallOption) (*userv1.UserData, error) {
				calls = append(calls, "Me")
				assertAuthorizationMetadata(t, ctx)
				me := sharedfixtures.MeUserData()
				me.Fullname = "NASA O'Neil"
				me.MustChangePassword = tc.restricted
				return me, nil
			}}
			profile := &profilemocks.ProfileService{UpdateFunc: func(ctx context.Context, request *profilev1.ProfileRequest) (*profilev1.ProfileResponse, error) {
				calls = append(calls, "UpdateProfile")
				assertAuthorizationMetadata(t, ctx)
				if !proto.Equal(request, tc.want) {
					t.Errorf("request=%v want %v", request, tc.want)
				}
				md, _ := metadata.FromOutgoingContext(ctx)
				for _, key := range []string{"namespace", "user_id", "x-service-token", "x-service-name"} {
					if len(md.Get(key)) != 0 {
						t.Errorf("browser claim forwarded: %s", key)
					}
				}
				result := fixtures.ProfileResponse()
				result.Changed = !tc.unchanged
				return result, nil
			}}
			app := newProfileApp(t, service, profile)
			response, body := profileHTTPRequest(t, app, tc.raw, "en", !tc.missing)
			expectedCalls := []string{}
			if !tc.missing {
				expectedCalls = append(expectedCalls, "Me")
			}
			if tc.want != nil {
				expectedCalls = append(expectedCalls, "UpdateProfile")
			}
			if response.StatusCode != tc.status || !reflect.DeepEqual(calls, expectedCalls) {
				t.Fatalf("response=%d %s calls=%v want %v", response.StatusCode, body, calls, expectedCalls)
			}
			if tc.status == 200 {
				want := fixtures.ProfileNoOpBody
				if tc.want != nil {
					want = fixtures.ProfileSuccessBody
				}
				if tc.unchanged {
					want = strings.Replace(want, `"changed":true`, `"changed":false`, 1)
				}
				if body != want {
					t.Fatalf("body=%s want %s", body, want)
				}
			} else if !strings.Contains(body, `"data":[]`) || strings.Contains(body, "forged") {
				t.Fatalf("unsafe error=%s", body)
			}
		})
	}
}
func profileHTTPLocales(t *testing.T) {
	for _, tc := range []struct{ language, word string }{{"en", "invalid characters"}, {"id", "karakter tidak valid"}, {"zh-CN", "无效字符"}} {
		t.Run(tc.language, func(t *testing.T) {
			meCalls, profileCalls := 0, 0
			app := newProfileApp(t, &mocks.Service{MeFunc: func(ctx context.Context, _ *emptypb.Empty, _ ...grpc.CallOption) (*userv1.UserData, error) {
				meCalls++
				return sharedfixtures.MeUserData(), nil
			}}, &profilemocks.ProfileService{UpdateFunc: func(context.Context, *profilev1.ProfileRequest) (*profilev1.ProfileResponse, error) {
				profileCalls++
				return fixtures.ProfileResponse(), nil
			}})
			response, body := profileHTTPRequest(t, app, `{"name":"A1-secret","phone":"not-phone-secret","country_code":"oops-secret","address":"秘密-secret"}`, tc.language, true)
			var result struct {
				Code   string
				Errors map[string][]string
				Data   []any
			}
			if err := json.Unmarshal([]byte(body), &result); err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != 400 || result.Code != "BR" || len(result.Errors) != 4 || len(result.Data) != 0 || !strings.Contains(body, tc.word) || strings.Contains(body, "secret") || meCalls != 1 || profileCalls != 0 {
				t.Fatalf("response=%d %s calls=%d/%d", response.StatusCode, body, meCalls, profileCalls)
			}
		})
	}
}
func profileHTTPInvalidResults(t *testing.T) {
	for _, kind := range []string{"nil response", "nil profile", "zero id", "zero owner", "zero version", "wrong owner"} {
		t.Run(kind, func(t *testing.T) {
			calls := []string{}
			app := newProfileApp(t, &mocks.Service{MeFunc: func(context.Context, *emptypb.Empty, ...grpc.CallOption) (*userv1.UserData, error) {
				calls = append(calls, "Me")
				return sharedfixtures.MeUserData(), nil
			}}, &profilemocks.ProfileService{UpdateFunc: func(context.Context, *profilev1.ProfileRequest) (*profilev1.ProfileResponse, error) {
				calls = append(calls, "UpdateProfile")
				result := fixtures.ProfileResponse()
				switch kind {
				case "nil response":
					return nil, nil
				case "nil profile":
					result.Profile = nil
				case "zero id":
					result.Profile.Id = 0
				case "zero owner":
					result.Profile.UserId = 0
				case "zero version":
					result.Version = 0
				case "wrong owner":
					result.Profile.UserId = 999
				}
				return result, nil
			}})
			response, body := profileHTTPRequest(t, app, `{"name":"NASA"}`, "en", true)
			if response.StatusCode != 500 || !strings.Contains(body, `"data":[]`) || strings.Contains(body, "NASA") || !reflect.DeepEqual(calls, []string{"Me", "UpdateProfile"}) {
				t.Fatalf("response=%d %s calls=%v", response.StatusCode, body, calls)
			}
		})
	}
}
func profileHTTPFailures(t *testing.T) {
	for _, stage := range []string{"Me", "UpdateProfile"} {
		for _, httpStatus := range []int{401, 403, 409, 429, 503, 504, 499, 500} {
			t.Run(stage+"/"+http.StatusText(httpStatus), func(t *testing.T) {
				var failure error
				switch httpStatus {
				case 409, 429, 503:
					failure = testutils.ServiceFieldError(t, httpStatus)
				case 401:
					failure = common.ErrUnauthorized
				case 403:
					failure = status.Error(codes.PermissionDenied, "private-sentinel")
				case 504:
					failure = context.DeadlineExceeded
				case 499:
					failure = context.Canceled
				default:
					failure = status.Error(codes.Internal, "private-sentinel")
				}
				calls := []string{}
				app := newProfileApp(t, &mocks.Service{MeFunc: func(context.Context, *emptypb.Empty, ...grpc.CallOption) (*userv1.UserData, error) {
					calls = append(calls, "Me")
					if stage == "Me" {
						return nil, failure
					}
					return sharedfixtures.MeUserData(), nil
				}}, &profilemocks.ProfileService{UpdateFunc: func(context.Context, *profilev1.ProfileRequest) (*profilev1.ProfileResponse, error) {
					calls = append(calls, "UpdateProfile")
					return nil, failure
				}})
				response, body := profileHTTPRequest(t, app, `{"name":"NASA"}`, "id-ID", true)
				wantCalls := []string{"Me"}
				if stage == "UpdateProfile" {
					wantCalls = append(wantCalls, stage)
				}
				if response.StatusCode != httpStatus || !reflect.DeepEqual(calls, wantCalls) || strings.Contains(body, "private-sentinel") || !strings.Contains(body, `"data":[]`) {
					t.Fatalf("response=%d %s calls=%v want %v", response.StatusCode, body, calls, wantCalls)
				}
				if httpStatus == 409 || httpStatus == 429 || httpStatus == 503 {
					testutils.AssertServiceErrorResponse(t, response, body, httpStatus)
				}
			})
		}
	}
}
func newProfileApp(t *testing.T, user authservice.Service, profile authservice.ProfileService) *fiber.App {
	t.Helper()
	client, err := authservice.NewWithServices(newTransport(t), user, profile)
	if err != nil {
		t.Fatal(err)
	}
	return sharedfixtures.NewApp(t, client)
}
func profileHTTPRequest(t *testing.T, app *fiber.App, raw, language string, bearer bool) (*http.Response, string) {
	t.Helper()
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/me/profile?user_id=999&namespace=iam", strings.NewReader(raw))
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	request.Header.Set(locale.HTTPHeader, language)
	request.Header.Set("X-Service-Token", "forged")
	request.Header.Set("Namespace", "iam")
	request.Header.Set("User_ID", "999")
	request.Header.Set("X-Service-Name", "forged")
	if bearer {
		request.Header.Set(fiber.HeaderAuthorization, authorizationValue)
	}
	response, err := app.Test(request)
	return response, testutils.ResponseBody(t, response, err)
}
