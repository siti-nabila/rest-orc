package test_scenarios

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/gofiber/fiber/v3"
	profilev1 "github.com/siti-nabila/api-contracts/pb/profile/v1"
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

func profileConcurrentTarget(i int) uint64 {
	if i == 18 {
		return uint64(i + 100)
	}
	return uint64(i + 200)
}
func profileConcurrentMethod(op int) string {
	switch op {
	case 0:
		return userv1.UserService_UpdateUser_FullMethodName
	case 1:
		return ""
	case 2:
		return profilev1.ProfileService_UpdateProfile_FullMethodName
	case 3:
		return userv1.UserService_GetUser_FullMethodName
	case 4:
		return userv1.UserService_ChangeUserStatus_FullMethodName
	case 5:
		return sessionv1.SessionService_ListUserSessions_FullMethodName
	case 6:
		return userv1.UserService_Register_FullMethodName
	case 7:
		return userv1.UserService_Login_FullMethodName
	default:
		return userv1.UserService_ListUsers_FullMethodName
	}
}
func targetProfileWireConcurrent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const count = 36
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
			id, op := ids[0], i%9
			if info.FullMethod == userv1.UserService_Me_FullMethodName {
				u := shared.MeUserData()
				u.Id = uint64(i + 100)
				u.Version = 1
				u.Permissions = map[string]bool{"iam.user.update": true, "iam.user.read": true, "iam.user.status.update": true, "iam.session.read": true}
				if i == 27 {
					u.Permissions["iam.user.update"] = false
				}
				u.MustChangePassword = i == 19
				if op == 8 {
					u.RoleCodes = []int32{2}
					u.RoleNames = []string{"admin"}
				}
				if op == 1 || i == 27 {
					entered <- struct{}{}
					<-release
				}
				return u, nil
			}
			if info.FullMethod != profileConcurrentMethod(op) {
				t.Error("method crossed", i, info.FullMethod)
			}
			switch op {
			case 0:
				q := r.(*userv1.UpdateUserRequest)
				if q.UserId != profileConcurrentTarget(i) || q.ExpectedUserVersion == nil || *q.ExpectedUserVersion != uint64(9007199254740993+i) || q.FieldMask == nil || len(md.Get("version")) != 0 {
					t.Error("target/version crossed", q)
				}
				switch i {
				case 0:
					if q.Name == nil || *q.Name != "Alpha User" || q.Phone == nil || *q.Phone != "812345678901" || q.CountryCode == nil || *q.CountryCode != "+62" || q.Address != nil || !reflect.DeepEqual(q.FieldMask.Paths, []string{"name", "phone", "country_code"}) {
						t.Error("selected values crossed", q)
					}
				case 9:
					if q.Address == nil || *q.Address != " Address nine\r\n " || q.Name != nil || q.Phone != nil || q.CountryCode != nil || !reflect.DeepEqual(q.FieldMask.Paths, []string{"address"}) {
						t.Error("partial address crossed", q)
					}
				case 18:
					if q.Phone == nil || *q.Phone != "628123456789" || q.CountryCode == nil || *q.CountryCode != "+62" || q.Name != nil || q.Address != nil || !reflect.DeepEqual(q.FieldMask.Paths, []string{"phone", "country_code"}) {
						t.Error("partial phone crossed", q)
					}
				}
				if i == 18 {
					if len(md.Get("idempotency-key")) != 0 {
						t.Error("optional key inherited")
					}
				} else {
					assertMetadata(t, md, "idempotency-key", "key, "+id)
				}
			case 2:
				q := r.(*profilev1.ProfileRequest)
				if q.Name != "Self User" || q.FieldMask == nil || !reflect.DeepEqual(q.FieldMask.Paths, []string{"name"}) {
					t.Error("self profile crossed", q)
				}
			case 3:
				q := r.(*userv1.GetUserRequest)
				if q.Id != profileConcurrentTarget(i) || len(md.Get("version")) != 0 || len(md.Get("idempotency-key")) != 0 {
					t.Error("detail crossed", q)
				}
			case 4:
				q := r.(*userv1.ChangeUserStatusRequest)
				if q.UserId != profileConcurrentTarget(i) || q.Status == nil || *q.Status != "SUSPENDED" || q.Reason != "reason "+id || q.ExpectedUserVersion == nil || *q.ExpectedUserVersion != uint64(9007199254740993+i) {
					t.Error("status crossed", q)
				}
				assertMetadata(t, md, "idempotency-key", "key, "+id)
			case 5:
				q := r.(*sessionv1.ListUserSessionsRequest)
				if q.UserId != profileConcurrentTarget(i) || q.Query.LastId != "cursor-"+id {
					t.Error("session crossed", q)
				}
			case 6, 7:
				q := r.(*userv1.AuthRequest)
				if q.Email != "user"+id+"@example.com" || q.Password != "valid-password" {
					t.Error("public crossed", q)
				}
			}
			entered <- struct{}{}
			<-release
			switch op {
			case 0:
				if i == 9 {
					return nil, status.Error(codes.PermissionDenied, "private-sentinel")
				}
				u := shared.UserDetailResponse(profileConcurrentTarget(i)).User
				u.Name = "target-" + id
				u.Version = uint64(9007199254740900 + i)
				u.Roles[0].Name = "role-" + id
				u.ApplicationAccess[0].Name = "app-" + id
				return &userv1.UpdateUserResponse{User: u, Changed: i == 0}, nil
			case 2:
				u := userfixtures.ProfileResponse()
				u.Profile.UserId = uint64(i + 100)
				u.Profile.Name = "self-" + id
				u.Version = uint64(9007199254740900 + i)
				u.Changed = i%18 == 2
				return u, nil
			case 3:
				u := shared.UserDetailResponse(profileConcurrentTarget(i))
				u.User.Name = "detail-" + id
				u.User.Version = uint64(9007199254740900 + i)
				return u, nil
			case 4:
				return &userv1.ChangeUserStatusResponse{UserId: profileConcurrentTarget(i), Status: "SUSPENDED", Version: uint64(9007199254740900 + i), Changed: i%18 == 4}, nil
			case 5:
				u := shared.MySessionsResponse(profileConcurrentTarget(i))
				u.Items[0].Id = concurrentSessionID(i)
				u.Items[0].DeviceLabel = "device-" + id
				u.NextCursor = "next-" + id
				return u, nil
			case 6:
				return shared.RegistrationResponse(), nil
			case 7:
				u := shared.LoginTokenResponse()
				u.Token = "token-" + id
				u.RefreshToken = "refresh-" + id
				return u, nil
			default:
				return next(c, r)
			}
		})
		defer h.close(t)
		app := shared.NewApp(t, h.client, func(c fiber.Ctx) error {
			md := metadata.Pairs("x-request-id", c.Get("X-Request-ID"))
			i, _ := strconv.Atoi(c.Get("X-Request-ID"))
			if i%9 == 0 || i%9 == 3 {
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
				id, op := strconv.Itoa(i), i%9
				method, path, body := "PATCH", fmt.Sprintf("/api/v1/users/%d/profile", profileConcurrentTarget(i)), `{"name":"Delta User"}`
				switch op {
				case 0:
					switch i {
					case 0:
						body = `{"name":"  alpha  User ","country_code":"62","phone":"0812345678901"}`
					case 9:
						body = `{"address":" Address nine\r\n "}`
					case 18:
						body = `{"country_code":"+62","phone":"62628123456789"}`
					}
				case 1:
					body = []string{"null", "{}", `{"name":null,"field_mask":["name"],"user_id":99}`, `{"email":"forged"}`}[i/9]
				case 2:
					path = "/api/v1/me/profile"
					body = `{"name":"  self  User "}`
				case 3:
					method, path, body = "GET", fmt.Sprintf("/api/v1/users/%d", profileConcurrentTarget(i)), "{"
				case 4:
					path = fmt.Sprintf("/api/v1/users/%d/status", profileConcurrentTarget(i))
					body = `{"status":"SUSPENDED","reason":"  reason ` + id + `  "}`
				case 5:
					method, path, body = "GET", fmt.Sprintf("/api/v1/users/%d/sessions?last_id=cursor-%s", profileConcurrentTarget(i), id), ""
				case 6:
					method, path = "POST", "/api/v1/users/create"
				case 7:
					method, path = "POST", "/api/v1/users/login"
				case 8:
					method, path, body = "GET", "/api/v1/users", ""
				}
				if op == 6 || op == 7 {
					body = `{"email":"user` + id + `@example.com","password":"valid-password"}`
				}
				req := httptest.NewRequest(method, path, strings.NewReader(body))
				req.Header.Set("Authorization", "Bearer owner-"+id)
				req.Header.Set("X-Request-ID", id)
				req.Header.Set(locale.HTTPHeader, []string{"en", "id"}[i%2])
				req.Header.Set("Version", strconv.FormatUint(uint64(9007199254740993+i), 10))
				if i != 18 {
					req.Header.Set("Idempotency-Key", "key, "+id)
				}
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
			i, op := got.i, got.i%9
			id := strconv.Itoa(i)
			body := testutils.ResponseBody(t, got.res, got.err)
			want := 200
			if i == 9 || i == 19 || i == 27 {
				want = 403
			}
			if got.res.StatusCode != want {
				t.Fatalf("request %d status=%d want=%d %s", i, got.res.StatusCode, want, body)
			}
			if op != 2 && op != 8 && got.res.Header.Get("Cache-Control") != "no-store" {
				t.Fatal("no-store", i)
			}
			if op != 7 && len(got.res.Cookies()) != 0 {
				t.Fatal("cookie crossed", i)
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
			v := strconv.Quote(strconv.FormatUint(uint64(9007199254740900+i), 10))
			switch op {
			case 0:
				var user struct {
					ID            uint64
					Name, Version string
					Roles         []struct{ Name string }
					Apps          []struct{ Name string } `json:"application_access"`
				}
				if len(env.Data) != 2 || json.Unmarshal(env.Data["user"], &user) != nil || user.ID != profileConcurrentTarget(i) || user.Name != "target-"+id || strconv.Quote(user.Version) != v || len(user.Roles) != 2 || user.Roles[0].Name != "role-"+id || len(user.Apps) != 1 || user.Apps[0].Name != "app-"+id || string(env.Data["changed"]) != strconv.FormatBool(i == 0) {
					t.Fatal("target response crossed", body)
				}
			case 1:
				if len(env.Data) != 1 || string(env.Data["changed"]) != "false" {
					t.Fatal("local no-op crossed", body)
				}
			case 2:
				var p struct {
					UserID uint64 `json:"user_id"`
					Name   string
				}
				if json.Unmarshal(env.Data["profile"], &p) != nil || p.UserID != uint64(i+100) || p.Name != "self-"+id || string(env.Data["version"]) != v || len(env.Data) != 3 {
					t.Fatal("self response crossed", body)
				}
			case 3:
				if len(env.Data) != 12 || string(env.Data["id"]) != fmt.Sprint(profileConcurrentTarget(i)) || string(env.Data["name"]) != strconv.Quote("detail-"+id) || string(env.Data["version"]) != v {
					t.Fatal("detail crossed", body)
				}
			case 4:
				if len(env.Data) != 4 || string(env.Data["user_id"]) != fmt.Sprint(profileConcurrentTarget(i)) || string(env.Data["version"]) != v || string(env.Data["status"]) != `"SUSPENDED"` {
					t.Fatal("status crossed", body)
				}
			case 5:
				var items []struct {
					ID     string
					UserID uint64 `json:"user_id"`
					Device string `json:"device_label"`
				}
				if json.Unmarshal(env.Data["items"], &items) != nil || len(items) != 1 || items[0].ID != concurrentSessionID(i) || items[0].UserID != profileConcurrentTarget(i) || items[0].Device != "device-"+id || string(env.Data["next_cursor"]) != strconv.Quote("next-"+id) {
					t.Fatal("session crossed", body)
				}
			case 6:
				if body != shared.RegisterSuccessBody {
					t.Fatal(body)
				}
			case 7:
				if string(env.Data["token"]) != strconv.Quote("token-"+id) || len(got.res.Cookies()) != 2 {
					t.Fatal("login crossed", body)
				}
				for _, cookie := range got.res.Cookies() {
					if cookie.Name == "__Secure-refresh" && cookie.Value != "refresh-"+id {
						t.Fatal("refresh crossed")
					}
				}
			case 8:
				if body != userfixtures.SuccessBody {
					t.Fatal(body)
				}
			}
			assertNoSecrets(t, body, cfg.ServiceToken)
			for _, secret := range []string{"private-sentinel", "private-password", "private-dob", "Bearer owner-", "refresh-"} {
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
			if i%9 == 6 || i%9 == 7 {
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
			op := i % 9
			want := []string{userv1.UserService_Me_FullMethodName}
			if op == 6 || op == 7 {
				want = []string{profileConcurrentMethod(op)}
			} else if op != 1 && i != 27 {
				want = append(want, profileConcurrentMethod(op))
			}
			if !reflect.DeepEqual(methods[i], want) {
				t.Fatal("call order/retry", i, methods[i], want)
			}
		}
	})
}
