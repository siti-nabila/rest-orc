package test_scenarios

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	authdictionary "github.com/siti-nabila/api-contracts/pkg/dictionary/auth"
	authhandler "github.com/siti-nabila/rest-orc/internal/features/auth/handler"
	handlerfixtures "github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func publicAuthSuccess(t *testing.T) {
	for _, tc := range []struct {
		name, route, password, token, body string
	}{
		{"register raw input", authhandler.RegisterRoute, "aA1!~~", "register-token", handlerfixtures.RegisterSuccessBody},
		{"register minimum", authhandler.RegisterRoute, "aaaaaa", "register-token", handlerfixtures.RegisterSuccessBody},
		{"register maximum", authhandler.RegisterRoute, strings.Repeat("~", 50), "register-token", handlerfixtures.RegisterSuccessBody},
		{"login raw input", authhandler.LoginRoute, " PaSs ", "login-token", handlerfixtures.LoginSuccessBody},
		{"login unicode bytes", authhandler.LoginRoute, "ééé", "login-token", handlerfixtures.LoginSuccessBody},
		{"login spaces", authhandler.LoginRoute, "      ", "login-token", handlerfixtures.LoginSuccessBody},
		{"login control", authhandler.LoginRoute, "ab\x00\t\r\n", "login-token", handlerfixtures.LoginSuccessBody},
		{"login maximum", authhandler.LoginRoute, strings.Repeat("é", 25), "login-token", handlerfixtures.LoginSuccessBody},
	} {
		for _, unknown := range []bool{false, true} {
			t.Run(tc.name, func(t *testing.T) {
				service, unexpectedCalls := rejectingAuthService(t)
				calls := 0
				const email = " User.Name@Example.COM "
				call := func(ctx context.Context, req *userv1.AuthRequest, _ ...grpc.CallOption) (*userv1.UserTokenResponse, error) {
					calls++
					testutils.AssertServiceContext(t, ctx)
					if req.GetEmail() != email || req.GetPassword() != tc.password {
						t.Error("RPC received different raw email or password")
					}
					if tc.route == authhandler.RegisterRoute {
						return handlerfixtures.RegistrationResponse(), nil
					}
					return handlerfixtures.LoginTokenResponse(), nil
				}
				if tc.route == authhandler.RegisterRoute {
					service.RegisterFunc = call
				} else {
					service.LoginFunc = call
				}
				payload := map[string]any{"email": email, "password": tc.password}
				if unknown {
					payload["confirm_password"] = "does-not-match"
					payload["unknown"] = map[string]any{"ignored": true}
				}
				encoded, err := json.Marshal(payload)
				if err != nil {
					t.Fatal(err)
				}
				response, body := authHTTP(t, newAuthApp(t, service), tc.route, string(encoded), "id-ID")
				if response.StatusCode != 200 || body != tc.body {
					t.Error("success status or lifecycle envelope changed")
				}
				wantCookies := 0
				if tc.route == authhandler.LoginRoute {
					wantCookies = 2
				}
				if len(response.Header.Values(fiber.HeaderSetCookie)) != wantCookies {
					t.Error("auth success unexpectedly set cookies")
				}
				if calls != 1 || *unexpectedCalls != 0 {
					t.Errorf("business calls = %d, unexpected RPCs = %d, want 1/0", calls, *unexpectedCalls)
				}
			})
		}
	}
}

func loginServiceFailure(t *testing.T) {
	service, unexpectedCalls := rejectingAuthService(t)
	calls := 0
	s, err := status.New(codes.Internal, "private authentication detail").WithDetails(
		&errdetails.ErrorInfo{Reason: authdictionary.ErrLoginFailed.Key(), Metadata: map[string]string{
			"http_status": "401", "code": "LOGIN_DENIED",
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	service.LoginFunc = func(ctx context.Context, req *userv1.AuthRequest, _ ...grpc.CallOption) (*userv1.UserTokenResponse, error) {
		calls++
		testutils.AssertServiceContext(t, ctx)
		if req.GetEmail() != "user@example.com" || req.GetPassword() != "legacy password" {
			t.Error("login arguments changed before downstream rejection")
		}
		return nil, s.Err()
	}
	response, body := authHTTP(t, newAuthApp(t, service), authhandler.LoginRoute,
		`{"email":"user@example.com","password":"legacy password"}`, "id-ID")
	if response.StatusCode != 401 || body != `{"code":"LOGIN_DENIED","errors":{"description":"Login gagal."},"data":[]}` {
		t.Error("downstream login status, code or safe localized body changed")
	}
	if calls != 1 || *unexpectedCalls != 0 {
		t.Errorf("login calls = %d, unexpected RPCs = %d, want 1/0 without retry", calls, *unexpectedCalls)
	}
}
