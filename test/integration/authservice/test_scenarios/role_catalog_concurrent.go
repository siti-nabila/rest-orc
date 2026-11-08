package test_scenarios

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/gofiber/fiber/v3"
	profilev1 "github.com/siti-nabila/api-contracts/pb/profile/v1"
	rolev1 "github.com/siti-nabila/api-contracts/pb/role/v1"
	sessionv1 "github.com/siti-nabila/api-contracts/pb/session/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	roleerrors "github.com/siti-nabila/api-contracts/pkg/dictionary/role"
	"github.com/siti-nabila/api-contracts/pkg/grpcerror"
	"github.com/siti-nabila/api-contracts/pkg/grpcerror/mapping"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	userfixtures "github.com/siti-nabila/rest-orc/test/integration/users/fixtures"
	shared "github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func roleMixedMethod(op int) string {
	switch op {
	case 0, 1:
		return rolev1.RoleService_ListRoles_FullMethodName
	case 2:
		return rolev1.RoleService_GetRole_FullMethodName
	case 3:
		return userv1.UserService_CreateUser_FullMethodName
	case 4:
		return userv1.UserService_Register_FullMethodName
	case 5:
		return userv1.UserService_Login_FullMethodName
	case 6:
		return userv1.UserService_ListUsers_FullMethodName
	case 7:
		return profilev1.ProfileService_UpdateProfile_FullMethodName
	case 8:
		return userv1.UserService_GetUser_FullMethodName
	case 9:
		return userv1.UserService_ChangeUserStatus_FullMethodName
	case 10:
		return sessionv1.SessionService_ListUserSessions_FullMethodName
	case 11:
		return userv1.UserService_UpdateUser_FullMethodName
	}
	return ""
}
func concurrentRole(i int) *rolev1.RoleSummary {
	r := shared.RoleSummary()
	id := strconv.Itoa(i)
	r.Id = uint64(i + 200)
	r.RoleCode = int32(i + 30)
	r.Code = "clm.role_" + id
	r.Name = "role-name-" + id
	r.Version = uint64(9007199254740993 + i)
	r.AffectedUserCount = uint64(i)
	if i < 13 {
		desc := " desc-" + id + "\r\n "
		r.Description = &desc
	} else if i < 26 {
		empty := ""
		r.Description = &empty
	} else {
		r.Description = nil
		r.CreatedAt = nil
		r.UpdatedAt = nil
	}
	return r
}
func roleWireConcurrent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const count = 39
		entered, release := make(chan struct{}, count), make(chan struct{})
		cfg := secureConfig(t)
		h := newTransport(t, cfg, defaultServerTLS(t), func(c context.Context, r any, info *grpc.UnaryServerInfo, next grpc.UnaryHandler) (any, error) {
			md, _ := metadata.FromIncomingContext(c)
			ids := md.Get("x-request-id")
			if len(ids) != 1 {
				return nil, fmt.Errorf("missing request ID")
			}
			id := ids[0]
			i, _ := strconv.Atoi(id)
			op := i % 13
			if info.FullMethod == userv1.UserService_Me_FullMethodName {
				u := shared.MeUserData()
				u.Id = uint64(i + 100)
				u.Permissions = map[string]bool{"iam.role.read": i != 13, "iam.user.create": true, "iam.user.read": true, "iam.user.update": true, "iam.user.status.update": true, "iam.session.read": true}
				if op == 6 {
					u.RoleCodes = []int32{2}
					u.RoleNames = []string{"admin"}
				}
				if op == 12 || i == 13 {
					entered <- struct{}{}
					<-release
				}
				return u, nil
			}
			if info.FullMethod != roleMixedMethod(op) {
				t.Error("method crossed", i)
			}
			switch op {
			case 0, 1:
				q := r.(*rolev1.ListRolesRequest)
				if q.Query.Page != int32(i+1) || q.Query.Limit != 20 || q.Query.LastId != "cursor+/="+id || q.Query.Search == nil || q.Query.Search.Keyword != "term- "+id || int32(q.Query.Search.Mode) != 5 || !reflect.DeepEqual(q.Query.Search.Fields, []string{"keyword"}) || len(q.Query.Fields) != 0 || q.Filter.ApplicationCode != "clm" || len(q.Query.Sort) != 3 {
					t.Error("role query crossed", i)
				}
				for n, field := range []string{"created_at", "updated_at", "id"} {
					if q.Query.Sort[n].Field != field || !q.Query.Sort[n].Desc {
						t.Error("role sort crossed")
					}
				}
				want := []string{"ACTIVE", "DISABLED"}
				if i >= 26 {
					want = nil
				}
				if !reflect.DeepEqual(q.Filter.Statuses, want) || len(md.Get("version")) != 0 || len(md.Get("idempotency-key")) != 0 {
					t.Error("role filters/read metadata crossed")
				}
			case 2:
				q := r.(*rolev1.GetRoleRequest)
				if q.RoleId != uint64(i+200) || len(md.Get("version")) != 0 || len(md.Get("idempotency-key")) != 0 {
					t.Error("role target crossed")
				}
			case 3:
				q := r.(*userv1.CreateUserRequest)
				if q.Email != " User"+id+"@Example.COM " || q.Password != "secret-"+id || q.Name != "Alpha User" || q.Phone != "0628123456789" || q.Address != "Address "+id || q.CountryCode == nil || *q.CountryCode != "+62" || q.Status == nil || *q.Status != "PENDING" || len(md.Get("version")) != 0 {
					t.Error("create crossed")
				}
				assertMetadata(t, md, "idempotency-key", "key, "+id)
			case 4, 5:
				q := r.(*userv1.AuthRequest)
				if q.Email != "user"+id+"@example.com" || q.Password != "secret-"+id {
					t.Error("public crossed")
				}
			case 7:
				q := r.(*profilev1.ProfileRequest)
				if q.Name != "Self User" {
					t.Error("self profile crossed")
				}
			case 8:
				q := r.(*userv1.GetUserRequest)
				if q.Id != uint64(i+200) {
					t.Error("user detail crossed")
				}
			case 9:
				q := r.(*userv1.ChangeUserStatusRequest)
				if q.UserId != uint64(i+200) || q.GetExpectedUserVersion() != uint64(9007199254740993+i) || q.GetStatus() != "SUSPENDED" || q.Reason != "reason "+id {
					t.Error("status crossed")
				}
				assertMetadata(t, md, "idempotency-key", "key, "+id)
			case 10:
				q := r.(*sessionv1.ListUserSessionsRequest)
				if q.UserId != uint64(i+200) || q.Query.LastId != "session-"+id {
					t.Error("sessions crossed")
				}
			case 11:
				q := r.(*userv1.UpdateUserRequest)
				if q.UserId != uint64(i+200) || q.Name == nil || *q.Name != "Target User" || q.GetExpectedUserVersion() != uint64(9007199254740993+i) {
					t.Error("target profile crossed")
				}
				assertMetadata(t, md, "idempotency-key", "key, "+id)
			}
			entered <- struct{}{}
			<-release
			switch op {
			case 0, 1:
				out := shared.RoleList()
				out.Items = []*rolev1.RoleSummary{concurrentRole(i)}
				out.Meta.Total = int64(9007199254740993 + i)
				out.Meta.Page = int32(i + 1)
				out.Meta.HasNext = i%2 == 0
				out.Meta.HasPrev = i > 0
				out.NextCursor = "next-/+" + id
				return out, nil
			case 2:
				if i == 15 {
					return nil, grpcerror.NewEncoder(mapping.RoleCodeMappings()...).Encode(roleerrors.ErrRoleNotFound, locale.English)
				}
				out := shared.RoleDetail()
				out.Role = concurrentRole(i)
				out.Permissions[0].Description = "permission-" + id
				if i == 28 {
					out.Permissions[0].ApplicationCode = "wrong-app"
				}
				return out, nil
			case 3:
				u := shared.UserDetailResponse(uint64(i + 200)).User
				u.Email = "user" + id + "@example.com"
				u.Status = "PENDING"
				u.Version = uint64(9007199254740993 + i)
				return &userv1.CreateUserResponse{User: u}, nil
			case 4:
				return shared.RegistrationResponse(), nil
			case 5:
				u := shared.LoginTokenResponse()
				u.Token = "token-" + id
				u.RefreshToken = "refresh-" + id
				return u, nil
			case 7:
				u := userfixtures.ProfileResponse()
				u.Profile.UserId = uint64(i + 100)
				u.Profile.Name = "self-" + id
				return u, nil
			case 8:
				u := shared.UserDetailResponse(uint64(i + 200))
				u.User.Name = "detail-" + id
				return u, nil
			case 9:
				return &userv1.ChangeUserStatusResponse{UserId: uint64(i + 200), Status: "SUSPENDED", Version: uint64(9007199254740993 + i)}, nil
			case 10:
				u := shared.MySessionsResponse(uint64(i + 200))
				u.Items[0].DeviceLabel = "device-" + id
				return u, nil
			case 11:
				u := shared.UserDetailResponse(uint64(i + 200)).User
				u.Name = "profile-" + id
				return &userv1.UpdateUserResponse{User: u}, nil
			}
			return next(c, r)
		})
		defer h.close(t)
		app := shared.NewApp(t, h.client, func(c fiber.Ctx) error {
			md := metadata.Pairs("x-request-id", c.Get("X-Request-ID"), "version", "inherited", "idempotency-key", "inherited")
			c.SetContext(metadata.NewOutgoingContext(c.Context(), md))
			return c.Next()
		})
		type result struct {
			i   int
			res *http.Response
			err error
		}
		results := make(chan result, count)
		for i := 0; i < count; i++ {
			go func(i int) {
				id, op := strconv.Itoa(i), i%13
				method, path, body := "GET", "/api/v1/roles", ""
				switch op {
				case 0, 1:
					if op == 1 {
						path += "/options"
					}
					values := url.Values{"page": {strconv.Itoa(i + 1)}, "limit": {"20"}, "keyword": {" term-\t" + id + " "}, "application_code": {"clm"}, "last_id": {"cursor+/=" + id}, "namespace": {"forged"}}
					if i < 26 {
						values.Set("statuses", " DISABLED,ACTIVE,ACTIVE ")
					}
					path += "?" + values.Encode()
				case 2:
					path = fmt.Sprintf("/api/v1/roles/0%d", i+200)
				case 3:
					method, path = "POST", "/api/v1/users"
					body = `{"email":" User` + id + `@Example.COM ","password":"secret-` + id + `","name":" alpha  user ","phone":"00628123456789","country_code":"62","address":"Address ` + id + `"}`
				case 4, 5:
					method, path = "POST", "/api/v1/users/create"
					if op == 5 {
						path = "/api/v1/users/login"
					}
					body = `{"email":"user` + id + `@example.com","password":"secret-` + id + `"}`
				case 6:
					path = "/api/v1/users"
				case 7:
					method, path, body = "PATCH", "/api/v1/me/profile", `{"name":" self  user "}`
				case 8:
					path = fmt.Sprintf("/api/v1/users/%d", i+200)
				case 9:
					method, path, body = "PATCH", fmt.Sprintf("/api/v1/users/%d/status", i+200), `{"status":"SUSPENDED","reason":" reason `+id+` "}`
				case 10:
					path = fmt.Sprintf("/api/v1/users/%d/sessions?last_id=session-%s", i+200, id)
				case 11:
					method, path, body = "PATCH", fmt.Sprintf("/api/v1/users/%d/profile", i+200), `{"name":" target  user "}`
				case 12:
					path = "/api/v1/roles?page=0&limit=101&application_code=Bad&statuses=bad&keyword=%00bad-secret"
				}
				req := httptest.NewRequest(method, path, strings.NewReader(body))
				req.Header.Set("Authorization", "Bearer owner-"+id)
				req.Header.Set("X-Request-ID", id)
				req.Header.Set(locale.HTTPHeader, []string{"en", "id"}[i%2])
				req.Header.Set("Version", strconv.FormatUint(uint64(9007199254740993+i), 10))
				req.Header.Set("Idempotency-Key", "key, "+id)
				req.Header.Set("Origin", shared.BrowserOrigin)
				req.Header.Set("Content-Type", "application/json")
				res, err := app.Test(req, fiber.TestConfig{})
				results <- result{i, res, err}
			}(i)
		}
		for range count {
			<-entered
		}
		close(release)
		for range count {
			got := <-results
			i, op := got.i, got.i%13
			id := strconv.Itoa(i)
			body := testutils.ResponseBody(t, got.res, got.err)
			want := 200
			if i == 13 {
				want = 403
			}
			if i == 15 {
				want = 404
			}
			if i == 28 {
				want = 500
			}
			if op == 12 {
				want = 400
			}
			if got.res.StatusCode != want {
				t.Fatal(i, got.res.StatusCode, want, body)
			}
			if op != 5 && len(got.res.Cookies()) != 0 {
				t.Fatal("cookie crossed")
			}
			if op != 6 && op != 7 && got.res.Header.Get("Cache-Control") != "no-store" {
				t.Fatal("cache crossed")
			}
			if want != 200 {
				if !strings.Contains(body, `"data":[]`) {
					t.Fatal(body)
				}
				if op == 12 {
					var env struct{ Errors map[string][]string }
					_ = json.Unmarshal([]byte(body), &env)
					if len(env.Errors) != 5 || strings.Contains(body, "bad-secret") {
						t.Fatal("field errors crossed", body)
					}
				}
				continue
			}
			var env struct{ Data map[string]json.RawMessage }
			if json.Unmarshal([]byte(body), &env) != nil {
				t.Fatal(body)
			}
			switch op {
			case 0, 1, 2:
				var rawRole map[string]json.RawMessage
				if op == 2 {
					if len(env.Data) != 2 {
						t.Fatal("detail shape")
					}
					_ = json.Unmarshal(env.Data["role"], &rawRole)
					if !strings.Contains(string(env.Data["permissions"]), `"description":"permission-`+id+`"`) {
						t.Fatal("permissions crossed")
					}
				} else {
					if len(env.Data) != 8 || string(env.Data["total"]) != strconv.FormatInt(int64(9007199254740993+i), 10) || string(env.Data["page"]) != fmt.Sprint(i+1) || string(env.Data["next_cursor"]) != strconv.Quote("next-/+"+id) || string(env.Data["has_next"]) != strconv.FormatBool(i%2 == 0) || string(env.Data["has_prev"]) != strconv.FormatBool(i > 0) {
						t.Fatal("role page crossed", body)
					}
					var items []map[string]json.RawMessage
					_ = json.Unmarshal(env.Data["items"], &items)
					if len(items) != 1 {
						t.Fatal("role items")
					}
					rawRole = items[0]
				}
				if op == 1 {
					label := "role-name-" + id
					if i < 13 {
						label = " desc-" + id + "\r\n "
					}
					if len(rawRole) != 4 || string(rawRole["value"]) != strconv.Quote(fmt.Sprint(i+200)) || string(rawRole["role_code"]) != strconv.Quote(fmt.Sprint(i+30)) || string(rawRole["code"]) != strconv.Quote("clm.role_"+id) || string(rawRole["label"]) != strconv.Quote("clm.role_"+id+" - "+label) {
						t.Fatal("option crossed", body)
					}
				} else {
					if len(rawRole) != 13 || string(rawRole["id"]) != fmt.Sprint(i+200) || string(rawRole["role_code"]) != fmt.Sprint(i+30) || string(rawRole["version"]) != strconv.Quote(strconv.FormatUint(uint64(9007199254740993+i), 10)) || string(rawRole["name"]) != strconv.Quote("role-name-"+id) || string(rawRole["affected_user_count"]) != fmt.Sprint(i) {
						t.Fatal("role metadata crossed", body)
					}
					if i >= 26 && (string(rawRole["description"]) != "null" || string(rawRole["created_at"]) != "null" || string(rawRole["updated_at"]) != "null") {
						t.Fatal("nullable metadata crossed", body)
					}
				}
			case 3:
				if !strings.Contains(string(env.Data["user"]), `"email":"user`+id+`@example.com"`) || !strings.Contains(string(env.Data["user"]), `"id":`+fmt.Sprint(i+200)+`,`) || !strings.Contains(string(env.Data["user"]), `"version":"`+strconv.FormatUint(uint64(9007199254740993+i), 10)+`"`) {
					t.Fatal("created user crossed")
				}
			case 4:
				if body != shared.RegisterSuccessBody {
					t.Fatal(body)
				}
			case 5:
				if string(env.Data["token"]) != strconv.Quote("token-"+id) || len(got.res.Cookies()) != 2 {
					t.Fatal("login crossed")
				}
				for _, cookie := range got.res.Cookies() {
					if cookie.Name == "__Secure-refresh" && cookie.Value != "refresh-"+id {
						t.Fatal("refresh crossed")
					}
				}
			case 6:
				if body != userfixtures.SuccessBody {
					t.Fatal(body)
				}
			case 7:
				if !strings.Contains(string(env.Data["profile"]), `"name":"self-`+id+`"`) {
					t.Fatal("self crossed")
				}
			case 8:
				if string(env.Data["id"]) != fmt.Sprint(i+200) || string(env.Data["name"]) != strconv.Quote("detail-"+id) {
					t.Fatal("detail crossed")
				}
			case 9:
				if string(env.Data["user_id"]) != fmt.Sprint(i+200) || string(env.Data["version"]) != strconv.Quote(strconv.FormatUint(uint64(9007199254740993+i), 10)) {
					t.Fatal("status crossed")
				}
			case 10:
				if !strings.Contains(string(env.Data["items"]), `"device_label":"device-`+id+`"`) {
					t.Fatal("session crossed")
				}
			case 11:
				if !strings.Contains(string(env.Data["user"]), `"name":"profile-`+id+`"`) {
					t.Fatal("profile crossed")
				}
			}
			assertNoSecrets(t, body, cfg.ServiceToken)
			for _, secret := range []string{"secret-", "private-password", "private-dob", "Bearer owner-", "refresh-"} {
				if strings.Contains(body, secret) {
					t.Fatal("secret response", i)
				}
			}
		}
		calls, dials := h.snapshot()
		if dials != 1 {
			t.Fatal("connection reuse")
		}
		methods := map[int][]string{}
		for _, c := range calls {
			id := c.metadata.Get("x-request-id")[0]
			i, _ := strconv.Atoi(id)
			methods[i] = append(methods[i], c.method)
			assertMetadata(t, c.metadata, "x-service-token", cfg.ServiceToken)
			assertMetadata(t, c.metadata, locale.MetadataKey, []string{"en", "id"}[i%2])
			if i%13 == 4 || i%13 == 5 {
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
			op := i % 13
			want := []string{userv1.UserService_Me_FullMethodName, roleMixedMethod(op)}
			if op == 4 || op == 5 {
				want = want[1:]
			}
			if op == 12 || i == 13 {
				want = want[:1]
			}
			if !reflect.DeepEqual(methods[i], want) {
				t.Fatal("call order/duplicate Me", i, methods[i], want)
			}
		}
	})
}
