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

func assignmentMixedMethod(op int) string {
	switch op {
	case 0:
		return assignmentExpectedMethod("list")
	case 1:
		return assignmentExpectedMethod("grant")
	case 2:
		return assignmentExpectedMethod("revoke")
	case 3:
		return userv1.UserService_GetUser_FullMethodName
	case 4:
		return rolev1.RoleService_ListRoles_FullMethodName
	case 5:
		return userv1.UserService_Login_FullMethodName
	}
	return ""
}
func assignmentWireConcurrent(t *testing.T) {
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
			op := i % 6
			if info.FullMethod == userv1.UserService_Me_FullMethodName {
				u := shared.MeUserData()
				u.Id = uint64(100 + i)
				u.Permissions = map[string]bool{"iam.user.read": i != 6, "iam.user.role.grant": true, "iam.user.role.revoke": true, "iam.role.read": true}
				if i == 6 {
					entered <- struct{}{}
					<-release
				}
				return u, nil
			}
			if info.FullMethod != assignmentMixedMethod(op) {
				t.Error("method crossed", i, info.FullMethod)
			}
			target, role, version := uint64(300+i), uint64(200+i), uint64(9007199254740993+i)
			if op <= 4 {
				if len(md.Get("version")) != 0 {
					t.Error("version metadata crossed")
				}
				if op == 1 || op == 2 {
					assertMetadata(t, md, "idempotency-key", "key, "+id)
				} else if len(md.Get("idempotency-key")) != 0 {
					t.Error("read key crossed")
				}
			}
			switch op {
			case 0:
				q := r.(*userv1.ListUserRolesRequest)
				if q.UserId != target || q.Query.Page != int32(i+1) || q.Query.LastId != "cursor+/="+id || q.Query.Search.Keyword != "term "+id || !reflect.DeepEqual(q.Query.Search.Fields, []string{"keyword"}) || int32(q.Query.Search.Mode) != 5 || len(q.Query.Sort) != 1 || q.Query.Sort[0].Field != "id" || q.Query.Sort[0].Desc || q.Filter.ApplicationCode != "clm" {
					t.Error("list query crossed", i, q)
				}
				want := []string{"ACTIVE", "REVOKED"}
				if i >= 12 {
					want = nil
				}
				if !reflect.DeepEqual(q.Filter.Statuses, want) {
					t.Error("states crossed", i, q.Filter.Statuses)
				}
			case 1:
				q := r.(*userv1.GrantUserRolesRequest)
				if q.UserId != target || !reflect.DeepEqual(q.RoleIds, []uint64{role}) || q.Reason != "reason "+id || q.ExpectedUserVersion == nil || *q.ExpectedUserVersion != version {
					t.Error("grant crossed", i, q)
				}
			case 2:
				q := r.(*userv1.RevokeUserRolesRequest)
				if q.UserId != target || !reflect.DeepEqual(q.RoleIds, []uint64{role}) || q.Reason != "reason "+id || q.ExpectedUserVersion == nil || *q.ExpectedUserVersion != version {
					t.Error("revoke crossed", i, q)
				}
			case 3:
				if r.(*userv1.GetUserRequest).Id != target {
					t.Error("detail crossed")
				}
			case 4:
				if r.(*rolev1.ListRolesRequest).Query.LastId != "catalog-"+id {
					t.Error("catalog crossed")
				}
			case 5:
				q := r.(*userv1.AuthRequest)
				if q.Email != "user"+id+"@example.com" || q.Password != "secret-"+id {
					t.Error("login crossed")
				}
			}
			entered <- struct{}{}
			<-release
			if i == 7 {
				return nil, status.Error(codes.PermissionDenied, "private-sentinel")
			}
			switch op {
			case 0:
				out := shared.UserRoleList(target)
				out.Version = version
				out.Meta.Total = int64(1<<32 + i)
				out.Meta.Page = int32(i + 1)
				out.NextCursor = "next-" + id
				out.Items[0].Id = uint64(500 + i)
				out.Items[0].RoleId = role
				out.Items[0].Name = "assignment-" + id
				out.Items[0].IsSystem = i >= 12
				if i >= 12 {
					out.Items[0].GrantedAt = nil
					reason := " revoked-" + id + "\r\n "
					out.Items[0].RevokeReason = &reason
				}
				return out, nil
			case 1:
				return &userv1.GrantUserRolesResponse{UserId: target, Version: version, Changed: true, ChangedRoleIds: []uint64{role}}, nil
			case 2:
				ids := []uint64{}
				if i < 12 {
					ids = []uint64{role}
				}
				return &userv1.RevokeUserRolesResponse{UserId: target, Version: version, Changed: len(ids) > 0, ChangedRoleIds: ids}, nil
			case 3:
				out := shared.UserDetailResponse(target)
				out.User.Name = "detail-" + id
				return out, nil
			case 4:
				out := shared.RoleList()
				out.Items[0].Name = "catalog-" + id
				return out, nil
			case 5:
				out := shared.LoginTokenResponse()
				out.Token = "token-" + id
				out.RefreshToken = "refresh-" + id
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
				id := strconv.Itoa(i)
				op := i % 6
				method, path, body := "GET", fmt.Sprintf("/api/v1/users/%d/roles", 300+i), ""
				switch op {
				case 0:
					q := url.Values{"page": {strconv.Itoa(i + 1)}, "keyword": {" term\t" + id + " "}, "application_code": {"clm"}, "last_id": {"cursor+/=" + id}}
					if i < 12 {
						q.Set("statuses", " REVOKED,ACTIVE,ACTIVE ")
					}
					path += "?" + q.Encode()
				case 1, 2:
					method = "POST"
					suffix := "grant"
					if op == 2 {
						suffix = "revoke"
					}
					path += "/" + suffix
					body = fmt.Sprintf(`{"role_ids":["%d"],"reason":" reason %d "}`, 200+i, i)
				case 3:
					path = fmt.Sprintf("/api/v1/users/%d", 300+i)
				case 4:
					path = "/api/v1/roles?last_id=catalog-" + id
				case 5:
					method, path = "POST", "/api/v1/users/login"
					body = `{"email":"user` + id + `@example.com","password":"secret-` + id + `"}`
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
			i, op := got.i, got.i%6
			id := strconv.Itoa(i)
			body := testutils.ResponseBody(t, got.res, got.e)
			want := 200
			if i == 6 || i == 7 {
				want = 403
			}
			if got.res.StatusCode != want || got.res.Header.Get("Cache-Control") != "no-store" {
				t.Fatal(i, want, got.res.StatusCode, body)
			}
			if op != 5 && len(got.res.Cookies()) != 0 {
				t.Fatal("cookie crossed", i)
			}
			if strings.Contains(body, "private-sentinel") {
				t.Fatal(body)
			}
			if want != 200 {
				continue
			}
			var env struct{ Data map[string]json.RawMessage }
			if json.Unmarshal([]byte(body), &env) != nil {
				t.Fatal(body)
			}
			if op <= 2 {
				if string(env.Data["user_id"]) != strconv.Itoa(300+i) || string(env.Data["version"]) != `"`+strconv.FormatUint(uint64(9007199254740993+i), 10)+`"` {
					t.Fatal("target/version crossed", i, body)
				}
			}
			switch op {
			case 0:
				if !strings.Contains(body, `"name":"assignment-`+id+`"`) || !strings.Contains(body, `"next_cursor":"next-`+id+`"`) {
					t.Fatal(body)
				}
				if i >= 12 && !strings.Contains(body, `"granted_at":null`) {
					t.Fatal("null crossed", body)
				}
			case 1:
				if string(env.Data["changed_role_ids"]) != "["+strconv.Itoa(200+i)+"]" {
					t.Fatal(body)
				}
			case 2:
				ids := "[]"
				if i < 12 {
					ids = "[" + strconv.Itoa(200+i) + "]"
				}
				if string(env.Data["changed_role_ids"]) != ids {
					t.Fatal(body)
				}
			case 3:
				if !strings.Contains(body, `"name":"detail-`+id+`"`) {
					t.Fatal(body)
				}
			case 4:
				if !strings.Contains(body, `"name":"catalog-`+id+`"`) {
					t.Fatal(body)
				}
			case 5:
				if len(got.res.Cookies()) != 2 || !strings.Contains(body, "token-"+id) {
					t.Fatal("login cookies/tokens crossed", body)
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
			op := i % 6
			byID[id] = append(byID[id], c.method)
			if c.tls != tls.VersionTLS13 || c.deadline.IsZero() {
				t.Fatal("TLS/deadline")
			}
			assertMetadata(t, c.metadata, "x-service-token", cfg.ServiceToken)
			assertMetadata(t, c.metadata, locale.MetadataKey, []string{"en", "id"}[i%2])
			if op != 5 {
				assertMetadata(t, c.metadata, "authorization", "Bearer owner-"+id)
			}
		}
		for i := 0; i < count; i++ {
			want := []string{userv1.UserService_Me_FullMethodName, assignmentMixedMethod(i % 6)}
			if i == 6 {
				want = want[:1]
			}
			if i%6 == 5 {
				want = want[1:]
			}
			if !reflect.DeepEqual(byID[strconv.Itoa(i)], want) {
				t.Fatal("order/retry", i, byID[strconv.Itoa(i)], want)
			}
		}
	})
}
