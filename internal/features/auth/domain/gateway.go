package domain

import "context"

type RegisterGateway interface {
	Register(ctx context.Context, request AuthRequest) (RegisterResponse, error)
}

type LoginGateway interface {
	Login(ctx context.Context, request AuthRequest) (SessionResult, error)
}
