package mocks

import (
	"context"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
)

type Client struct {
	ListUsersFunc func(
		context.Context,
		*userv1.ListUsersRequest,
	) (*userv1.ListUsersResponse, error)
	RegisterFunc func(
		context.Context,
		*userv1.AuthRequest,
	) (*userv1.UserTokenResponse, error)
	LoginFunc func(
		context.Context,
		*userv1.AuthRequest,
	) (*userv1.UserTokenResponse, error)
	Calls int
}

func (client *Client) ListUsers(
	ctx context.Context,
	request *userv1.ListUsersRequest,
) (*userv1.ListUsersResponse, error) {
	client.Calls++
	return client.ListUsersFunc(ctx, request)
}

func (client *Client) Register(
	ctx context.Context,
	request *userv1.AuthRequest,
) (*userv1.UserTokenResponse, error) {
	if client.RegisterFunc == nil {
		panic("unexpected Register call")
	}
	return client.RegisterFunc(ctx, request)
}

func (client *Client) Login(
	ctx context.Context,
	request *userv1.AuthRequest,
) (*userv1.UserTokenResponse, error) {
	if client.LoginFunc == nil {
		panic("unexpected Login call")
	}
	return client.LoginFunc(ctx, request)
}
