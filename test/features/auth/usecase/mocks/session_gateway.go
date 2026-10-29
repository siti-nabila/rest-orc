package mocks

import (
	"context"

	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
)

type SessionGateway struct {
	RefreshFunc func(context.Context, string) (domain.SessionResult, error)
	LogoutFunc  func(context.Context, domain.LogoutCredential) (domain.LogoutResponse, error)
}

func (g *SessionGateway) Refresh(ctx context.Context, token string) (domain.SessionResult, error) {
	return g.RefreshFunc(ctx, token)
}
func (g *SessionGateway) Logout(ctx context.Context, proof domain.LogoutCredential) (domain.LogoutResponse, error) {
	return g.LogoutFunc(ctx, proof)
}
