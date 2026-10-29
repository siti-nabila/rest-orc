package test_scenarios

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	requestauth "github.com/siti-nabila/rest-orc/internal/auth"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/test/auth/mocks"
	"github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func MeSnapshots() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "Me maps the complete snapshot and skips nil assignments", Run: fullMeSnapshot},
		{Name: "Me restricted snapshot has no privileges", Run: restrictedMeSnapshot},
		{Name: "Me rejects nil and invalid timestamp responses", Run: invalidMeSnapshots},
		{Name: "Me absent optional values stay absent", Run: absentMeValues},
		{Name: "Principal context isolates all mutable fields", Run: isolatedPrincipal},
		{Name: "Principal concurrent readers cannot mutate shared snapshot", Run: concurrentPrincipalReaders},
	}
}

func fullMeSnapshot(t *testing.T) {
	user := fixtures.MeUserData()
	user.Roles = append(user.Roles, nil)
	user.ApplicationAccess = append(user.ApplicationAccess, nil)
	ctx := authenticateMe(t, user)
	want := fixtures.MePrincipal()
	assertPrincipal(t, ctx, want)
	user.RoleCodes[0] = 2
	user.RoleNames[0] = "changed"
	user.Roles[0].Code = "changed"
	user.ApplicationAccess[0].Code = "changed"
	user.Permissions["clm.document.read"] = false
	user.CreatedAt.Seconds++
	assertPrincipal(t, ctx, want)
	if want.HasRole(requestauth.RoleAdmin.Int32()) || !want.HasRole(3) {
		t.Fatal("legacy role membership changed")
	}
}

func restrictedMeSnapshot(t *testing.T) {
	user := fixtures.MeUserData()
	user.MustChangePassword = true
	user.RoleCodes = []int32{requestauth.RoleAdmin.Int32()}
	principal, ok := requestauth.PrincipalFromContext(authenticateMe(t, user))
	if !ok || !principal.MustChangePassword || principal.ID != 17 || principal.Version != user.Version || principal.CreatedAt == nil {
		t.Fatalf("restricted profile lost: %#v", principal)
	}
	if len(principal.RoleCodes)+len(principal.RoleNames)+len(principal.Roles)+len(principal.ApplicationAccess)+len(principal.Permissions) != 0 || principal.HasRole(2) {
		t.Fatalf("restricted privileges survived: %#v", principal)
	}
}

func invalidMeSnapshots(t *testing.T) {
	for _, tc := range []struct {
		name string
		user *userv1.UserData
		want error
	}{
		{name: "nil response", want: dictionary.ErrGRPCAuthMeResponseRequired},
		{name: "invalid seconds", user: &userv1.UserData{CreatedAt: &timestamppb.Timestamp{Seconds: 253402300800}}, want: common.ErrInternalServerError},
		{name: "invalid nanos", user: &userv1.UserData{CreatedAt: &timestamppb.Timestamp{Nanos: -1}}, want: common.ErrInternalServerError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &mocks.MeClient{MeFunc: func(context.Context) (*userv1.UserData, error) { return tc.user, nil }}
			ctx, err := newAuthenticator(t, client).Authenticate(context.Background(), "Bearer member")
			if !errors.Is(err, tc.want) || ctx != nil || client.Calls != 1 {
				t.Fatalf("Authenticate = (%v, %v), calls=%d", ctx, err, client.Calls)
			}
		})
	}
}

func absentMeValues(t *testing.T) {
	assertPrincipal(t, authenticateMe(t, &userv1.UserData{Id: 17}), requestauth.Principal{ID: 17})
	if _, ok := requestauth.PrincipalFromContext(nil); ok {
		t.Fatal("nil context has a principal")
	}
	if _, ok := requestauth.PrincipalFromContext(context.Background()); ok {
		t.Fatal("empty context has a principal")
	}
}

func isolatedPrincipal(t *testing.T) {
	input := fixtures.MePrincipal()
	want := fixtures.MePrincipal()
	ctx := requestauth.WithPrincipal(context.Background(), input)
	mutatePrincipal(input)
	assertPrincipal(t, ctx, want)
	output, _ := requestauth.PrincipalFromContext(ctx)
	mutatePrincipal(output)
	assertPrincipal(t, ctx, want)
}

func concurrentPrincipalReaders(t *testing.T) {
	want := fixtures.MePrincipal()
	ctx := requestauth.WithPrincipal(context.Background(), want)
	var wg sync.WaitGroup
	for range 12 {
		wg.Go(func() {
			p, _ := requestauth.PrincipalFromContext(ctx)
			mutatePrincipal(p)
			assertPrincipal(t, ctx, want)
		})
	}
	wg.Wait()
}

func mutatePrincipal(p requestauth.Principal) {
	p.RoleCodes[0] = 2
	p.RoleNames[0] = "changed"
	p.Roles[0].Name = "changed"
	p.ApplicationAccess[0].Name = "changed"
	p.Permissions["clm.document.read"] = false
	*p.CreatedAt = p.CreatedAt.AddDate(1, 0, 0)
}

func assertPrincipal(t *testing.T, ctx context.Context, want requestauth.Principal) {
	t.Helper()
	got, ok := requestauth.PrincipalFromContext(ctx)
	if !ok || !reflect.DeepEqual(got, want) {
		t.Errorf("principal = %#v, want %#v", got, want)
	}
}

func authenticateMe(t *testing.T, user *userv1.UserData) context.Context {
	t.Helper()
	client := &mocks.MeClient{MeFunc: func(context.Context) (*userv1.UserData, error) { return user, nil }}
	ctx, err := newAuthenticator(t, client).Authenticate(context.Background(), "Bearer member")
	if err != nil || client.Calls != 1 {
		t.Fatalf("Authenticate error=%v, Me calls=%d", err, client.Calls)
	}
	return ctx
}
