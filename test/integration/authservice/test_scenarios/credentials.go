package test_scenarios

import (
	"context"
	"crypto/tls"
	"reflect"
	"testing"
	"testing/synctest"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

func All() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "self-session HTTP wire uses exact SDK methods and authoritative outcomes", Run: mySessionsWireOutcomes},
		{Name: "self-session wire deadlines cancellation and no retry", Run: mySessionsWireContext},
		{Name: "concurrent self-session and Me isolate owners metadata and cookies", Run: mySessionsWireConcurrent},
		{Name: "session wire deadlines and cancellation have no retries or browser effects", Run: sessionWireContext},
		{Name: "mixed concurrent refresh logout and Me remain isolated on one connection", Run: sessionWireConcurrent},
		{Name: "session SDK wire preserves exact proof mode method errors and secure context", Run: sessionWireOutcomes},
		{Name: "password HTTP uses generated SDK exact credentials and one secure connection", Run: passwordWireOutcomes},
		{Name: "password HTTP preserves bounded deadlines and cancellation without cleanup on failure", Run: passwordWireContext},
		{Name: "concurrent password credentials metadata results and cookies stay isolated", Run: passwordWireConcurrent},
		{Name: "profile HTTP dispatches generated SDK with exact mask and one secure shared connection", Run: profileWireOutcomes},
		{Name: "profile HTTP preserves bounded deadlines cancellation and cleanup", Run: profileWireContext},
		{Name: "concurrent profile requests isolate caller payload and metadata", Run: profileWireConcurrent},
		{Name: "Me HTTP uses one secure snapshot for non-admin and restricted users", Run: meHTTPSnapshots},
		{Name: "Me HTTP obtains fresh snapshots on one connection", Run: meHTTPFreshSnapshots},
		{Name: "Me HTTP missing bearer performs no RPC", Run: meHTTPMissingBearer},
		{Name: "Me HTTP failures preserve safe status locale and retry guidance", Run: meHTTPErrors},
		{Name: "Me HTTP retains request context deadlines and cancellation", Run: meHTTPContext},
		{Name: "Me HTTP concurrent snapshots and metadata stay isolated", Run: meHTTPConcurrent},
		{Name: "secure HTTP pipeline preserves public routes and caller identity", Run: secureHTTPOutcomes},
		{Name: "secure HTTP pipeline preserves downstream denials and conflicts", Run: secureHTTPErrors},
		{Name: "request deadlines and cancellation remain bounded", Run: contextBoundaries},
		{Name: "concurrent credentials remain isolated on one connection", Run: concurrentCredentials},
		{Name: "credentials change only on explicit client reconstruction", Run: credentialRotation},
		{Name: "TLS authentication rejects untrusted peers and downgrades", Run: tlsBoundaries},
		{Name: "conflicting service metadata fails before dispatch", Run: conflictingCredentials},
		{Name: "every auth RPC sends one server credential over TLS", Run: credentialsOnEveryRPC},
	}
}

func credentialsOnEveryRPC(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cfg := secureConfig(t)
		h := newTransport(t, cfg, defaultServerTLS(t), nil)
		defer h.close(t)
		request := &userv1.AuthRequest{Email: " User@Example.COM ", Password: "valid-password"}
		public := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(locale.MetadataKey, "id-ID", "x-request-id", "public-request"))
		for index, invoke := range []func(context.Context, *userv1.AuthRequest) (*userv1.UserTokenResponse, error){h.client.Register, h.client.Login} {
			got, err := invoke(public, request)
			if err != nil || index == 0 && (got.GetToken() != "" || got.GetRegistrationId() != 17 || got.GetRegistrationStatus() != "PENDING") || index == 1 && got.GetToken() != "login-token" {
				t.Fatalf("public RPC failed: %v", err)
			}
		}
		md := metadata.Pairs("authorization", "Bearer user-test", locale.MetadataKey, "zh-CN", "x-request-id", "protected-request")
		ctx := metadata.NewOutgoingContext(context.Background(), md)
		me, err := h.client.Me(ctx)
		if err != nil || me.GetId() != 17 {
			t.Fatal("Me response mapping changed")
		}
		list, err := h.client.ListUsers(ctx, &userv1.ListUsersRequest{})
		if err != nil || list.GetTotal() != 1 {
			t.Fatal("ListUsers response mapping changed")
		}
		calls, dials := h.snapshot()
		if len(calls) != 4 || dials != 1 {
			t.Fatalf("RPCs/dials = %d/%d, want 4/1", len(calls), dials)
		}
		methods := []string{userv1.UserService_Register_FullMethodName, userv1.UserService_Login_FullMethodName, userv1.UserService_Me_FullMethodName, userv1.UserService_ListUsers_FullMethodName}
		for i, got := range calls {
			if got.method != methods[i] || got.tls != tls.VersionTLS13 || !reflect.DeepEqual(got.metadata.Get("x-service-token"), []string{cfg.ServiceToken}) {
				t.Error("method, negotiated TLS or service credential differs")
			}
			if got.deadline.IsZero() {
				t.Error("RPC has no configured deadline")
			}
			if i < 2 {
				if len(got.metadata.Get("authorization")) != 0 || !proto.Equal(got.request.(*userv1.AuthRequest), request) {
					t.Error("public RPC invented a bearer or changed raw arguments")
				}
				assertMetadata(t, got.metadata, locale.MetadataKey, "id-ID")
				assertMetadata(t, got.metadata, "x-request-id", "public-request")
			} else {
				assertMetadata(t, got.metadata, "authorization", "Bearer user-test")
				assertMetadata(t, got.metadata, locale.MetadataKey, "zh-CN")
				assertMetadata(t, got.metadata, "x-request-id", "protected-request")
			}
		}
		original, _ := metadata.FromOutgoingContext(ctx)
		if !reflect.DeepEqual(original, md) || len(md.Get("x-service-token")) != 0 {
			t.Error("caller metadata mutated")
		}
	})
}

func assertMetadata(t *testing.T, md metadata.MD, key, expected string) {
	t.Helper()
	if !reflect.DeepEqual(md.Get(key), []string{expected}) {
		t.Errorf("metadata %s was changed or duplicated", key)
	}
}
