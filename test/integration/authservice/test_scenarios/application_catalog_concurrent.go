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
	applicationv1 "github.com/siti-nabila/api-contracts/pb/application/v1"
	rolev1 "github.com/siti-nabila/api-contracts/pb/role/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	shared "github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func applicationMixedMethod(op int) string {
	switch op {
	case 0, 1:
		return applicationv1.ApplicationService_ListApplications_FullMethodName
	case 2:
		return rolev1.RoleService_ListRoles_FullMethodName
	case 3:
		return userv1.UserService_GetUser_FullMethodName
	case 4:
		return userv1.UserService_GrantUserRoles_FullMethodName
	case 5:
		return userv1.UserService_RevokeUserRoles_FullMethodName
	case 6:
		return userv1.UserService_Login_FullMethodName
	case 7:
		return userv1.UserService_ListUserRoles_FullMethodName
	}
	return ""
}
func applicationWireConcurrent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const count = 24
		entered, release := make(chan struct{}, count), make(chan struct{})
		cfg := secureConfig(t)
		h := newTransport(t, cfg, defaultServerTLS(t), func(c context.Context, r any, info *grpc.UnaryServerInfo, _ grpc.UnaryHandler) (any, error) {
			md, _ := metadata.FromIncomingContext(c)
			ids := md.Get("x-request-id")
			if len(ids) != 1 {
				return nil, fmt.Errorf("missing request ID")
			}
			id := ids[0]
			i, _ := strconv.Atoi(id)
			op := i % 8
			target, role, version := uint64(300+i), uint64(400+i), uint64(9007199254740993+i)
			if info.FullMethod == userv1.UserService_Me_FullMethodName {
				u := shared.MeUserData()
				u.Id = uint64(100 + i)
				u.Permissions = map[string]bool{"iam.permission.read": i != 8, "iam.role.read": true, "iam.user.read": true, "iam.user.role.grant": true, "iam.user.role.revoke": true}
				if i == 8 {
					entered <- struct{}{}
					<-release
				}
				return u, nil
			}
			if info.FullMethod != applicationMixedMethod(op) {
				t.Error("method crossed", i, info.FullMethod)
			}
			if op != 6 {
				if len(md.Get("version")) != 0 {
					t.Error("version metadata crossed")
				}
				if op == 4 || op == 5 {
					assertMetadata(t, md, "idempotency-key", "key, "+id)
				} else if len(md.Get("idempotency-key")) != 0 {
					t.Error("read key crossed")
				}
			}
			switch op {
			case 0, 1:
				q := r.(*applicationv1.ListApplicationsRequest)
				if q.Query.Page != int32(i+1) || q.Query.Limit != 20 || q.Query.LastId != "cursor+/="+id || q.Query.Search == nil || q.Query.Search.Keyword != "term "+id || !reflect.DeepEqual(q.Query.Search.Fields, []string{"keyword"}) || int32(q.Query.Search.Mode) != 5 || len(q.Query.Fields) != 0 || len(q.Query.Sort) != 1 || q.Query.Sort[0].Field != "id" || !q.Query.Sort[0].Desc {
					t.Error("application query crossed", i, q)
				}
				want := []string{"ACTIVE", "DISABLED"}
				if i >= 16 {
					want = nil
				}
				if !reflect.DeepEqual(q.Filter.Statuses, want) {
					t.Error("status crossed", i, q.Filter)
				}
			case 2:
				if r.(*rolev1.ListRolesRequest).Query.LastId != "role-"+id {
					t.Error("role cursor crossed")
				}
			case 3:
				if r.(*userv1.GetUserRequest).Id != target {
					t.Error("detail target crossed")
				}
			case 4:
				q := r.(*userv1.GrantUserRolesRequest)
				if q.UserId != target || !reflect.DeepEqual(q.RoleIds, []uint64{role}) || q.Reason != "reason "+id || q.ExpectedUserVersion == nil || *q.ExpectedUserVersion != version {
					t.Error("grant crossed", i, q)
				}
			case 5:
				q := r.(*userv1.RevokeUserRolesRequest)
				if q.UserId != target || !reflect.DeepEqual(q.RoleIds, []uint64{role}) || q.Reason != "reason "+id || q.ExpectedUserVersion == nil || *q.ExpectedUserVersion != version {
					t.Error("revoke crossed", i, q)
				}
			case 6:
				q := r.(*userv1.AuthRequest)
				if q.Email != "user"+id+"@example.com" || q.Password != "secret-"+id {
					t.Error("login crossed")
				}
			case 7:
				q := r.(*userv1.ListUserRolesRequest)
				if q.UserId != target || q.Query.LastId != "assignment-"+id {
					t.Error("assignment target crossed")
				}
			}
			entered <- struct{}{}
			<-release
			if i == 10 {
				return nil, status.Error(codes.PermissionDenied, "private-sentinel")
			}
			switch op {
			case 0, 1:
				out := shared.ApplicationList()
				out.Items[0].Id = uint64(200 + i)
				out.Items[0].Code = "app_" + id
				out.Items[0].Name = " name-" + id + "\té\r\n "
				out.Items[0].IsSystem = i%2 == 0
				if i == 1 {
					out.Items[0].Name = ""
				}
				if i >= 16 {
					out.Items[0].Status = "ACTIVE"
				}
				out.Meta.Total = int64(1<<32 + i)
				out.Meta.Page = int32(i + 1)
				out.Meta.HasNext = i < 16
				out.Meta.HasPrev = i > 0
				out.NextCursor = "next-/+" + id
				if i == 16 {
					out.Items = nil
				}
				if i == 17 {
					out.Items[0].Status = "REVOKED"
				}
				return out, nil
			case 2:
				out := shared.RoleList()
				out.Items[0].Name = "role-" + id
				return out, nil
			case 3:
				out := shared.UserDetailResponse(target)
				out.User.Name = "detail-" + id
				return out, nil
			case 4:
				return &userv1.GrantUserRolesResponse{UserId: target, Version: version, Changed: true, ChangedRoleIds: []uint64{role}}, nil
			case 5:
				ids := []uint64{}
				if i < 16 {
					ids = []uint64{role}
				}
				return &userv1.RevokeUserRolesResponse{UserId: target, Version: version, Changed: len(ids) > 0, ChangedRoleIds: ids}, nil
			case 6:
				out := shared.LoginTokenResponse()
				out.Token = "token-" + id
				out.RefreshToken = "refresh-" + id
				return out, nil
			case 7:
				out := shared.UserRoleList(target)
				out.Items[0].Name = "assignment-" + id
				return out, nil
			}
			return nil, status.Error(codes.Internal, "unexpected method")
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
			e   error
		}
		results := make(chan result, count)
		for i := 0; i < count; i++ {
			go func(i int) {
				id, op := strconv.Itoa(i), i%8
				method, path, body := "GET", "/api/v1/applications", ""
				switch op {
				case 0, 1:
					if op == 1 {
						path += "/options"
					}
					values := url.Values{"page": {strconv.Itoa(i + 1)}, "keyword": {" term\t" + id + " "}, "last_id": {"cursor+/=" + id}, "namespace": {"forged"}}
					if i < 16 {
						values.Set("statuses", " DISABLED,ACTIVE,ACTIVE ")
					}
					path += "?" + values.Encode()
				case 2:
					path = "/api/v1/roles?last_id=role-" + id
				case 3:
					path = fmt.Sprintf("/api/v1/users/%d", 300+i)
				case 4, 5:
					method = "POST"
					suffix := "grant"
					if op == 5 {
						suffix = "revoke"
					}
					path = fmt.Sprintf("/api/v1/users/%d/roles/%s", 300+i, suffix)
					body = fmt.Sprintf(`{"role_ids":["%d"],"reason":" reason %d "}`, 400+i, i)
				case 6:
					method, path = "POST", "/api/v1/users/login"
					body = `{"email":"user` + id + `@example.com","password":"secret-` + id + `"}`
				case 7:
					path = fmt.Sprintf("/api/v1/users/%d/roles?last_id=assignment-%s", 300+i, id)
				}
				req := httptest.NewRequest(method, path, strings.NewReader(body))
				req.Header.Set("Authorization", "Bearer owner-"+id)
				req.Header.Set("X-Request-ID", id)
				req.Header.Set(locale.HTTPHeader, []string{"en", "id"}[i%2])
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
			i, op := got.i, got.i%8
			id := strconv.Itoa(i)
			body := testutils.ResponseBody(t, got.res, got.e)
			want := 200
			if i == 8 || i == 10 {
				want = 403
			}
			if i == 17 {
				want = 500
			}
			if got.res.StatusCode != want || got.res.Header.Get("Cache-Control") != "no-store" {
				t.Fatal(i, want, got.res.StatusCode, body)
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
				continue
			}
			var env struct{ Data map[string]json.RawMessage }
			if json.Unmarshal([]byte(body), &env) != nil {
				t.Fatal(body)
			}
			switch op {
			case 0, 1:
				if len(env.Data) != 8 || string(env.Data["total"]) != strconv.FormatInt(int64(1<<32+i), 10) || string(env.Data["page"]) != strconv.Itoa(i+1) || string(env.Data["next_cursor"]) != `"next-/+`+id+`"` {
					t.Fatal("page crossed", i, body)
				}
				var items []map[string]json.RawMessage
				_ = json.Unmarshal(env.Data["items"], &items)
				if i == 16 {
					if string(env.Data["items"]) != "[]" {
						t.Fatal(body)
					}
					continue
				}
				if op == 0 {
					if len(items[0]) != 5 || string(items[0]["code"]) != `"app_`+id+`"` || string(items[0]["is_system"]) != strconv.FormatBool(i%2 == 0) {
						t.Fatal(body)
					}
				} else {
					name := " name-" + id + "\té\r\n "
					if i == 1 {
						name = ""
					}
					label, _ := json.Marshal("app_" + id + " - " + name)
					if len(items[0]) != 2 || string(items[0]["value"]) != `"app_`+id+`"` || string(items[0]["label"]) != string(label) {
						t.Fatal(body)
					}
				}
			case 2:
				if !strings.Contains(body, `"name":"role-`+id+`"`) {
					t.Fatal(body)
				}
			case 3:
				if !strings.Contains(body, `"name":"detail-`+id+`"`) {
					t.Fatal(body)
				}
			case 4, 5:
				if string(env.Data["user_id"]) != strconv.Itoa(300+i) || string(env.Data["version"]) != `"`+strconv.FormatUint(uint64(9007199254740993+i), 10)+`"` {
					t.Fatal(body)
				}
				ids := "[" + strconv.Itoa(400+i) + "]"
				if op == 5 && i >= 16 {
					ids = "[]"
				}
				if string(env.Data["changed_role_ids"]) != ids {
					t.Fatal(body)
				}
			case 6:
				if len(got.res.Cookies()) != 2 || !strings.Contains(body, "token-"+id) {
					t.Fatal("login cookies crossed", body)
				}
			case 7:
				if !strings.Contains(body, `"name":"assignment-`+id+`"`) {
					t.Fatal(body)
				}
			}
		}
		calls, dials := h.snapshot()
		if dials != 1 {
			t.Fatal(dials)
		}
		byID := map[string][]string{}
		for _, c := range calls {
			id := c.metadata.Get("x-request-id")[0]
			i, _ := strconv.Atoi(id)
			byID[id] = append(byID[id], c.method)
			if c.tls != tls.VersionTLS13 || c.deadline.IsZero() {
				t.Fatal("TLS/deadline")
			}
			assertMetadata(t, c.metadata, "x-service-token", cfg.ServiceToken)
			assertMetadata(t, c.metadata, locale.MetadataKey, []string{"en", "id"}[i%2])
			if i%8 != 6 {
				assertMetadata(t, c.metadata, "authorization", "Bearer owner-"+id)
			}
		}
		for i := 0; i < count; i++ {
			want := []string{userv1.UserService_Me_FullMethodName, applicationMixedMethod(i % 8)}
			if i == 8 {
				want = want[:1]
			}
			if i%8 == 6 {
				want = want[1:]
			}
			if !reflect.DeepEqual(byID[strconv.Itoa(i)], want) {
				t.Fatal("order/retry", i, byID[strconv.Itoa(i)], want)
			}
		}
	})
}
