package test_scenarios

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/gofiber/fiber/v3"
	sessionv1 "github.com/siti-nabila/api-contracts/pb/session/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	userfixtures "github.com/siti-nabila/rest-orc/test/integration/users/fixtures"
	shared "github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func detailConcurrentTarget(i int) uint64 {
	if i == 16 {
		return uint64(i + 100)
	}
	return uint64(i + 200)
}
func detailConcurrentMethod(op int) string {
	switch op {
	case 0:
		return userv1.UserService_GetUser_FullMethodName
	case 1:
		return sessionv1.SessionService_ListUserSessions_FullMethodName
	case 2:
		return sessionv1.SessionService_RevokeUserSession_FullMethodName
	case 3:
		return sessionv1.SessionService_RevokeAllUserSessions_FullMethodName
	case 4:
		return sessionv1.SessionService_ListMySessions_FullMethodName
	case 5:
		return userv1.UserService_Register_FullMethodName
	case 6:
		return userv1.UserService_Login_FullMethodName
	default:
		return userv1.UserService_ListUsers_FullMethodName
	}
}
func detailWireConcurrent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const count = 32
		entered, release := make(chan struct{}, count), make(chan struct{})
		cfg := secureConfig(t)
		h := newTransport(t, cfg, defaultServerTLS(t), func(c context.Context, r any, info *grpc.UnaryServerInfo, next grpc.UnaryHandler) (any, error) {
			md, _ := metadata.FromIncomingContext(c)
			ids := md.Get("x-request-id")
			if len(ids) != 1 {
				return nil, fmt.Errorf("missing request ID")
			}
			i, err := strconv.Atoi(ids[0])
			if err != nil {
				return nil, err
			}
			id := ids[0]
			op := i % 8
			if info.FullMethod == userv1.UserService_Me_FullMethodName {
				u := shared.MeUserData()
				u.Id = uint64(i + 100)
				u.Version = 1
				u.Permissions = map[string]bool{"iam.user.read": true, "iam.session.read": true, "iam.session.revoke": true}
				if i == 24 {
					u.Permissions["iam.user.read"] = false
					entered <- struct{}{}
					<-release
				}
				if op == 7 {
					u.RoleCodes = []int32{2}
					u.RoleNames = []string{"admin"}
				}
				return u, nil
			}
			if info.FullMethod != detailConcurrentMethod(op) {
				t.Error("method crossed requests")
			}
			switch op {
			case 0:
				if r.(*userv1.GetUserRequest).Id != detailConcurrentTarget(i) {
					t.Error("detail target crossed")
				}
				if len(md.Get("version")) != 0 || len(md.Get("idempotency-key")) != 0 {
					t.Error("detail preconditions")
				}
			case 1:
				q := r.(*sessionv1.ListUserSessionsRequest)
				if q.UserId != uint64(i+200) || q.Query.LastId != "cursor-"+id {
					t.Error("admin list crossed")
				}
			case 2:
				q := r.(*sessionv1.RevokeUserSessionRequest)
				if q.UserId != uint64(i+200) || q.SessionId != concurrentSessionID(i) {
					t.Error("revoke target crossed")
				}
				assertMetadata(t, md, "idempotency-key", "key, "+id)
			case 3:
				q := r.(*sessionv1.RevokeAllUserSessionsRequest)
				if q.UserId != uint64(i+200) || q.ExpectedUserVersion == nil || *q.ExpectedUserVersion != uint64(9007199254740993+i) {
					t.Error("revoke-all target/version crossed")
				}
				assertMetadata(t, md, "idempotency-key", "key, "+id)
			case 4:
				if r.(*sessionv1.ListMySessionsRequest).Query.LastId != "cursor-"+id {
					t.Error("self cursor crossed")
				}
			case 5, 6:
				q := r.(*userv1.AuthRequest)
				if q.Email != "user"+id+"@example.com" || q.Password != "valid-password" {
					t.Error("public payload crossed")
				}
			}
			entered <- struct{}{}
			<-release
			switch op {
			case 0:
				if i == 8 {
					return nil, status.Error(codes.PermissionDenied, "private-sentinel")
				}
				result := shared.UserDetailResponse(detailConcurrentTarget(i))
				result.User.Name = "target-" + id
				result.User.Version = uint64(9007199254740993 + i)
				result.User.Roles[0].Name = "role-" + id
				result.User.ApplicationAccess[0].Name = "app-" + id
				result.User.MustChangePassword = i == 16
				if i == 16 {
					result.User.Status = "SUSPENDED"
				}
				return result, nil
			case 1, 4:
				owner := uint64(i + 200)
				if op == 4 {
					owner = uint64(i + 100)
				}
				result := shared.MySessionsResponse(owner)
				result.Items[0].Id = concurrentSessionID(i)
				result.Items[0].DeviceLabel = "device-" + id
				result.NextCursor = "next-" + id
				return result, nil
			case 2:
				return &sessionv1.RevokeSessionResponse{Changed: i%16 == 2}, nil
			case 3:
				return &sessionv1.RevokeAllUserSessionsResponse{Changed: i%16 == 3, Version: uint64(9007199254740900 + i)}, nil
			case 5:
				return shared.RegistrationResponse(), nil
			case 6:
				result := shared.LoginTokenResponse()
				result.Token = "token-" + id
				result.RefreshToken = "refresh-" + id
				return result, nil
			default:
				return next(c, r)
			}
		})
		defer h.close(t)
		app := shared.NewApp(t, h.client, func(c fiber.Ctx) error {
			md := metadata.Pairs("x-request-id", c.Get("X-Request-ID"))
			if i, _ := strconv.Atoi(c.Get("X-Request-ID")); i%8 == 0 {
				md.Set("version", "inherited")
				md.Set("idempotency-key", "inherited")
			}
			c.SetContext(metadata.NewOutgoingContext(c.Context(), md))
			return c.Next()
		})
		type outcome struct {
			i   int
			res *http.Response
			err error
		}
		results := make(chan outcome, count)
		for i := 0; i < count; i++ {
			go func(i int) {
				id := strconv.Itoa(i)
				method, path, body := "GET", fmt.Sprintf("/api/v1/users/%d", detailConcurrentTarget(i)), "{"
				switch i % 8 {
				case 1:
					path = fmt.Sprintf("/api/v1/users/%d/sessions?last_id=cursor-%s", i+200, id)
				case 2:
					method, path = "DELETE", fmt.Sprintf("/api/v1/users/%d/sessions/%s", i+200, concurrentSessionID(i))
				case 3:
					method, path = "POST", fmt.Sprintf("/api/v1/users/%d/sessions/revoke-all", i+200)
				case 4:
					path = "/api/v1/me/sessions?last_id=cursor-" + id
				case 5:
					method, path = "POST", "/api/v1/users/create"
				case 6:
					method, path = "POST", "/api/v1/users/login"
				case 7:
					path = "/api/v1/users"
				}
				if i%8 == 5 || i%8 == 6 {
					body = `{"email":"user` + id + `@example.com","password":"valid-password"}`
				}
				req := httptest.NewRequest(method, path, strings.NewReader(body))
				req.Header.Set("Authorization", "Bearer owner-"+id)
				req.Header.Set("X-Request-ID", id)
				req.Header.Set(locale.HTTPHeader, []string{"en", "id"}[i%2])
				req.Header.Set("Idempotency-Key", "key, "+id)
				req.Header.Set("Version", strconv.FormatUint(uint64(9007199254740993+i), 10))
				req.Header.Set("Origin", shared.BrowserOrigin)
				req.Header.Set("Content-Type", "application/json")
				res, err := app.Test(req, fiber.TestConfig{})
				results <- outcome{i, res, err}
			}(i)
		}
		for range count {
			<-entered
		}
		close(release)
		for range count {
			got := <-results
			i := got.i
			id := strconv.Itoa(i)
			op := i % 8
			body := testutils.ResponseBody(t, got.res, got.err)
			want := 200
			if i == 8 || i == 24 {
				want = 403
			}
			if got.res.StatusCode != want {
				t.Fatalf("request %d status %d want %d %s", i, got.res.StatusCode, want, body)
			}
			if op != 7 && got.res.Header.Get("Cache-Control") != "no-store" {
				t.Fatal("no-store")
			}
			if op != 6 && len(got.res.Cookies()) != 0 {
				t.Fatal("cookie crossed")
			}
			if want != 200 {
				if !strings.Contains(body, `"data":[]`) {
					t.Fatal(body)
				}
				continue
			}
			var env struct{ Data map[string]json.RawMessage }
			if json.Unmarshal([]byte(body), &env) != nil {
				t.Fatal(body)
			}
			switch op {
			case 0:
				var roles []struct{ Name string }
				var apps []struct{ Name string }
				json.Unmarshal(env.Data["roles"], &roles)
				json.Unmarshal(env.Data["application_access"], &apps)
				if len(env.Data) != 12 || string(env.Data["id"]) != fmt.Sprint(detailConcurrentTarget(i)) || string(env.Data["version"]) != `"`+strconv.FormatUint(uint64(9007199254740993+i), 10)+`"` || string(env.Data["name"]) != `"target-`+id+`"` || len(roles) != 2 || roles[0].Name != "role-"+id || len(apps) != 1 || apps[0].Name != "app-"+id || string(env.Data["must_change_password"]) != strconv.FormatBool(i == 16) {
					t.Fatal("detail crossed", body)
				}
				if i == 16 && string(env.Data["status"]) != `"SUSPENDED"` {
					t.Fatal(body)
				}
			case 1, 4:
				var items []struct {
					ID     string `json:"id"`
					UserID uint64 `json:"user_id"`
					Device string `json:"device_label"`
				}
				owner := uint64(i + 200)
				if op == 4 {
					owner = uint64(i + 100)
				}
				if json.Unmarshal(env.Data["items"], &items) != nil || len(items) != 1 || items[0].ID != concurrentSessionID(i) || items[0].UserID != owner || items[0].Device != "device-"+id || string(env.Data["next_cursor"]) != `"next-`+id+`"` {
					t.Fatal("sessions crossed", body)
				}
			case 2:
				if len(env.Data) != 1 || string(env.Data["changed"]) != strconv.FormatBool(i%16 == 2) {
					t.Fatal(body)
				}
			case 3:
				if len(env.Data) != 2 || string(env.Data["changed"]) != strconv.FormatBool(i%16 == 3) || string(env.Data["version"]) != `"`+strconv.FormatUint(uint64(9007199254740900+i), 10)+`"` {
					t.Fatal(body)
				}
			case 5:
				if body != shared.RegisterSuccessBody {
					t.Fatal(body)
				}
			case 6:
				if string(env.Data["token"]) != `"token-`+id+`"` || len(got.res.Cookies()) != 2 {
					t.Fatal("login crossed", body)
				}
				for _, cookie := range got.res.Cookies() {
					if cookie.Name == "__Secure-refresh" && cookie.Value != "refresh-"+id {
						t.Fatal("refresh crossed")
					}
				}
			case 7:
				if body != userfixtures.SuccessBody {
					t.Fatal(body)
				}
			}
			assertNoSecrets(t, body, cfg.ServiceToken)
			for _, secret := range []string{"private-password", "private-dob", "Bearer owner-", "refresh-"} {
				if strings.Contains(body, secret) {
					t.Fatal("secret leaked")
				}
			}
		}
		calls, dials := h.snapshot()
		if dials != 1 {
			t.Fatal("connection not reused")
		}
		methods := map[int][]string{}
		for _, c := range calls {
			id := c.metadata.Get("x-request-id")[0]
			i, _ := strconv.Atoi(id)
			methods[i] = append(methods[i], c.method)
			assertMetadata(t, c.metadata, "x-service-token", cfg.ServiceToken)
			assertMetadata(t, c.metadata, locale.MetadataKey, []string{"en", "id"}[i%2])
			if i%8 == 5 || i%8 == 6 {
				if len(c.metadata.Get("authorization")) != 0 {
					t.Fatal("public bearer")
				}
			} else {
				assertMetadata(t, c.metadata, "authorization", "Bearer owner-"+id)
			}
			if c.tls != tls.VersionTLS13 || c.deadline.IsZero() {
				t.Fatal("TLS/deadline")
			}
		}
		for i := 0; i < count; i++ {
			sequence := methods[i]
			op := i % 8
			n := 2
			if op == 5 || op == 6 || i == 24 {
				n = 1
			}
			if len(sequence) != n {
				t.Fatal("extra lookup/retry", i, sequence)
			}
			if op == 5 || op == 6 {
				if sequence[0] != detailConcurrentMethod(op) {
					t.Fatal("public order")
				}
			} else if sequence[0] != userv1.UserService_Me_FullMethodName || n == 2 && sequence[1] != detailConcurrentMethod(op) {
				t.Fatal("protected order", sequence)
			}
		}
	})
}
