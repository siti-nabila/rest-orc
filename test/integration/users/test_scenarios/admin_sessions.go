package test_scenarios

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	sessionv1 "github.com/siti-nabila/api-contracts/pb/session/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	"github.com/siti-nabila/rest-orc/internal/client/authservice"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	clientmocks "github.com/siti-nabila/rest-orc/test/client/authservice/mocks"
	shared "github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func AdminSessionScenarios() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "caller retry rechecks backend authority conflict and saved result", Run: adminHTTPReplayAuthority},
		{Name: "admin sessions one Me one target RPC exact outcomes and backend replay", Run: adminHTTPOutcomes},
		{Name: "admin session authority validation and route coexistence", Run: adminHTTPGuards},
	}
}

func adminRequest(op string) *http.Request {
	method, path := "GET", "/api/v1/users/00023/sessions?last_id=%252F%2B%20"
	if op == "one" {
		method, path = "DELETE", "/api/v1/users/23/sessions/"+shared.MySessionID
	}
	if op == "all" {
		method, path = "POST", "/api/v1/users/23/sessions/revoke-all"
	}
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	req := httptest.NewRequest(method, path+sep+"user_id=99&version=99&namespace=forged", strings.NewReader("{"))
	req.Header.Set("Authorization", authorizationValue)
	req.Header.Set("Idempotency-Key", "opaque, key")
	req.Header.Set("Version", "9223372036854775807")
	req.Header.Set(locale.HTTPHeader, "id-ID")
	req.Header.Set("X-Service-Token", "forged-secret")
	req.AddCookie(&http.Cookie{Name: "__Secure-refresh", Value: "forged-cookie"})
	return req
}

func adminClientApp(t *testing.T, user authservice.Service, sdk authservice.AdminSessionService, before ...fiber.Handler) *fiber.App {
	t.Helper()
	client, err := authservice.NewWithAdminSessionService(newTransport(t), user, &clientmocks.ProfileService{}, &clientmocks.PasswordService{}, &clientmocks.SessionService{}, &clientmocks.SelfSessionService{}, sdk)
	if err != nil {
		t.Fatal(err)
	}
	return shared.NewApp(t, client, before...)
}

