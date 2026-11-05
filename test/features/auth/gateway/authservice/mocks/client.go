package mocks

import (
	"context"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
)

type Client struct {
	RegisterFunc  func(context.Context, *userv1.AuthRequest) (*userv1.UserTokenResponse, error)
	LoginFunc     func(context.Context, *userv1.AuthRequest) (*userv1.UserTokenResponse, error)
	RegisterCalls int
	LoginCalls    int
}

func (client *Client) Register(ctx context.Context, request *userv1.AuthRequest) (*userv1.UserTokenResponse, error) {
	client.RegisterCalls++
	if client.RegisterFunc == nil {
		panic("unexpected Register call")
	}
	return client.RegisterFunc(ctx, request)
}

func (client *Client) Login(ctx context.Context, request *userv1.AuthRequest) (*userv1.UserTokenResponse, error) {
	client.LoginCalls++
	if client.LoginFunc == nil {
		panic("unexpected Login call")
	}
	return client.LoginFunc(ctx, request)
}
