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
	"google.golang.org/grpc/metadata"
)

func adminSessionsWireConcurrent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const count = 28
		entered, release := make(chan struct{}, count), make(chan struct{})
		cfg := secureConfig(t)
		h := newTransport(t, cfg, defaultServerTLS(t), func(ctx context.Context, r any, info *grpc.UnaryServerInfo, next grpc.UnaryHandler) (any, error) {
			md, _ := metadata.FromIncomingContext(ctx)
			ids := md.Get("x-request-id")
			if len(ids) != 1 {
				return nil, fmt.Errorf("missing request ID")
			}
			i, err := strconv.Atoi(ids[0])
			if err != nil {
				return nil, err
			}
			id := ids[0]
			op := i % 7
			denied := i >= 21 && op < 3
			if info.FullMethod == userv1.UserService_Me_FullMethodName {
				u := shared.MeUserData()
				u.Id = uint64(i + 100)
				u.Permissions = map[string]bool{"iam.session.read": true, "iam.session.revoke": true}
				if denied {
					u.Permissions = map[string]bool{"iam.session.*": true}
					entered <- struct{}{}
					<-release
				}
				if op == 6 {
					u.RoleCodes = []int32{2}
					u.RoleNames = []string{"admin"}
				}
				return u, nil
			}
			switch op {
			case 0:
				request := r.(*sessionv1.ListUserSessionsRequest)
				if request.UserId != uint64(i+200) || request.Query.LastId != "cursor-"+id {
					t.Error("list target/cursor crossed")
				}
				if len(md.Get("idempotency-key")) != 0 || len(md.Get("version")) != 0 {
					t.Error("list precondition leaked")
				}
			case 1:
				request := r.(*sessionv1.RevokeUserSessionRequest)
				if request.UserId != uint64(i+200) || request.SessionId != concurrentSessionID(i) {
					t.Error("single target crossed")
				}
				assertMetadata(t, md, "idempotency-key", "key, "+id)
			case 2:
				request := r.(*sessionv1.RevokeAllUserSessionsRequest)
				if request.UserId != uint64(i+200) || request.ExpectedUserVersion == nil || *request.ExpectedUserVersion != uint64(9007199254740993+i) {
					t.Error("all target/version crossed")
				}
				assertMetadata(t, md, "idempotency-key", "key, "+id)
			case 3:
				if r.(*sessionv1.ListMySessionsRequest).Query.LastId != "cursor-"+id {
					t.Error("self cursor crossed")
				}
			case 4, 5:
				request := r.(*userv1.AuthRequest)
				if request.Email != "user"+id+"@example.com" || request.Password != "valid-password" {
					t.Error("public payload crossed")
				}
			}
			entered <- struct{}{}
			<-release
			switch op {
			case 0, 3:
				owner := uint64(i + 200)
				if op == 3 {
					owner = uint64(i + 100)
				}
				page := shared.MySessionsResponse(owner)
				page.NextCursor = "next-" + id
				page.Items[0].Id = concurrentSessionID(i)
				page.Items[0].DeviceLabel = "device-" + id
				page.Items[0].IsCurrent = i%2 == 0
				if i == 14 {
					page.Items[0].UserId = 999
				}
				return page, nil
			case 1:
				if i == 8 {
					return nil, testutils.ServiceFieldError(t, 409)
				}
				return &sessionv1.RevokeSessionResponse{Changed: i%2 == 0}, nil
			case 2:
				return &sessionv1.RevokeAllUserSessionsResponse{Changed: i%2 == 0, Version: uint64(9007199254740900 + i)}, nil
			case 4:
				return shared.RegistrationResponse(), nil
			case 5:
				result := shared.LoginTokenResponse()
				result.Token = "token-" + id
				result.RefreshToken = "refresh-" + id
				return result, nil
			default:
				return next(ctx, r)
			}
		})
		defer h.close(t)
		app := shared.NewApp(t, h.client, func(c fiber.Ctx) error {
			c.SetContext(metadata.NewOutgoingContext(c.Context(), metadata.Pairs("x-request-id", c.Get("X-Request-ID"))))
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
				op := i % 7
				method, path, body := "GET", fmt.Sprintf("/api/v1/users/%d/sessions?last_id=cursor-%s", i+200, id), "{"
				switch op {
				case 1:
					method, path = "DELETE", fmt.Sprintf("/api/v1/users/%d/sessions/%s", i+200, concurrentSessionID(i))
				case 2:
					method, path = "POST", fmt.Sprintf("/api/v1/users/%d/sessions/revoke-all", i+200)
				case 3:
					path = "/api/v1/me/sessions?last_id=cursor-" + id
				case 4:
					method, path = "POST", "/api/v1/users/create"
				case 5:
					method, path = "POST", "/api/v1/users/login"
				case 6:
					path = "/api/v1/users"
				}
				if op == 4 || op == 5 {
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
			body := testutils.ResponseBody(t, got.res, got.err)
			i := got.i
			id := strconv.Itoa(i)
			op := i % 7
			want := 200
			if i >= 21 && op < 3 {
				want = 403
			}
			if i == 8 {
				want = 409
			}
			if i == 14 {
				want = 500
			}
			if got.res.StatusCode != want {
				t.Fatalf("request %d status %d want %d %s", i, got.res.StatusCode, want, body)
			}
			if op != 6 && got.res.Header.Get("Cache-Control") != "no-store" {
				t.Fatal("cache")
			}
			if op != 5 && len(got.res.Cookies()) != 0 {
				t.Fatal("cookies crossed")
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
			case 0, 3:
				var items []struct {
					ID      string `json:"id"`
					UserID  uint64 `json:"user_id"`
					Device  string `json:"device_label"`
					Current bool   `json:"is_current"`
				}
				owner := uint64(i + 200)
				if op == 3 {
					owner = uint64(i + 100)
				}
				if json.Unmarshal(env.Data["items"], &items) != nil || len(items) != 1 || items[0].ID != concurrentSessionID(i) || items[0].UserID != owner || items[0].Device != "device-"+id || items[0].Current != (i%2 == 0) || string(env.Data["next_cursor"]) != `"next-`+id+`"` {
					t.Fatal("result crossed", body)
				}
			case 1:
				if len(env.Data) != 1 || string(env.Data["changed"]) != strconv.FormatBool(i%2 == 0) {
					t.Fatal(body)
				}
			case 2:
				if len(env.Data) != 2 || string(env.Data["changed"]) != strconv.FormatBool(i%2 == 0) || string(env.Data["version"]) != `"`+strconv.FormatUint(uint64(9007199254740900+i), 10)+`"` {
					t.Fatal(body)
				}
			case 4:
				if body != shared.RegisterSuccessBody {
					t.Fatal(body)
				}
			case 5:
				if string(env.Data["token"]) != `"token-`+id+`"` || len(got.res.Cookies()) != 2 {
					t.Fatal("login crossed", body)
				}
				for _, c := range got.res.Cookies() {
					if c.Name == "__Secure-refresh" && c.Value != "refresh-"+id {
						t.Fatal("refresh crossed")
					}
				}
			case 6:
				if body != userfixtures.SuccessBody {
					t.Fatal(body)
				}
			}
			assertNoSecrets(t, body, cfg.ServiceToken)
			if strings.Contains(body, "Bearer owner-") || strings.Contains(body, "refresh-") {
				t.Fatal("secret leak")
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
			op := i % 7
			methods[i] = append(methods[i], c.method)
			assertMetadata(t, c.metadata, "x-service-token", cfg.ServiceToken)
			assertMetadata(t, c.metadata, locale.MetadataKey, []string{"en", "id"}[i%2])
			if op == 4 || op == 5 {
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
			op := i % 7
			sequence := methods[i]
			wantMethod := ""
			switch op {
			case 0, 1, 2:
				wantMethod = adminSessionsWireMethod([]string{"list", "one", "all"}[op])
			case 3:
				wantMethod = mySessionsWireMethod("list")
			case 4:
				wantMethod = userv1.UserService_Register_FullMethodName
			case 5:
				wantMethod = userv1.UserService_Login_FullMethodName
			case 6:
				wantMethod = userv1.UserService_ListUsers_FullMethodName
			}
			n := 2
			if op == 4 || op == 5 || i >= 21 && op < 3 {
				n = 1
			}
			if len(sequence) != n {
				t.Fatal("retries/extra lookup", i, sequence)
			}
			if op == 4 || op == 5 {
				if sequence[0] != wantMethod {
					t.Fatal("public order")
				}
			} else {
				if sequence[0] != userv1.UserService_Me_FullMethodName || n == 2 && sequence[1] != wantMethod {
					t.Fatal("protected order", sequence)
				}
			}
		}
	})
}