func adminHTTPOutcomes(t *testing.T) {
	for _, op := range []string{"list", "one", "all"} {
		for _, mode := range []string{"success", "changed", "target self", "replay", "nil", "invalid result", "owner mismatch", "empty page", "401", "403", "404", "409", "429", "503", "504", "499", "500"} {
			t.Run(op+"/"+mode, func(t *testing.T) {
				calls := []string{}
				target := uint64(23)
				if mode == "target self" {
					target = 17
				}
				want := 200
				var failure error
				switch mode {
				case "nil", "invalid result":
					want = 500
				case "owner mismatch":
					if op == "list" {
						want = 500
					}
				case "401":
					want = 401
					failure = common.ErrUnauthorized
				case "403":
					want = 403
					failure = status.Error(codes.PermissionDenied, "private-sentinel")
				case "404":
					want = 404
					failure = status.Error(codes.NotFound, "private-sentinel")
				case "409", "429", "503":
					fmt.Sscan(mode, &want)
					failure = testutils.ServiceFieldError(t, want)
				case "504":
					want = 504
					failure = context.DeadlineExceeded
				case "499":
					want = 499
					failure = context.Canceled
				case "500":
					want = 500
					failure = status.Error(codes.Internal, "private-sentinel")
				}
				user := &mocks.Service{MeFunc: func(c context.Context, _ *emptypb.Empty, _ ...grpc.CallOption) (*userv1.UserData, error) {
					calls = append(calls, "Me")
					assertAuthorizationMetadata(t, c)
					u := shared.MeUserData()
					u.Permissions = map[string]bool{domain.SessionReadPermission: true, domain.SessionRevokePermission: true}
					return u, nil
				}}
				check := func(c context.Context, method string) {
					t.Helper()
					calls = append(calls, method)
					assertAuthorizationMetadata(t, c)
					md, _ := metadata.FromOutgoingContext(c)
					keys := md.Get("idempotency-key")
					if method == "list" {
						if len(keys) != 0 {
							t.Error("list inherited key")
						}
					} else if !reflect.DeepEqual(keys, []string{"opaque, key"}) {
						t.Error("mutation key", keys)
					}
					if len(md.Get("version")) != 0 || len(md.Get("x-service-token")) != 0 || len(md.Get("namespace")) != 0 {
						t.Error("authority leaked", md)
					}
				}
				sdk := &clientmocks.AdminSessionService{List: func(c context.Context, r *sessionv1.ListUserSessionsRequest) (*sessionv1.ListSessionsResponse, error) {
					check(c, "list")
					q := r.Query
					if r.UserId != target || q.Page != 1 || q.Limit != 20 || q.LastId != "%2F+ " || q.Fields != nil || q.Search != nil || len(q.Sort) != 2 || q.Sort[0].Field != "created_at" || !q.Sort[0].Desc || q.Sort[1].Field != "id" || !q.Sort[1].Desc {
						t.Error("request", r)
					}
					if failure != nil || mode == "nil" {
						return nil, failure
					}
					result := shared.MySessionsResponse(target)
					if mode == "invalid result" {
						result.Meta = nil
					}
					if mode == "owner mismatch" {
						result.Items[0].UserId = 99
					}
					if mode == "empty page" {
						result.Items = nil
					}
					return result, nil
				},
					Revoke: func(c context.Context, r *sessionv1.RevokeUserSessionRequest) (*sessionv1.RevokeSessionResponse, error) {
						check(c, "one")
						if r.UserId != target || r.SessionId != shared.MySessionID {
							t.Error(r)
						}
						if failure != nil || mode == "nil" || mode == "invalid result" {
							return nil, failure
						}
						return &sessionv1.RevokeSessionResponse{Changed: mode == "changed"}, nil
					},
					All: func(c context.Context, r *sessionv1.RevokeAllUserSessionsRequest) (*sessionv1.RevokeAllUserSessionsResponse, error) {
						check(c, "all")
						if r.UserId != target || r.ExpectedUserVersion == nil || *r.ExpectedUserVersion != 9223372036854775807 {
							t.Error(r)
						}
						if failure != nil || mode == "nil" {
							return nil, failure
						}
						version := uint64(9007199254740993)
						if mode == "invalid result" {
							version = 0
						}
						return &sessionv1.RevokeAllUserSessionsResponse{Changed: mode == "changed", Version: version}, nil
					}}
				parentMD := metadata.Pairs("idempotency-key", "inherited", "version", "88", "x-request-id", "admin-http")
				parent := metadata.NewOutgoingContext(context.Background(), parentMD)
				app := adminClientApp(t, user, sdk, func(c fiber.Ctx) error { c.SetContext(parent); return c.Next() })
				repeats := 1
				if mode == "replay" {
					repeats = 2
				}
				for i := 0; i < repeats; i++ {
					req := adminRequest(op)
					if target == 17 {
						req.URL.Path = strings.Replace(req.URL.Path, "/23/", "/17/", 1)
						req.URL.Path = strings.Replace(req.URL.Path, "/00023/", "/17/", 1)
					}
					res, err := app.Test(req)
					body := testutils.ResponseBody(t, res, err)
					if res.StatusCode != want || res.Header.Get("Cache-Control") != "no-store" || len(res.Cookies()) != 0 {
						t.Fatalf("%d want %d %s", res.StatusCode, want, body)
					}
					if strings.Contains(body, "private-sentinel") || strings.Contains(body, "forged-secret") || strings.Contains(body, "login_required") {
						t.Fatal(body)
					}
					if want == 200 {
						var env struct{ Data map[string]json.RawMessage }
						if json.Unmarshal([]byte(body), &env) != nil {
							t.Fatal(body)
						}
						switch op {
						case "list":
							var items []map[string]json.RawMessage
							json.Unmarshal(env.Data["items"], &items)
							if len(env.Data) != 8 || string(env.Data["total"]) != "2147483648" || string(env.Data["has_prev"]) != "true" || string(env.Data["next_cursor"]) != `"opaque-next"` {
								t.Fatal(body)
							}
							if mode == "empty page" {
								if string(env.Data["items"]) != "[]" {
									t.Fatal(body)
								}
							} else if len(items) != 1 || len(items[0]) != 11 || string(items[0]["user_id"]) != fmt.Sprint(target) || string(items[0]["last_seen_at"]) != "null" || string(items[0]["created_at"]) != `"2020-01-01T00:00:00.123456789Z"` || string(items[0]["is_current"]) != "false" {
								t.Fatal(body)
							}
						case "one", "all":
							changed := "false"
							if mode == "changed" {
								changed = "true"
							}
							message := "User session revoked successfully."
							data := `{"changed":` + changed + `}`
							if op == "all" {
								message = "All user sessions revoked successfully."
								data = `{"changed":` + changed + `,"version":"9007199254740993"}`
							}
							if body != `{"code":"SS","message":"`+message+`","data":`+data+`}` {
								t.Fatal(body)
							}
						}
					} else if !strings.Contains(body, `"data":[]`) {
						t.Fatal(body)
					}
					if want == 409 || want == 429 || want == 503 {
						testutils.AssertServiceErrorResponse(t, res, body, want)
					}
				}
				expected := []string{}
				for i := 0; i < repeats; i++ {
					expected = append(expected, "Me", op)
				}
				if !reflect.DeepEqual(calls, expected) {
					t.Fatal("order", calls)
				}
				md, _ := metadata.FromOutgoingContext(parent)
				if !reflect.DeepEqual(md, parentMD) {
					t.Fatal("parent changed")
				}
			})
		}
	}
}

