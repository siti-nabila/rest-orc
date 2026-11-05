package mocks

import (
	"context"
	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
)

type MySessionsGateway struct {
	List   func(context.Context, domain.MySessionsQuery) (domain.MySessionsPage, error)
	Revoke func(context.Context, string) (domain.RevokeMySessionResponse, error)
	All    func(context.Context) (domain.RevokeAllMySessionsResponse, error)
}

func (g *MySessionsGateway) ListMySessions(c context.Context, q domain.MySessionsQuery) (domain.MySessionsPage, error) {
	return g.List(c, q)
}
func (g *MySessionsGateway) RevokeMySession(c context.Context, id string) (domain.RevokeMySessionResponse, error) {
	return g.Revoke(c, id)
}
func (g *MySessionsGateway) RevokeAllMySessions(c context.Context) (domain.RevokeAllMySessionsResponse, error) {
	return g.All(c)
}
