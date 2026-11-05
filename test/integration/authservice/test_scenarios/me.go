package test_scenarios

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"testing"
	"testing/synctest"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	sharedfixtures "github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

func meHTTPSnapshots(t *testing.T) {
	for _, restricted := range []bool{false, true} {
		synctest.Test(t, func(t *testing.T) {
			cfg := secureConfig(t)
			h := newTransport(t, cfg, defaultServerTLS(t), func(_ context.Context, _ any, info *grpc.UnaryServerInfo, _ grpc.UnaryHandler) (any, error) {
				if info.FullMethod != userv1.UserService_Me_FullMethodName {
					t.Error("unexpected aggregation RPC")
				}
				user := sharedfixtures.MeUserData()
				// Keep privileges deliberately populated to exercise restricted suppression.
				user.MustChangePassword = restricted
				return user, nil
			})
			defer h.close(t)
			payload := `{"user_id":999,"application_code":"iam","namespace":"iam","permissions":{"iam.admin":true},"x-service-token":"ignored"}`
			response, body := secureHTTPRequest(t, sharedfixtures.NewApp(t, h.client), http.MethodGet, "/api/v1/me?application_code=iam&permissions=iam.admin", payload, true)
			data := readMeData(t, response, body)
			for field, want := range map[string]string{
				"id": "17", "email": `"member@example.com"`, "version": `"9007199254740993"`,
				"created_at": `"2026-10-20T08:09:10.123456789Z"`, "phone": `"+62+628123"`,
			} {
				assertMeField(t, data, field, want)
			}
			if restricted {
				assertMeField(t, data, "must_change_password", "true")
				for _, field := range []string{"role_codes", "role_names", "roles", "application_access"} {
					assertMeField(t, data, field, "[]")
				}
				assertMeField(t, data, "permissions", "{}")
			} else {
				assertMeField(t, data, "must_change_password", "false")
				assertMeField(t, data, "role_codes", "[3]")
				var permissions map[string]bool
				if err := json.Unmarshal(data["permissions"], &permissions); err != nil {
					t.Fatal(err)
				}
				if !permissions["clm.document.read"] || permissions["clm.document.delete"] || permissions["iam.admin"] {
					t.Fatal("permission values changed")
				}
			}
			calls, dials := h.snapshot()
			assertHTTPCalls(t, calls, []string{userv1.UserService_Me_FullMethodName}, cfg.ServiceToken, true)
			req, ok := calls[0].request.(*emptypb.Empty)
			if !ok || proto.Size(req) != 0 || calls[0].tls != tls.VersionTLS13 || calls[0].deadline.IsZero() || dials != 1 {
				t.Fatal("Me empty request, secure transport or connection reuse changed")
			}
			assertNoSecrets(t, body, cfg.ServiceToken)
		})
	}
}

func meHTTPFreshSnapshots(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var sequence atomic.Int32
		cfg := secureConfig(t)
		h := newTransport(t, cfg, defaultServerTLS(t), func(context.Context, any, *grpc.UnaryServerInfo, grpc.UnaryHandler) (any, error) {
			user := sharedfixtures.MeUserData()
			if sequence.Add(1) == 2 {
				user.Email = "updated@example.com"
				user.Version = 7
				user.MustChangePassword = true
			}
			return user, nil
		})
		defer h.close(t)
		app := sharedfixtures.NewApp(t, h.client)
		first, body := secureHTTPRequest(t, app, http.MethodGet, "/api/v1/me", "", true)
		assertMeField(t, readMeData(t, first, body), "email", `"member@example.com"`)
		second, body := secureHTTPRequest(t, app, http.MethodGet, "/api/v1/me", "", true)
		data := readMeData(t, second, body)
		assertMeField(t, data, "email", `"updated@example.com"`)
		assertMeField(t, data, "version", `"7"`)
		assertMeField(t, data, "permissions", "{}")
		assertMeField(t, data, "roles", "[]")
		calls, dials := h.snapshot()
		assertHTTPCalls(t, calls, []string{userv1.UserService_Me_FullMethodName, userv1.UserService_Me_FullMethodName}, cfg.ServiceToken, true)
		if dials != 1 {
			t.Fatal("Me created another connection")
		}
	})
}

func meHTTPMissingBearer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newTransport(t, secureConfig(t), defaultServerTLS(t), nil)
		defer h.close(t)
		response, _ := secureHTTPRequest(t, sharedfixtures.NewApp(t, h.client), http.MethodGet, "/api/v1/me", "", false)
		calls, dials := h.snapshot()
		if response.StatusCode != http.StatusUnauthorized || len(calls) != 0 || dials != 0 {
			t.Fatal("missing bearer reached Me or changed status")
		}
	})
}

func readMeData(t *testing.T, response *http.Response, body string) map[string]json.RawMessage {
	t.Helper()
	var envelope struct {
		Code    string                     `json:"code"`
		Message string                     `json:"message"`
		Data    map[string]json.RawMessage `json:"data"`
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("Me status=%d body=%s", response.StatusCode, body)
	}
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Code != "SS" || envelope.Message != "Profile retrieved successfully." || len(envelope.Data) != 15 {
		t.Fatalf("unexpected Me envelope: %s", body)
	}
	return envelope.Data
}

func assertMeField(t *testing.T, data map[string]json.RawMessage, field, want string) {
	t.Helper()
	if got := string(data[field]); got != want {
		t.Errorf("%s=%s, want %s", field, got, want)
	}
}