func adminHTTPGuards(t *testing.T) {
	for _, op := range []string{"list", "one", "all"} {
		for _, mode := range []string{"missing bearer", "me failure", "role only", "wildcard", "false permission", "restricted", "target me", "bad target", "bad UUID", "missing key", "missing version", "listusers", "preflight", "denied origin", "cookie only", "revoked authority", "key conflict"} {
			t.Run(op+"/"+mode, func(t *testing.T) {
				meCalls, rpcCalls := 0, 0
				want := 403
				wantMe := 1
				user := &mocks.Service{MeFunc: func(context.Context, *emptypb.Empty, ...grpc.CallOption) (*userv1.UserData, error) {
					meCalls++
					if mode == "me failure" {
						return nil, common.ErrUnauthorized
					}
					u := shared.MeUserData()
					u.Permissions = map[string]bool{domain.SessionReadPermission: true, domain.SessionRevokePermission: true}
					switch mode {
					case "role only":
						u.RoleCodes = []int32{2}
						u.RoleNames = []string{"admin"}
						u.Permissions = nil
					case "wildcard":
						u.Permissions = map[string]bool{"iam.session.*": true}
					case "false permission":
						u.Permissions = map[string]bool{domain.SessionReadPermission: false, domain.SessionRevokePermission: false}
					case "restricted":
						u.MustChangePassword = true
					}
					return u, nil
				}}
				backendError := func() error {
					rpcCalls++
					if mode == "key conflict" {
						return testutils.ServiceFieldError(t, 409)
					}
					return common.ErrForbidden
				}
				sdk := &clientmocks.AdminSessionService{List: func(context.Context, *sessionv1.ListUserSessionsRequest) (*sessionv1.ListSessionsResponse, error) {
					return nil, backendError()
				}, Revoke: func(context.Context, *sessionv1.RevokeUserSessionRequest) (*sessionv1.RevokeSessionResponse, error) {
					return nil, backendError()
				}, All: func(context.Context, *sessionv1.RevokeAllUserSessionsRequest) (*sessionv1.RevokeAllUserSessionsResponse, error) {
					return nil, backendError()
				}}
				app := adminClientApp(t, user, sdk)
				req := adminRequest(op)
				switch mode {
				case "missing bearer", "cookie only":
					req.Header.Del("Authorization")
					want = 401
					wantMe = 0
				case "me failure":
					want = 401
				case "target me":
					req.URL.Path = strings.Replace(req.URL.Path, "/23/", "/me/", 1)
					req.URL.Path = strings.Replace(req.URL.Path, "/00023/", "/me/", 1)
					want = 400
				case "bad target":
					req.URL.Path = strings.Replace(req.URL.Path, "/23/", "/2147483648/", 1)
					req.URL.Path = strings.Replace(req.URL.Path, "/00023/", "/2147483648/", 1)
					want = 400
				case "bad UUID":
					if op != "one" {
						return
					}
					req.URL.Path = strings.Replace(req.URL.Path, shared.MySessionID, "bad", 1)
					want = 400
				case "missing key":
					if op == "list" {
						return
					}
					req.Header.Del("Idempotency-Key")
					want = 400
				case "missing version":
					if op != "all" {
						return
					}
					req.Header.Del("Version")
					want = 400
				case "listusers":
					req = httptest.NewRequest("GET", "/api/v1/users", nil)
					req.Header.Set("Authorization", authorizationValue)
				case "preflight", "denied origin":
					req.Method = "OPTIONS"
					req.Header.Set("Origin", shared.BrowserOrigin)
					req.Header.Set("Access-Control-Request-Method", "DELETE")
					want = 204
					wantMe = 0
					if mode == "denied origin" {
						req.Header.Set("Origin", "https://evil.example.test")
						want = 403
					}
				case "key conflict":
					want = 409
				}
				res, err := app.Test(req)
				body := testutils.ResponseBody(t, res, err)
				wantRPC := 0
				if mode == "revoked authority" || mode == "key conflict" {
					wantRPC = 1
				}
				if res.StatusCode != want || meCalls != wantMe || rpcCalls != wantRPC || len(res.Cookies()) != 0 {
					t.Fatalf("%d want %d calls %d/%d %s", res.StatusCode, want, meCalls, rpcCalls, body)
				}
				if mode != "listusers" && res.Header.Get("Cache-Control") != "no-store" {
					t.Fatal("cache")
				}
			})
		}
	}
}

