package test_scenarios

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v3"
	applicationv1 "github.com/siti-nabila/api-contracts/pb/application/v1"
	sessionv1 "github.com/siti-nabila/api-contracts/pb/session/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	usersfixtures "github.com/siti-nabila/rest-orc/test/integration/users/fixtures"
	shared "github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"testing/synctest"
)

func accessMixedMethod(op int) string {
	switch op {
	case 0:
		return accessExpectedMethod("list")
	case 1:
		return accessExpectedMethod("grant")
	case 2:
		return accessExpectedMethod("revoke")
	case 3:
		return applicationv1.ApplicationService_ListApplications_FullMethodName
	case 4:
		return userv1.UserService_GrantUserRoles_FullMethodName
	case 5:
		return userv1.UserService_GetUser_FullMethodName
	case 6:
		return userv1.UserService_Login_FullMethodName
	case 7:
		return userv1.UserService_ChangeUserStatus_FullMethodName
	case 8:
		return userv1.UserService_UpdateUser_FullMethodName
	case 9:
		return sessionv1.SessionService_ListUserSessions_FullMethodName
	case 10:
		return userv1.UserService_CreateUser_FullMethodName
	case 11:
		return userv1.UserService_ListUsers_FullMethodName
	}
	return ""
}
func accessWireConcurrent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const count = 48
		entered, release := make(chan struct{}, count), make(chan struct{})
		cfg := secureConfig(t)
		h := newTransport(t, cfg, defaultServerTLS(t), func(c context.Context, r any, info *grpc.UnaryServerInfo, _ grpc.UnaryHandler) (any, error) {
			md, _ := metadata.FromIncomingContext(c)
			raw := md.Get("x-request-id")
			if len(raw) != 1 {
				return nil, fmt.Errorf("request ID missing")
			}
			id := raw[0]
			i, _ := strconv.Atoi(id)
			op := i % 12
			target, role, version := uint64(300+i), uint64(400+i), uint64(9007199254740993+i)
			code := "app_" + id
			if info.FullMethod == userv1.UserService_Me_FullMethodName {
				u := shared.MeUserData()
				u.Id = uint64(100 + i)
				u.Version = 1
				u.Permissions = map[string]bool{"iam.user.read": i != 12, "iam.user.application.grant": true, "iam.user.application.revoke": true, "iam.user.role.grant": i != 13, "iam.user.role.revoke": true, "iam.permission.read": true, "iam.user.status.update": true, "iam.user.update": true, "iam.session.read": true, "iam.user.create": true}
				if op == 11 {
					u.RoleCodes = []int32{2}
					u.RoleNames = []string{"admin"}
				}
				if i == 12 || i == 13 {
					entered <- struct{}{}
					<-release
				}
				return u, nil
			}
			if info.FullMethod != accessMixedMethod(op) {
				t.Error("method crossed", i, info.FullMethod)
			}
			mutation := op == 1 || op == 2 || op == 4 || op == 7 || op == 8 || op == 10
			if op != 6 && op != 11 {
				if len(md.Get("version")) != 0 {
					t.Error("version leaked", i)
				}
				if mutation {
					assertMetadata(t, md, "idempotency-key", "key, "+id)
				} else if len(md.Get("idempotency-key")) != 0 {
					t.Error("read key leaked", i)
				}
			}
			switch op {
			case 0:
				q := r.(*userv1.ListUserApplicationAccessRequest)
				if q.UserId != target || q.Query.Page != int32(i+1) || q.Query.Limit != 20 || q.Query.LastId != "cursor+/="+id || q.Query.Search == nil || q.Query.Search.Keyword != "term "+id || int32(q.Query.Search.Mode) != 1 || !reflect.DeepEqual(q.Query.Search.Fields, []string{"application_code", "name"}) || len(q.Query.Fields) != 0 || len(q.Query.Sort) != 1 || q.Query.Sort[0].Field != "application_id" || q.Query.Sort[0].Desc || q.Filter.ApplicationCode != code {
					t.Error("access query crossed", i, q)
				}
				if !reflect.DeepEqual(q.Filter.Statuses, []string{"ACTIVE", "REVOKED"}) {
					t.Error("states crossed")
				}
			case 1:
				q := r.(*applicationv1.GrantApplicationAccessRequest)
				if q.UserId != target || q.ApplicationCode != code || !reflect.DeepEqual(q.RoleIds, []uint64{role}) || q.Reason != "reason "+id || q.ExpectedUserVersion == nil || *q.ExpectedUserVersion != version {
					t.Error("access grant crossed", i, q)
				}
			case 2:
				q := r.(*applicationv1.RevokeApplicationAccessRequest)
				if q.UserId != target || q.ApplicationCode != code || q.Reason != "reason "+id || q.ExpectedUserVersion == nil || *q.ExpectedUserVersion != version {
					t.Error("access revoke crossed", i, q)
				}
			case 3:
				q := r.(*applicationv1.ListApplicationsRequest)
				if q.Query.LastId != "catalog-"+id || q.Query.Search.Keyword != "term "+id || int32(q.Query.Search.Mode) != 5 {
					t.Error("catalog query crossed", i, q)
				}
			case 4:
				q := r.(*userv1.GrantUserRolesRequest)
				if q.UserId != target || !reflect.DeepEqual(q.RoleIds, []uint64{role}) || q.Reason != "reason "+id || q.GetExpectedUserVersion() != version {
					t.Error("selected roles crossed", i, q)
				}
			case 5:
				if r.(*userv1.GetUserRequest).Id != target {
					t.Error("detail crossed")
				}
			case 6:
				q := r.(*userv1.AuthRequest)
				if q.Email != "user"+id+"@example.com" || q.Password != "secret-"+id {
					t.Error("public crossed", i, q)
				}
			case 7:
				q := r.(*userv1.ChangeUserStatusRequest)
				if q.UserId != target || q.GetStatus() != "SUSPENDED" || q.GetExpectedUserVersion() != version || q.Reason != "reason "+id {
					t.Error("status crossed", i, q)
				}
			case 8:
				q := r.(*userv1.UpdateUserRequest)
				if q.UserId != target || q.GetExpectedUserVersion() != version || q.GetName() != "ALPHA USER" || !reflect.DeepEqual(q.FieldMask.Paths, []string{"name"}) {
					t.Error("profile crossed", i, q)
				}
			case 9:
				q := r.(*sessionv1.ListUserSessionsRequest)
				if q.UserId != target || q.Query.LastId != "session-"+id {
					t.Error("session crossed", i, q)
				}
			case 10:
				q := r.(*userv1.CreateUserRequest)
				if q.Email != "new"+id+"@example.com" || q.GetStatus() != "PENDING" {
					t.Error("create crossed", i, q)
				}
			case 11:
				if r.(*userv1.ListUsersRequest).Query.LastId != strconv.Itoa(i+1) {
					t.Error("legacy list crossed", i, r)
				}
			}
			entered <- struct{}{}
			<-release
			if i == 15 {
				return nil, status.Error(codes.PermissionDenied, "private-sentinel")
			}
			if i == 25 {
				return nil, testutils.ServiceFieldError(t, 429)
			}
			switch op {
			case 0:
				out := shared.UserAccessHistory(target)
				out.Version = version
				out.Items[0].ApplicationId = uint64(600 + i)
				out.Items[0].ApplicationCode = code
				out.Items[0].Name = "history-" + id
				out.Items[0].IsSystem = i >= 24
				out.Meta.Total = int64(1<<32 + i)
				out.Meta.Page = int32(i + 1)
				out.NextCursor = "next-" + id
				if i == 24 {
					out.Items[0].GrantedAt = nil
				}
				if i >= 36 {
					s := " old reason-" + id + "\té "
					actor := uint64(100 + i)
					out.Items[0].RevokeReason = &s
					out.Items[0].RevokedBy = &actor
				}
				return out, nil
			case 1:
				ids := []uint64{role}
				if i >= 36 {
					ids = []uint64{}
				}
				return &applicationv1.GrantApplicationAccessResponse{UserId: target, ApplicationId: uint64(600 + i), AccessStatus: accessWireStatus("ACTIVE"), Version: version, Changed: true, ChangedRoleIds: ids}, nil
			case 2:
				ids := []uint64{}
				changed := i != 14
				state := accessWireStatus("REVOKED")
				if i == 14 || i == 26 {
					state = nil
				}
				if i == 26 {
					for k := 0; k < 25; k++ {
						ids = append(ids, role+uint64(k))
					}
				}
				return &applicationv1.RevokeApplicationAccessResponse{UserId: target, ApplicationId: uint64(600 + i), AccessStatus: state, ChangedRoleIds: ids, Version: version, Changed: changed}, nil
			case 3:
				out := shared.ApplicationList()
				out.Items[0].Code = code
				out.Items[0].Name = "catalog-" + id
				return out, nil
			case 4:
				return &userv1.GrantUserRolesResponse{UserId: target, Version: version, Changed: true, ChangedRoleIds: []uint64{role}}, nil
			case 5:
				out := shared.UserDetailResponse(target)
				out.User.Name = "detail-" + id
				return out, nil
			case 6:
				out := shared.LoginTokenResponse()
				out.Token = "token-" + id
				out.RefreshToken = "refresh-" + id
				return out, nil
			case 7:
				return &userv1.ChangeUserStatusResponse{UserId: target, Status: "SUSPENDED", Version: version, Changed: true}, nil
			case 8:
				out := shared.UserDetailResponse(target).User
				out.Name = "profile-" + id
				out.Version = version
				return &userv1.UpdateUserResponse{User: out, Changed: true}, nil
			case 9:
				out := shared.MySessionsResponse(target)
				out.Items[0].DeviceLabel = "device-" + id
				return out, nil
			case 10:
				out := shared.UserDetailResponse(uint64(900 + i)).User
				out.Email = "new" + id + "@example.com"
				out.Status = "PENDING"
				out.Name = "created-" + id
				return &userv1.CreateUserResponse{User: out}, nil
			case 11:
				return usersfixtures.ListUsersResponse(), nil
			}
			return nil, status.Error(codes.Internal, "unexpected method")
		})
		defer h.close(t)
		app := shared.NewApp(t, h.client, func(c fiber.Ctx) error {
			md := metadata.Pairs("x-request-id", c.Get("X-Request-ID"), "version", "old", "idempotency-key", "old")
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
				id, op := strconv.Itoa(i), i%12
				target := 300 + i
				code := "app_" + id
				method, path, body := "GET", "", ""
				switch op {
				case 0:
					values := url.Values{"page": {strconv.Itoa(i + 1)}, "keyword": {" term\t" + id + " "}, "last_id": {"cursor+/=" + id}, "application_code": {code}, "statuses": {" REVOKED,ACTIVE,ACTIVE "}}
					path = fmt.Sprintf("/api/v1/users/%d/applications?%s", target, values.Encode())
				case 1, 2:
					method = "POST"
					suffix := "grant"
					if op == 2 {
						suffix = "revoke"
					}
					path = fmt.Sprintf("/api/v1/users/%d/applications/%s/%s", target, code, suffix)
					body = fmt.Sprintf(`{"role_ids":["%d"],"reason":" reason %d "}`, 400+i, i)
				case 3:
					path = "/api/v1/applications?last_id=catalog-" + id + "&keyword=term%20" + id
				case 4:
					method = "POST"
					path = fmt.Sprintf("/api/v1/users/%d/roles/grant", target)
					body = fmt.Sprintf(`{"role_ids":["%d"],"reason":" reason %d "}`, 400+i, i)
				case 5:
					path = fmt.Sprintf("/api/v1/users/%d", target)
				case 6:
					method, path = "POST", "/api/v1/users/login"
					body = `{"email":"user` + id + `@example.com","password":"secret-` + id + `"}`
				case 7:
					method = "PATCH"
					path = fmt.Sprintf("/api/v1/users/%d/status", target)
					body = fmt.Sprintf(`{"status":"SUSPENDED","reason":" reason %d "}`, i)
				case 8:
					method = "PATCH"
					path = fmt.Sprintf("/api/v1/users/%d/profile", target)
					body = `{"name":"ALPHA USER"}`
				case 9:
					path = fmt.Sprintf("/api/v1/users/%d/sessions?last_id=session-%s", target, id)
				case 10:
					method, path = "POST", "/api/v1/users"
					body = `{"email":"new` + id + `@example.com","password":"valid-password","name":"alpha user","phone":"812345678901","country_code":"+62"}`
				case 11:
					path = "/api/v1/users?last_id=" + strconv.Itoa(i+1)
				}
				req := httptest.NewRequest(method, path, strings.NewReader(body))
				req.Header.Set("Authorization", "Bearer owner-"+id)
				req.Header.Set("X-Request-ID", id)
				req.Header.Set(locale.HTTPHeader, []string{"en", "id", "zh"}[i%3])
				req.Header.Set("Version", strconv.FormatUint(uint64(9007199254740993+i), 10))
				req.Header.Set("Idempotency-Key", "key, "+id)
				req.Header.Set("Origin", shared.BrowserOrigin)
				req.Header.Set("Content-Type", "application/json")
				res, e := app.Test(req, fiber.TestConfig{})
				results <- result{i, res, e}
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
			if i == 12 || i == 13 || i == 15 {
				want = 403
			}
			if i == 24 {
				want = 500
			}
			if i == 25 {
				want = 429
			}
			if got.res.StatusCode != want {
				t.Fatal(i, got.res.StatusCode, body)
			}
			if op != 11 && got.res.Header.Get("Cache-Control") != "no-store" {
				t.Fatal("cache crossed", i)
			}
			if op != 6 && len(got.res.Cookies()) != 0 {
				t.Fatal("cookie crossed", i)
			}
			if strings.Contains(body, "private-sentinel") {
				t.Fatal(body)
			}
			if want != 200 {
				if !strings.Contains(body, `"data":[]`) {
					t.Fatal(body)
				}
				if i == 25 {
					testutils.AssertServiceErrorResponse(t, got.res, body, 429)
				}
				continue
			}
			var env struct{ Data map[string]json.RawMessage }
			if json.Unmarshal([]byte(body), &env) != nil {
				t.Fatal(body)
			}
			switch op {
			case 0:
				var items []map[string]json.RawMessage
				_ = json.Unmarshal(env.Data["items"], &items)
				if len(env.Data) != 10 || len(items) != 1 || len(items[0]) != 11 || string(items[0]["application_code"]) != `"app_`+id+`"` || string(items[0]["name"]) != `"history-`+id+`"` || string(env.Data["user_id"]) != strconv.Itoa(300+i) || string(env.Data["version"]) != `"`+strconv.FormatUint(uint64(9007199254740993+i), 10)+`"` || string(env.Data["total"]) != strconv.FormatInt(int64(1<<32+i), 10) || string(env.Data["next_cursor"]) != `"next-`+id+`"` {
					t.Fatal("history crossed", i, body)
				}
				if i >= 36 {
					var reason string
					_ = json.Unmarshal(items[0]["revoke_reason"], &reason)
					if reason != " old reason-"+id+"\té " {
						t.Fatal(body)
					}
				} else if string(items[0]["revoke_reason"]) != "null" {
					t.Fatal(body)
				}
			case 1, 2:
				if len(env.Data) != 6 || string(env.Data["user_id"]) != strconv.Itoa(300+i) || string(env.Data["application_id"]) != strconv.Itoa(600+i) || string(env.Data["version"]) != `"`+strconv.FormatUint(uint64(9007199254740993+i), 10)+`"` {
					t.Fatal("compound crossed", i, body)
				}
				var ids []uint64
				_ = json.Unmarshal(env.Data["changed_role_ids"], &ids)
				wantIDs := 0
				if op == 1 && i < 36 {
					wantIDs = 1
				}
				if i == 26 {
					wantIDs = 25
				}
				if len(ids) != wantIDs || string(env.Data["changed"]) != strconv.FormatBool(i != 14) {
					t.Fatal(body)
				}
				state := `"ACTIVE"`
				if op == 2 {
					state = `"REVOKED"`
				}
				if i == 14 || i == 26 {
					state = "null"
				}
				if string(env.Data["access_status"]) != state {
					t.Fatal(body)
				}
			case 3:
				if !strings.Contains(body, `"code":"app_`+id+`"`) {
					t.Fatal(body)
				}
			case 4:
				if string(env.Data["changed_role_ids"]) != "["+strconv.Itoa(400+i)+"]" {
					t.Fatal(body)
				}
			case 5:
				if !strings.Contains(body, `"name":"detail-`+id+`"`) {
					t.Fatal(body)
				}
			case 6:
				if len(got.res.Cookies()) != 2 || !strings.Contains(body, "token-"+id) {
					t.Fatal(body)
				}
			case 7:
				if string(env.Data["user_id"]) != strconv.Itoa(300+i) {
					t.Fatal(body)
				}
			case 8:
				if !strings.Contains(body, `"name":"profile-`+id+`"`) {
					t.Fatal(body)
				}
			case 9:
				if !strings.Contains(body, `"device_label":"device-`+id+`"`) {
					t.Fatal(body)
				}
			case 10:
				if !strings.Contains(body, `"name":"created-`+id+`"`) {
					t.Fatal(body)
				}
			}
		}
		calls, dials := h.snapshot()
		if dials != 1 {
			t.Fatal("connection reuse", dials)
		}
		byID := map[string][]string{}
		for _, call := range calls {
			id := call.metadata.Get("x-request-id")[0]
			i, _ := strconv.Atoi(id)
			byID[id] = append(byID[id], call.method)
			if call.tls != tls.VersionTLS13 || call.deadline.IsZero() {
				t.Fatal("TLS deadline")
			}
			assertMetadata(t, call.metadata, "x-service-token", cfg.ServiceToken)
			assertMetadata(t, call.metadata, locale.MetadataKey, []string{"en", "id", "zh"}[i%3])
			if i%12 != 6 {
				assertMetadata(t, call.metadata, "authorization", "Bearer owner-"+id)
			}
		}
		for i := 0; i < count; i++ {
			want := []string{userv1.UserService_Me_FullMethodName, accessMixedMethod(i % 12)}
			if i == 12 || i == 13 {
				want = want[:1]
			}
			if i%12 == 6 {
				want = want[1:]
			}
			if !reflect.DeepEqual(byID[strconv.Itoa(i)], want) {
				t.Fatal("order retry", i, byID[strconv.Itoa(i)], want)
			}
		}
	})
}
