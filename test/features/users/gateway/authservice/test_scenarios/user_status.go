package test_scenarios

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"testing"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	gateway "github.com/siti-nabila/rest-orc/internal/features/users/gateway/authservice"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/grpc/metadata"
)

type statusClient func(context.Context, *userv1.ChangeUserStatusRequest) (*userv1.ChangeUserStatusResponse, error)

func (f statusClient) ChangeUserStatus(c context.Context, r *userv1.ChangeUserStatusRequest) (*userv1.ChangeUserStatusResponse, error) {
	return f(c, r)
}
func UserStatusScenarios() []testutils.Scenario {
	return []testutils.Scenario{{Name: "status gateway optional payload exact metadata isolation and safe result", Run: statusGatewayMapping}}
}
func statusGatewayMapping(t *testing.T) {
	if _, err := gateway.NewUserStatus(nil); err == nil {
		t.Fatal("nil client")
	}
	md := metadata.Pairs("authorization", "Bearer actor", "version", "88", "idempotency-key", "old", "idempotency-key", "old2", "x-request-id", "status")
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	calls := 0
	state, reason := "ACTIVE", "  approved\r\nreason  "
	request := domain.ChangeUserStatusRequest{UserID: 23, Status: &state, Reason: &reason, Version: 9007199254740993, IdempotencyKey: "opaque, key"}
	result := &userv1.ChangeUserStatusResponse{UserId: 23, Status: "ACTIVE", Version: 9007199254740993}
	var failure error
	sdk := statusClient(func(c context.Context, r *userv1.ChangeUserStatusRequest) (*userv1.ChangeUserStatusResponse, error) {
		calls++
		actual, _ := metadata.FromOutgoingContext(c)
		want := md.Copy()
		want.Delete("version")
		want.Set("idempotency-key", request.IdempotencyKey)
		if !reflect.DeepEqual(actual, want) || r.UserId != 23 || r.Status == nil || *r.Status != "ACTIVE" || r.ExpectedUserVersion == nil || *r.ExpectedUserVersion != request.Version || r.Reason != "approved\r\nreason" {
			t.Fatal(r, actual, want)
		}
		*r.Status = "SDK mutation"
		*r.ExpectedUserVersion = 1
		actual.Set("x-request-id", "SDK mutation")
		return result, failure
	})
	g, _ := gateway.NewUserStatus(sdk)
	if _, err := g.ChangeUserStatus(nil, request); err == nil {
		t.Fatal("nil context")
	}
	invalid := request
	invalid.UserID = 0
	if _, err := g.ChangeUserStatus(ctx, invalid); err == nil || calls != 0 {
		t.Fatal("guard dispatch")
	}
	for _, s := range []string{"PENDING", "ACTIVE", "SUSPENDED", "DEACTIVATED"} {
		for _, v := range []uint64{1, 9007199254740993, 9223372036854775807} {
			for _, changed := range []bool{true, false} {
				result = &userv1.ChangeUserStatusResponse{UserId: 23, Status: s, Version: v, Changed: changed}
				before := calls
				got, err := g.ChangeUserStatus(ctx, request)
				if err != nil || calls != before+1 || got.UserID != 23 || got.Status != s || got.Version != strconv.FormatUint(v, 10) || got.Changed != changed {
					t.Fatal(got, err)
				}
				raw, _ := json.Marshal(got)
				var data map[string]json.RawMessage
				_ = json.Unmarshal(raw, &data)
				if len(data) != 4 || string(data["changed"]) != strconv.FormatBool(changed) {
					t.Fatal(string(raw))
				}
			}
		}
	}
	if state != "ACTIVE" || reason != "  approved\r\nreason  " || request.Version != 9007199254740993 {
		t.Fatal("caller mutated")
	}
	after, _ := metadata.FromOutgoingContext(ctx)
	if !reflect.DeepEqual(md, after) || md.Get("x-request-id")[0] != "status" {
		t.Fatal("metadata mutated")
	}
	for _, mode := range []string{"nil", "zero ID", "large ID", "wrong ID", "empty status", "bad status", "zero version", "overflow"} {
		result = &userv1.ChangeUserStatusResponse{UserId: 23, Status: "ACTIVE", Version: 1}
		switch mode {
		case "nil":
			result = nil
		case "zero ID":
			result.UserId = 0
		case "large ID":
			result.UserId = 2147483648
		case "wrong ID":
			result.UserId = 99
		case "empty status":
			result.Status = ""
		case "bad status":
			result.Status = "active"
		case "zero version":
			result.Version = 0
		case "overflow":
			result.Version = 1 << 63
		}
		got, err := g.ChangeUserStatus(ctx, request)
		if !errors.Is(err, common.ErrInternalServerError) || got.UserID != 0 {
			t.Fatal(mode, got, err)
		}
	}
	failure = errors.New("dependency")
	if got, err := g.ChangeUserStatus(ctx, request); err != failure || got.UserID != 0 {
		t.Fatal(got, err)
	}
}
