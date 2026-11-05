package test_scenarios

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
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
	usererrors "github.com/siti-nabila/api-contracts/pkg/dictionary/user"
	"github.com/siti-nabila/api-contracts/pkg/grpcerror"
	"github.com/siti-nabila/api-contracts/pkg/grpcerror/mapping"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	userfixtures "github.com/siti-nabila/rest-orc/test/integration/users/fixtures"
	shared "github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func creationMixedMethod(op int) string {
	switch op {
	case 0, 1, 2:
		return userv1.UserService_CreateUser_FullMethodName
	case 3:
		return userv1.UserService_Register_FullMethodName
	case 4:
		return userv1.UserService_Login_FullMethodName
	case 5:
		return userv1.UserService_ListUsers_FullMethodName
	case 6:
		return profilev1.ProfileService_UpdateProfile_FullMethodName
	case 7:
		return userv1.UserService_GetUser_FullMethodName
	case 8:
		return userv1.UserService_ChangeUserStatus_FullMethodName
	case 9:
		return sessionv1.SessionService_ListUserSessions_FullMethodName
	case 10:
		return userv1.UserService_UpdateUser_FullMethodName
	}
	return ""
}
func creationWireConcurrent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const count = 36
		entered, release := make(chan struct{}, count*2), make(chan struct{})
		cfg := secureConfig(t)
		h := newTransport(t, cfg, defaultServerTLS(t), func(c context.Context, r any, info *grpc.UnaryServerInfo, next grpc.UnaryHandler) (any, error) {
			md, _ := metadata.FromIncomingContext(c)
			ids := md.Get("x-request-id")
			if len(ids) != 1 {
				return nil, fmt.Errorf("missing ID")
			}
			id := ids[0]
			i, _ := strconv.Atoi(id)
			op := i % 12
			if info.FullMethod == userv1.UserService_Me_FullMethodName {
				u := shared.MeUserData()
				u.Id = uint64(i + 100)
				u.Permissions = map[string]bool{"iam.user.create": true, "iam.user.read": true, "iam.user.update": true, "iam.user.status.update": true, "iam.session.read": true}
				if op == 5 {
					u.RoleCodes = []int32{2}
					u.RoleNames = []string{"admin"}
				}
				if op == 11 {
					entered <- struct{}{}
					<-release
				}
				return u, nil
			}
			if info.FullMethod != creationMixedMethod(op) {
				t.Error("method crossed", i, info.FullMethod)
			}
			switch op {
			case 0, 1, 2:
				q := r.(*userv1.CreateUserRequest)
				wantName := string(rune('A'+i%26)) + "lpha User"
				if q.Email != " User"+id+"@Example.COM " || q.Password != "secret-"+id || q.Name != wantName || q.Phone != "628123456789" || q.Address != " Address "+id+"\r\n " || q.CountryCode == nil || *q.CountryCode != "+62" || q.Status == nil || *q.Status != "PENDING" || len(md.Get("version")) != 0 {
					t.Error("create inputs crossed", i)
				}
				assertMetadata(t, md, "idempotency-key", "key, "+id)
			case 3, 4:
				q := r.(*userv1.AuthRequest)
				if q.Email != "user"+id+"@example.com" || q.Password != "secret-"+id {
					t.Error("public credentials crossed")
				}
			case 6:
				q := r.(*profilev1.ProfileRequest)
				if q.Name != "Self User" {
					t.Error("self crossed")
				}
			case 7:
				q := r.(*userv1.GetUserRequest)
				if q.Id != uint64(i+200) || len(md.Get("idempotency-key")) != 0 || len(md.Get("version")) != 0 {
					t.Error("detail crossed")
				}
			case 8:
				q := r.(*userv1.ChangeUserStatusRequest)
				if q.UserId != uint64(i+200) || q.GetExpectedUserVersion() != uint64(9007199254740993+i) || q.GetStatus() != "SUSPENDED" || q.Reason != "reason "+id {
					t.Error("status crossed")
				}
				assertMetadata(t, md, "idempotency-key", "key, "+id)
			case 9:
				q := r.(*sessionv1.ListUserSessionsRequest)
				if q.UserId != uint64(i+200) || q.Query.LastId != "cursor-"+id {
					t.Error("session crossed")
				}
			case 10:
				q := r.(*userv1.UpdateUserRequest)
				if q.UserId != uint64(i+200) || q.Name == nil || *q.Name != "Target User" || q.GetExpectedUserVersion() != uint64(9007199254740993+i) || len(md.Get("version")) != 0 {
					t.Error("profile crossed")
				}
				assertMetadata(t, md, "idempotency-key", "key, "+id)
			}
			entered <- struct{}{}
			<-release
			switch op {
			case 0, 1, 2:
				if op == 2 {
					return nil, grpcerror.NewEncoder(mapping.UserCodeMappings()...).Encode(usererrors.ErrIdempotencyConflict, locale.English)
				}
				u := shared.UserDetailResponse(uint64(i + 200)).User
				u.Email = "user" + id + "@example.com"
				u.Status = "PENDING"
				u.Version = uint64(9007199254740993 + i)
				u.Roles[0].Name = "role-" + id
				u.ApplicationAccess[0].Name = "app-" + id
				return &userv1.CreateUserResponse{User: u}, nil
			case 3:
				return shared.RegistrationResponse(), nil
			case 4:
				u := shared.LoginTokenResponse()
				u.Token = "token-" + id
				u.RefreshToken = "refresh-" + id
				return u, nil
			case 6:
				u := userfixtures.ProfileResponse()
				u.Profile.UserId = uint64(i + 100)
				u.Profile.Name = "self-" + id
				return u, nil
			case 7:
				u := shared.UserDetailResponse(uint64(i + 200))
				u.User.Name = "detail-" + id
				return u, nil
			case 8:
				return &userv1.ChangeUserStatusResponse{UserId: uint64(i + 200), Status: "SUSPENDED", Version: uint64(9007199254740993 + i)}, nil
			case 9:
				u := shared.MySessionsResponse(uint64(i + 200))
				u.Items[0].DeviceLabel = "device-" + id
				return u, nil
			case 10:
				u := shared.UserDetailResponse(uint64(i + 200)).User
				u.Name = "profile-" + id
				return &userv1.UpdateUserResponse{User: u}, nil
			}
			return next(c, r)
		})
		defer h.close(t)
		app := shared.NewApp(t, h.client, func(c fiber.Ctx) error {
			md := metadata.Pairs("x-request-id", c.Get("X-Request-ID"))
			i, _ := strconv.Atoi(c.Get("X-Request-ID"))
			if i%12 < 3 || i%12 == 7 || i%12 == 10 {
				md.Set("version", "inherited")
				md.Set("idempotency-key", "inherited")
			}
			c.SetContext(metadata.NewOutgoingContext(c.Context(), md))
			return c.Next()
		})
		type result struct {
			i     int
			res   *http.Response
			err   error
			first string
		}
		results := make(chan result, count)
		for i := 0; i < count; i++ {
			go func(i int) {
				id, op := strconv.Itoa(i), i%12
				method, path, raw := "POST", "/api/v1/users", ""
				switch op {
				case 0, 1, 2:
					body := map[string]any{"email": " User" + id + "@Example.COM ", "password": "secret-" + id, "name": " " + string(rune('a'+i%26)) + "lpha  user ", "phone": "62628123456789", "country_code": "62", "address": " Address " + id + "\r\n ", "roles": []string{"admin"}}
					b, _ := json.Marshal(body)
					raw = string(b)
				case 3, 4:
					path = "/api/v1/users/create"
					if op == 4 {
						path = "/api/v1/users/login"
					}
					raw = `{"email":"user` + id + `@example.com","password":"secret-` + id + `"}`
				case 5:
					method, path = "GET", "/api/v1/users"
				case 6:
					method, path, raw = "PATCH", "/api/v1/me/profile", `{"name":"  self  user "}`
				case 7:
					method, path = "GET", fmt.Sprintf("/api/v1/users/%d", i+200)
				case 8:
					method, path, raw = "PATCH", fmt.Sprintf("/api/v1/users/%d/status", i+200), `{"status":"SUSPENDED","reason":" reason `+id+` "}`
				case 9:
					method, path = "GET", fmt.Sprintf("/api/v1/users/%d/sessions?last_id=cursor-%s", i+200, id)
				case 10:
					method, path, raw = "PATCH", fmt.Sprintf("/api/v1/users/%d/profile", i+200), `{"name":" target  user "}`
				case 11:
					raw = `{"email":"bad","password":"bad","name":"!","phone":"bad","country_code":"bad","status":"ACTIVE"}`
				}
				invoke := func() (*http.Response, error) {
					req := httptest.NewRequest(method, path, strings.NewReader(raw))
					req.Header.Set("Authorization", "Bearer owner-"+id)
					req.Header.Set("X-Request-ID", id)
					req.Header.Set(locale.HTTPHeader, []string{"en", "id"}[i%2])
					req.Header.Set("Version", strconv.FormatUint(uint64(9007199254740993+i), 10))
					req.Header.Set("Idempotency-Key", "key, "+id)
					req.Header.Set("Origin", shared.BrowserOrigin)
					req.Header.Set("Content-Type", "application/json")
					return app.Test(req, fiber.TestConfig{})
				}
				res, err := invoke()
				first := ""
				if op == 1 {
					if err != nil {
						results <- result{i, res, err, first}
						return
					}
					b, e := io.ReadAll(res.Body)
					res.Body.Close()
					first = string(b)
					if e != nil || res.StatusCode != 200 {
						t.Error("first replay failure", i, res.StatusCode, first)
					}
					res, err = invoke()
				}
				results <- result{i, res, err, first}
			}(i)
		}
		for range count {
			<-entered
		}
		close(release)
		for range count {
			got := <-results
			i, op := got.i, got.i%12
			id := strconv.Itoa(i)
			body := testutils.ResponseBody(t, got.res, got.err)
			want := 200
			if op == 2 {
				want = 409
			}
			if op == 11 {
				want = 400
			}
			if got.res.StatusCode != want {
				t.Fatal(i, got.res.StatusCode, body)
			}
			if op != 4 && len(got.res.Cookies()) != 0 {
				t.Fatal("cookies crossed", i)
			}
			if op != 5 && op != 6 && got.res.Header.Get("Cache-Control") != "no-store" {
				t.Fatal("cache crossed", i)
			}
			if op == 1 && body != got.first {
				t.Fatal("replay snapshot changed")
			}
			var env struct{ Data map[string]json.RawMessage }
			if want != 200 {
				if !strings.Contains(body, `"data":[]`) {
					t.Fatal(body)
				}
				if op == 11 {
					for _, field := range []string{"email", "password", "name", "phone", "country_code", "status"} {
						if !strings.Contains(body, `"`+field+`"`) {
							t.Fatal("field crossed", body)
						}
					}
				}
				continue
			}
			if json.Unmarshal([]byte(body), &env) != nil {
				t.Fatal(body)
			}
			switch op {
			case 0, 1:
				var user struct {
					ID             uint64
					Email, Version string
					Roles          []struct{ Name string }
					Apps           []struct{ Name string } `json:"application_access"`
				}
				if len(env.Data) != 1 || json.Unmarshal(env.Data["user"], &user) != nil || user.ID != uint64(i+200) || user.Email != "user"+id+"@example.com" || user.Version != strconv.FormatUint(uint64(9007199254740993+i), 10) || user.Roles[0].Name != "role-"+id || user.Apps[0].Name != "app-"+id {
					t.Fatal("creation response crossed", body)
				}
			case 3:
				if body != shared.RegisterSuccessBody {
					t.Fatal(body)
				}
			case 4:
				if string(env.Data["token"]) != strconv.Quote("token-"+id) || len(got.res.Cookies()) != 2 {
					t.Fatal("login crossed")
				}
				for _, cookie := range got.res.Cookies() {
					if cookie.Name == "__Secure-refresh" && cookie.Value != "refresh-"+id {
						t.Fatal("refresh crossed")
					}
				}
			case 5:
				if body != userfixtures.SuccessBody {
					t.Fatal(body)
				}
			case 6:
				if !strings.Contains(string(env.Data["profile"]), `"name":"self-`+id+`"`) {
					t.Fatal("self crossed", body)
				}
			case 7:
				if string(env.Data["id"]) != fmt.Sprint(i+200) || string(env.Data["name"]) != strconv.Quote("detail-"+id) {
					t.Fatal("detail crossed")
				}
			case 8:
				if string(env.Data["user_id"]) != fmt.Sprint(i+200) || string(env.Data["version"]) != strconv.Quote(strconv.FormatUint(uint64(9007199254740993+i), 10)) {
					t.Fatal("status crossed")
				}
			case 9:
				if !strings.Contains(string(env.Data["items"]), `"device_label":"device-`+id+`"`) {
					t.Fatal("session crossed", body)
				}
			case 10:
				if !strings.Contains(string(env.Data["user"]), `"name":"profile-`+id+`"`) {
					t.Fatal("profile crossed", body)
				}
			}
			assertNoSecrets(t, body, cfg.ServiceToken)
			for _, secret := range []string{"secret-", "private-password", "private-dob", "Bearer owner-", "refresh-"} {
				if strings.Contains(body, secret) {
					t.Fatal("secret crossed", i)
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
			if i%12 == 3 || i%12 == 4 {
				if len(c.metadata.Get("authorization")) != 0 {
					t.Fatal("public bearer")
				}
			} else {
				assertMetadata(t, c.metadata, "authorization", "Bearer owner-"+id)
			}
			if c.tls != tls.VersionTLS13 || c.deadline.IsZero() {
				t.Fatal("transport boundary")
			}
		}
		for i := 0; i < count; i++ {
			op := i % 12
			want := []string{userv1.UserService_Me_FullMethodName, creationMixedMethod(op)}
			if op == 3 || op == 4 {
				want = want[1:]
			}
			if op == 11 {
				want = want[:1]
			}
			if op == 1 {
				want = append(want, userv1.UserService_Me_FullMethodName, creationMixedMethod(op))
			}
			if !reflect.DeepEqual(methods[i], want) {
				t.Fatal("order or duplicate Me", i, methods[i], want)
			}
		}
	})
}
