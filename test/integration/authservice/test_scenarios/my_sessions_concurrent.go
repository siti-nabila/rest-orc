package test_scenarios

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v3"
	sessionv1 "github.com/siti-nabila/api-contracts/pb/session/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	shared "github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"testing/synctest"
)

func concurrentSessionID(i int) string { return fmt.Sprintf("00000000-0000-0000-0000-%012d", i) }
func mySessionsWireConcurrent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const count = 16
		entered, release := make(chan struct{}, count), make(chan struct{})
		cfg := secureConfig(t)
		h := newTransport(t, cfg, defaultServerTLS(t), func(ctx context.Context, r any, info *grpc.UnaryServerInfo, _ grpc.UnaryHandler) (any, error) {
			md, _ := metadata.FromIncomingContext(ctx)
			ids := md.Get("x-request-id")
			if len(ids) != 1 {
				return nil, fmt.Errorf("missing request ID")
			}
			i, e := strconv.Atoi(ids[0])
			if e != nil {
				return nil, e
			}
			id := ids[0]
			if info.FullMethod == userv1.UserService_Me_FullMethodName && i%4 != 3 {
				u := shared.MeUserData()
				u.Id = uint64(i + 100)
				return u, nil
			}
			op := []string{"list", "one", "all", "me"}[i%4]
			want := mySessionsWireMethod(op)
			if op == "me" {
				want = userv1.UserService_Me_FullMethodName
			}
			if info.FullMethod != want {
				t.Error("method crossed requests")
			}
			switch op {
			case "list":
				q := r.(*sessionv1.ListMySessionsRequest).Query
				if q.LastId != "cursor-"+id {
					t.Error("cursor crossed requests")
				}
			case "one":
				if r.(*sessionv1.RevokeMySessionRequest).SessionId != concurrentSessionID(i) {
					t.Error("target crossed requests")
				}
			}
			entered <- struct{}{}
			<-release
			switch op {
			case "list":
				page := shared.MySessionsResponse(uint64(i + 100))
				page.NextCursor = "next-" + id
				page.Items[0].Id = concurrentSessionID(i)
				page.Items[0].DeviceLabel = "device-" + id
				page.Items[0].IsCurrent = i%8 == 0
				return page, nil
			case "one":
				return &sessionv1.RevokeSessionResponse{Changed: i%8 == 1}, nil
			case "all":
				return &sessionv1.RevokeAllMySessionsResponse{Changed: i%8 == 2, LoginRequired: i%8 == 6}, nil
			default:
				u := shared.MeUserData()
				u.Id = uint64(i + 100)
				return u, nil
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
				method, path := "GET", "/api/v1/me/sessions?last_id=cursor-"+id
				switch i % 4 {
				case 1:
					method, path = "DELETE", "/api/v1/me/sessions/"+concurrentSessionID(i)
				case 2:
					method, path = "POST", "/api/v1/me/sessions/revoke-all"
				case 3:
					path = "/api/v1/me"
				}
				req := httptest.NewRequest(method, path, nil)
				req.Header.Set("Authorization", "Bearer owner-"+id)
				req.Header.Set("X-Request-ID", id)
				req.Header.Set(locale.HTTPHeader, []string{"en", "id"}[i%2])
				res, e := app.Test(req, fiber.TestConfig{})
				results <- outcome{i, res, e}
			}(i)
		}
		for range count {
			<-entered
		}
		close(release)
		for range count {
			got := <-results
			body := testutils.ResponseBody(t, got.res, got.err)
			if got.res.StatusCode != 200 {
				t.Fatalf("concurrent status=%d %s", got.res.StatusCode, body)
			}
			id := strconv.Itoa(got.i)
			var env struct{ Data map[string]json.RawMessage }
			if json.Unmarshal([]byte(body), &env) != nil {
				t.Fatal(body)
			}
			switch got.i % 4 {
			case 0:
				var items []struct {
					ID      string `json:"id"`
					UserID  uint64 `json:"user_id"`
					Device  string `json:"device_label"`
					Current bool   `json:"is_current"`
				}
				if json.Unmarshal(env.Data["items"], &items) != nil || len(items) != 1 || items[0].ID != concurrentSessionID(got.i) || items[0].UserID != uint64(got.i+100) || items[0].Device != "device-"+id || items[0].Current != (got.i%8 == 0) || string(env.Data["next_cursor"]) != `"next-`+id+`"` {
					t.Fatal("list crossed requests", body)
				}
			case 1:
				if string(env.Data["changed"]) != strconv.FormatBool(got.i%8 == 1) || len(env.Data) != 1 {
					t.Fatal("single flags crossed")
				}
			case 2:
				if string(env.Data["changed"]) != strconv.FormatBool(got.i%8 == 2) || string(env.Data["login_required"]) != strconv.FormatBool(got.i%8 == 6) {
					t.Fatal("all flags crossed")
				}
			case 3:
				if string(env.Data["id"]) != strconv.Itoa(got.i+100) {
					t.Fatal("Me owner crossed")
				}
			}
			testutils.AssertRefreshCookie(t, got.res, got.i%4 == 2 && got.i%8 == 6)
			if got.i%4 != 3 && got.res.Header.Get("Cache-Control") != "no-store" {
				t.Fatal("missing no-store")
			}
			assertNoSecrets(t, body, cfg.ServiceToken)
			if strings.Contains(body, "Bearer owner-") {
				t.Fatal("bearer leaked")
			}
		}
		calls, dials := h.snapshot()
		if len(calls) != count+3*count/4 || dials != 1 {
			t.Fatal("calls/dials", len(calls), dials)
		}
		methods := map[string][]string{}
		for _, c := range calls {
			id := c.metadata.Get("x-request-id")[0]
			i, _ := strconv.Atoi(id)
			methods[id] = append(methods[id], c.method)
			assertMetadata(t, c.metadata, "authorization", "Bearer owner-"+id)
			assertMetadata(t, c.metadata, "x-service-token", cfg.ServiceToken)
			assertMetadata(t, c.metadata, locale.MetadataKey, []string{"en", "id"}[i%2])
			if c.tls != tls.VersionTLS13 || c.deadline.IsZero() {
				t.Fatal("transport boundary changed")
			}
		}
		for i := 0; i < count; i++ {
			sequence := methods[strconv.Itoa(i)]
			n := 2
			if i%4 == 3 {
				n = 1
			}
			if len(sequence) != n || sequence[0] != userv1.UserService_Me_FullMethodName {
				t.Fatal("per-request Me order")
			}
			if n == 2 && sequence[1] != mySessionsWireMethod([]string{"list", "one", "all"}[i%4]) {
				t.Fatal("per-request RPC order")
			}
		}
	})
}