// Each caller retry reaches current authority rather than a REST replay cache.
func adminHTTPReplayAuthority(t *testing.T) {
	for _, op := range []string{"one", "all"} {
		for _, outcome := range []string{"committed replay", "authority revoked", "key conflict"} {
			t.Run(op+"/"+outcome, func(t *testing.T) {
				sequence := []string{}
				mutation := 0
				user := &mocks.Service{MeFunc: func(context.Context, *emptypb.Empty, ...grpc.CallOption) (*userv1.UserData, error) {
					sequence = append(sequence, "Me")
					u := shared.MeUserData()
					u.Permissions = map[string]bool{domain.SessionRevokePermission: true}
					return u, nil
				}}
				check := func(c context.Context, id uint64) error {
					t.Helper()
					sequence = append(sequence, op)
					mutation++
					md, _ := metadata.FromOutgoingContext(c)
					if id != 23 || !reflect.DeepEqual(md.Get("idempotency-key"), []string{"opaque, key"}) {
						t.Fatal("retry changed")
					}
					if mutation == 2 {
						switch outcome {
						case "authority revoked":
							return common.ErrForbidden
						case "key conflict":
							return testutils.ServiceFieldError(t, 409)
						}
					}
					return nil
				}
				sdk := &clientmocks.AdminSessionService{
					Revoke: func(c context.Context, r *sessionv1.RevokeUserSessionRequest) (*sessionv1.RevokeSessionResponse, error) {
						if r.SessionId != shared.MySessionID {
							t.Fatal("session changed")
						}
						if err := check(c, r.UserId); err != nil {
							return nil, err
						}
						return &sessionv1.RevokeSessionResponse{Changed: true}, nil
					},
					All: func(c context.Context, r *sessionv1.RevokeAllUserSessionsRequest) (*sessionv1.RevokeAllUserSessionsResponse, error) {
						if r.ExpectedUserVersion == nil || *r.ExpectedUserVersion != 9223372036854775807 {
							t.Fatal("version changed")
						}
						if err := check(c, r.UserId); err != nil {
							return nil, err
						}
						return &sessionv1.RevokeAllUserSessionsResponse{Changed: true, Version: 9007199254740993}, nil
					},
				}
				app := adminClientApp(t, user, sdk)
				firstBody := ""
				for attempt := 0; attempt < 2; attempt++ {
					res, err := app.Test(adminRequest(op))
					body := testutils.ResponseBody(t, res, err)
					want := 200
					if attempt == 1 {
						switch outcome {
						case "authority revoked":
							want = 403
						case "key conflict":
							want = 409
						}
					}
					if res.StatusCode != want || len(res.Cookies()) != 0 || res.Header.Get("Cache-Control") != "no-store" {
						t.Fatal(res.StatusCode, body)
					}
					if attempt == 0 {
						firstBody = body
					} else if outcome == "committed replay" && body != firstBody {
						t.Fatal("committed replay rewritten")
					}
					if want == 409 {
						testutils.AssertServiceErrorResponse(t, res, body, want)
					}
				}
				if !reflect.DeepEqual(sequence, []string{"Me", op, "Me", op}) {
					t.Fatal(sequence)
				}
			})
		}
	}
}
