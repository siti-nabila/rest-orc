package mocks

import (
	"context"

	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
)

type Gateway struct {
	RegisterFunc func(
		context.Context,
		domain.AuthRequest,
	) (domain.RegisterResponse, error)
	LoginFunc func(
		context.Context,
		domain.AuthRequest,
	) (domain.SessionResult, error)
	RegisterCalls int
	LoginCalls    int
}

func (gateway *Gateway) Register(
	ctx context.Context,
	request domain.AuthRequest,
) (domain.RegisterResponse, error) {
	gateway.RegisterCalls++
	if gateway.RegisterFunc == nil {
		panic("unexpected Register call")
	}
	return gateway.RegisterFunc(ctx, request)
}

func (gateway *Gateway) Login(
	ctx context.Context,
	request domain.AuthRequest,
) (domain.SessionResult, error) {
	gateway.LoginCalls++
	if gateway.LoginFunc == nil {
		panic("unexpected Login call")
	}
	return gateway.LoginFunc(ctx, request)
}
