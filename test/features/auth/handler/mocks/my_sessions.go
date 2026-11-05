package mocks

import (
	"context"
	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
)

type MySessionsList struct {
	Run func(context.Context, domain.SessionActor, domain.MySessionsQuery) (domain.MySessionsPage, error)
}

func (s *MySessionsList) Execute(c context.Context, a domain.SessionActor, q domain.MySessionsQuery) (domain.MySessionsPage, error) {
	return s.Run(c, a, q)
}

type MySessionsRevoke struct {
	Run func(context.Context, domain.SessionActor, string) (domain.RevokeMySessionResponse, error)
}

func (s *MySessionsRevoke) Execute(c context.Context, a domain.SessionActor, id string) (domain.RevokeMySessionResponse, error) {
	return s.Run(c, a, id)
}

type MySessionsAll struct {
	Run func(context.Context, domain.SessionActor) (domain.RevokeAllMySessionsResponse, error)
}

func (s *MySessionsAll) Execute(c context.Context, a domain.SessionActor) (domain.RevokeAllMySessionsResponse, error) {
	return s.Run(c, a)
}
