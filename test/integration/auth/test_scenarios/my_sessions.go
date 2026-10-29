package test_scenarios

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	sessionv1 "github.com/siti-nabila/api-contracts/pb/session/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/rest-orc/internal/client/authservice"
	"github.com/siti-nabila/rest-orc/internal/client/grpcclient"
	clientmocks "github.com/siti-nabila/rest-orc/test/client/authservice/mocks"
	grpcfixtures "github.com/siti-nabila/rest-orc/test/client/grpcclient/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func MySessionsScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "self-session HTTP ordered dispatch authority and backend outcomes", Run: mySessionHTTPOutcomes}, {Name: "self-session HTTP local controls and auth fail before business RPC", Run: mySessionHTTPDenials}}
}
func newMySessionApp(t *testing.T, mode string, check func(context.Context, string), before ...fiber.Handler) *fiber.App {
	t.Helper()
	transport, e := grpcclient.New(grpcfixtures.ValidConfig())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := transport.Close(); e != nil {
			t.Error(e)
		}
	})
	user, _ := rejectingAuthService(t)
	user.MeFunc = func(c context.Context, _ *emptypb.Empty, _ ...grpc.CallOption) (*userv1.UserData, error) {
		check(c, "me")
		if mode == "me failure" {
			return nil, status.Error(codes.Unauthenticated, "private")
		}
		u := fixtures.MeUserData()
		u.MustChangePassword = mode == "restricted"
		return u, nil
	}
	var failure error
	switch mode {
	case "409":
		failure = testutils.ServiceFieldError(t, 409)
	case "429":
		failure = testutils.ServiceFieldError(t, 429)
	case "503":
		failure = testutils.ServiceFieldError(t, 503)
	case "not found":
		failure = status.Error(codes.NotFound, "private")
	case "forbidden":
		failure = status.Error(codes.PermissionDenied, "private")
	case "timeout":
		failure = context.DeadlineExceeded
	case "cancel":
		failure = context.Canceled
	}
	sdk := &clientmocks.SelfSessionService{List: func(c context.Context, r *sessionv1.ListMySessionsRequest) (*sessionv1.ListSessionsResponse, error) {
		check(c, "list")
		q := r.Query
		if q.Page != 1 || q.Limit != 20 || q.LastId != "%2F+ " || q.Fields != nil || q.Search != nil || len(q.Sort) != 2 || q.Sort[0].Field != "created_at" || !q.Sort[0].Desc || q.Sort[1].Field != "id" || !q.Sort[1].Desc {
			t.Fatal("list request")
		}
		if mode == "nil" || failure != nil {
			return nil, failure
		}
		rpl := fixtures.MySessionsResponse(17)
		if mode == "owner mismatch" {
			rpl.Items[0].UserId = 99
		}
		if mode == "empty" {
			rpl.Items = nil
		}
		return rpl, nil
	}, Revoke: func(c context.Context, r *sessionv1.RevokeMySessionRequest) (*sessionv1.RevokeSessionResponse, error) {
		check(c, "one")
		if r.SessionId != fixtures.MySessionID {
			t.Fatal("target changed")
		}
		if mode == "nil" || failure != nil {
			return nil, failure
		}
		return &sessionv1.RevokeSessionResponse{Changed: mode == "true" || mode == "both"}, nil
	}, All: func(c context.Context, r *emptypb.Empty) (*sessionv1.RevokeAllMySessionsResponse, error) {
		check(c, "all")
		if r == nil {
			t.Fatal("Empty required")
		}
		if mode == "nil" || failure != nil {
			return nil, failure
		}
		return &sessionv1.RevokeAllMySessionsResponse{Changed: mode == "true" || mode == "both", LoginRequired: mode == "login" || mode == "both"}, nil
	}}
	client, e := authservice.NewWithSelfSessionService(transport, user, &clientmocks.ProfileService{}, &clientmocks.PasswordService{}, &clientmocks.SessionService{}, sdk)
	if e != nil {
		t.Fatal(e)
	}
	return fixtures.NewApp(t, client, before...)
}
func mySessionHTTPOutcomes(t *testing.T) {
	for _, op := range []string{"list", "one", "all"} {
		for _, mode := range []string{"false", "true", "login", "both", "nil", "empty", "owner mismatch", "not found", "forbidden", "409", "429", "503", "timeout", "cancel"} {
			if op != "list" && (mode == "empty" || mode == "owner mismatch") {
				continue
			}
			t.Run(op+"/"+mode, func(t *testing.T) {
				sequence := []string{}
				parentMD := metadata.Pairs("x-request-id", "self-request", "authorization", "Bearer parent")
				app := newMySessionApp(t, mode, func(c context.Context, method string) {
					sequence = append(sequence, method)
					testutils.AssertServiceContext(t, c)
					md, _ := metadata.FromOutgoingContext(c)
					if !reflect.DeepEqual(md.Get("authorization"), []string{"Bearer self"}) || !reflect.DeepEqual(md.Get("x-request-id"), []string{"self-request"}) {
						t.Fatal("metadata changed")
					}
					for _, key := range []string{"namespace", "x-service-token", "version", "idempotency-key"} {
						if len(md.Get(key)) != 0 {
							t.Fatal("browser claims forwarded")
						}
					}
				}, func(c fiber.Ctx) error {
					c.SetContext(metadata.NewOutgoingContext(c.Context(), parentMD))
					return c.Next()
				})
				method, path := "GET", "/api/v1/me/sessions?last_id=%252F%2B%20&user_id=999&namespace=forged"
				if op == "one" {
					method, path = "DELETE", "/api/v1/me/sessions/"+fixtures.MySessionID
				}
				if op == "all" {
					method, path = "POST", "/api/v1/me/sessions/revoke-all"
				}
				res, body := sessionHTTP(t, app, method, path, `{"user_id":999,"session_id":"forged","version":99}`, nil, map[string]string{"Authorization": "Bearer self", "X-Service-Token": "forged", "Version": "999", "Idempotency-Key": "ignored"})
				want := 200
				switch mode {
				case "nil":
					want = 500
				case "owner mismatch":
					want = 500
				case "not found":
					want = 404
				case "forbidden":
					want = 403
				case "409":
					want = 409
				case "429":
					want = 429
				case "503":
					want = 503
				case "timeout":
					want = 504
				case "cancel":
					want = 499
				}
				if res.StatusCode != want || !reflect.DeepEqual(sequence, []string{"me", op}) {
					t.Fatalf("status %d want %d sequence %v body %s", res.StatusCode, want, sequence, body)
				}
				if want == 200 {
					var env struct {
						Code, Message string
						Data          map[string]json.RawMessage
					}
					if json.Unmarshal([]byte(body), &env) != nil || env.Code != "SS" {
						t.Fatal(body)
					}
					if op == "list" {
						var items []map[string]json.RawMessage
						if json.Unmarshal(env.Data["items"], &items) != nil || string(env.Data["total"]) != "2147483648" || len(env.Data) != 8 || env.Message != "Sessions retrieved successfully." {
							t.Fatal(body)
						}
						if mode == "empty" {
							if string(env.Data["items"]) != "[]" {
								t.Fatal(body)
							}
						} else if len(items) != 1 || len(items[0]) != 11 || string(items[0]["last_seen_at"]) != "null" || string(items[0]["revoked_at"]) != "null" || string(items[0]["is_current"]) != "false" {
							t.Fatal(body)
						}
					} else {
						changed := mode == "true" || mode == "both"
						if string(env.Data["changed"]) != map[bool]string{true: "true", false: "false"}[changed] {
							t.Fatal(body)
						}
						if op == "one" && (len(env.Data) != 1 || env.Message != "Session revoked successfully.") {
							t.Fatal(body)
						}
						if op == "all" {
							login := mode == "login" || mode == "both"
							if len(env.Data) != 2 || string(env.Data["login_required"]) != map[bool]string{true: "true", false: "false"}[login] || env.Message != "All sessions revoked successfully." {
								t.Fatal(body)
							}
						}
					}
				}
				testutils.AssertRefreshCookie(t, res, want == 200 && op == "all" && (mode == "login" || mode == "both"))
				if mode == "409" || mode == "429" || mode == "503" {
					testutils.AssertServiceErrorResponse(t, res, body, want)
				}
				if strings.Contains(body, "private") || strings.Contains(body, "Bearer") {
					t.Fatal("secret leak")
				}
				if !reflect.DeepEqual(parentMD.Get("authorization"), []string{"Bearer parent"}) {
					t.Fatal("parent mutated")
				}
			})
		}
	}
}
func mySessionHTTPDenials(t *testing.T) {
	for _, op := range []struct{ method, path string }{{"GET", "/api/v1/me/sessions"}, {"DELETE", "/api/v1/me/sessions/" + fixtures.MySessionID}, {"POST", "/api/v1/me/sessions/revoke-all"}} {
		for _, mode := range []string{"missing bearer", "me failure", "restricted", "denied origin", "preflight"} {
			sequence := []string{}
			app := newMySessionApp(t, mode, func(_ context.Context, m string) { sequence = append(sequence, m) })
			method := op.method
			headers := map[string]string{"Authorization": "Bearer self"}
			want := 401
			if mode == "missing bearer" {
				delete(headers, "Authorization")
			}
			if mode == "restricted" {
				want = 403
			}
			if mode == "denied origin" {
				headers["Origin"] = "null"
				want = 403
			}
			if mode == "preflight" {
				method = "OPTIONS"
				headers["Access-Control-Request-Method"] = op.method
				want = 204
			}
			res, body := sessionHTTP(t, app, method, op.path, "", []*http.Cookie{{Name: "__Secure-refresh", Value: "ignored"}}, headers)
			wantCalls := 0
			if mode == "me failure" || mode == "restricted" {
				wantCalls = 1
			}
			if res.StatusCode != want || len(sequence) != wantCalls || len(res.Cookies()) != 0 {
				t.Fatalf("denial %s status=%d sequence=%v body=%s", mode, res.StatusCode, sequence, body)
			}
		}
	}
	for _, path := range []string{"/api/v1/me/sessions?page=0", "/api/v1/me/sessions?limit=101", "/api/v1/me/sessions?page=1&page=2", "/api/v1/me/sessions?last_id=" + strings.Repeat("x", 2049), "/api/v1/me/sessions?sort=", "/api/v1/me/sessions?fields=", "/api/v1/me/sessions?search=", "/api/v1/me/sessions/INVALID"} {
		calls := 0
		app := newMySessionApp(t, "false", func(_ context.Context, m string) {
			calls++
			if m != "me" {
				t.Fatal("invalid dispatched")
			}
		})
		method := "GET"
		if strings.HasSuffix(path, "INVALID") {
			method = "DELETE"
		}
		res, body := sessionHTTP(t, app, method, path, "", nil, map[string]string{"Authorization": "Bearer self"})
		if res.StatusCode != 400 || calls != 1 || len(res.Cookies()) != 0 || !strings.Contains(body, `"code":"BR"`) {
			t.Fatalf("local denial %s %d %s", path, res.StatusCode, body)
		}
	}
}
