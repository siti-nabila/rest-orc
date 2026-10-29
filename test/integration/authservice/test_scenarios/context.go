package test_scenarios

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/siti-nabila/api-contracts/pkg/locale"
	"github.com/siti-nabila/rest-orc/test/integration/authservice/fixtures"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func contextBoundaries(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newTransport(t, secureConfig(t), defaultServerTLS(t), nil)
		defer h.close(t)
		parent, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if _, err := h.client.Me(parent); err != nil {
			t.Fatal("RPC with earlier parent deadline failed")
		}
		calls, _ := h.snapshot()
		deadline, _ := parent.Deadline()
		if len(calls) != 1 || calls[0].deadline.After(deadline) {
			t.Fatal("client extended the parent deadline")
		}
		canceled, stop := context.WithCancel(context.Background())
		stop()
		expired, expire := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		defer expire()
		for _, tc := range []struct {
			ctx  context.Context
			code codes.Code
		}{{canceled, codes.Canceled}, {expired, codes.DeadlineExceeded}} {
			_, err := h.client.Me(tc.ctx)
			if status.Code(err) != tc.code {
				t.Error("canceled/expired context did not retain its RPC status")
			}
		}
		calls, _ = h.snapshot()
		if len(calls) != 1 {
			t.Error("already canceled/expired context reached the handler")
		}
	})
	for _, mode := range []string{"client timeout", "parent cancellation"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				entered, finished := make(chan struct{}), make(chan struct{})
				h := newTransport(t, secureConfig(t), defaultServerTLS(t), func(ctx context.Context, _ any, _ *grpc.UnaryServerInfo, _ grpc.UnaryHandler) (any, error) {
					close(entered)
					<-ctx.Done()
					close(finished)
					return nil, ctx.Err()
				})
				defer h.close(t)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				result := make(chan error, 1)
				start := time.Now()
				go func() { _, err := h.client.Me(ctx); result <- err }()
				<-entered
				code := codes.DeadlineExceeded
				if mode == "parent cancellation" {
					cancel()
					code = codes.Canceled
				}
				if status.Code(<-result) != code {
					t.Error("in-flight call lost cancellation/timeout status")
				}
				<-finished
				if mode == "client timeout" && time.Since(start) != 5*time.Second {
					t.Error("configured request timeout changed")
				}
				calls, _ := h.snapshot()
				if len(calls) != 1 {
					t.Error("cancellation or timeout retried the business RPC")
				}
			})
		})
	}
}

func concurrentCredentials(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const count = 8
		entered, release := make(chan struct{}, count), make(chan struct{})
		h := newTransport(t, secureConfig(t), defaultServerTLS(t), func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			entered <- struct{}{}
			<-release
			return handler(ctx, req)
		})
		defer h.close(t)
		wantLocales := map[string]string{}
		var workers sync.WaitGroup
		for i := range count {
			wantLocales[fmt.Sprint(i)] = []string{"en", "id"}[i%2]
			workers.Go(func() {
				id := fmt.Sprint(i)
				md := metadata.Pairs("authorization", "Bearer user-"+id, locale.MetadataKey, []string{"en", "id"}[i%2], "x-request-id", id)
				before := md.Copy()
				_, err := h.client.Me(metadata.NewOutgoingContext(context.Background(), md))
				if err != nil || !reflect.DeepEqual(md, before) {
					t.Error("concurrent RPC failed or mutated caller metadata")
				}
			})
		}
		for range count {
			<-entered
		}
		close(release)
		workers.Wait()
		calls, dials := h.snapshot()
		if len(calls) != count || dials != 1 {
			t.Fatalf("concurrent RPCs/dials = %d/%d, want %d/1", len(calls), dials, count)
		}
		seen := map[string]bool{}
		for _, got := range calls {
			ids := got.metadata.Get("x-request-id")
			if len(ids) != 1 || seen[ids[0]] {
				t.Fatal("concurrent request identity duplicated or missing")
			}
			id := ids[0]
			seen[id] = true
			assertMetadata(t, got.metadata, "authorization", "Bearer user-"+id)
			assertMetadata(t, got.metadata, "x-service-token", fixtures.Token(17))
			assertMetadata(t, got.metadata, locale.MetadataKey, wantLocales[id])
		}
	})
}

func credentialRotation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cfg := secureConfig(t)
		t.Setenv("AUTH_GRPC_SERVICE_TOKEN", cfg.ServiceToken)
		old := newTransport(t, cfg, defaultServerTLS(t), nil)
		defer old.close(t)
		t.Setenv("AUTH_GRPC_SERVICE_TOKEN", fixtures.Token(19))
		cfg.ServiceToken = fixtures.Token(19)
		newer := newTransport(t, cfg, defaultServerTLS(t), nil)
		defer newer.close(t)
		for _, h := range []*transport{old, newer, old} {
			if _, err := h.client.Me(context.Background()); err != nil {
				t.Fatal("explicitly configured client failed")
			}
		}
		oldCalls, _ := old.snapshot()
		for _, got := range oldCalls {
			assertMetadata(t, got.metadata, "x-service-token", fixtures.Token(17))
		}
		newCalls, _ := newer.snapshot()
		if len(oldCalls) != 2 || len(newCalls) != 1 {
			t.Fatal("rotation added a fallback or extra call")
		}
		assertMetadata(t, newCalls[0].metadata, "x-service-token", fixtures.Token(19))
	})
}
