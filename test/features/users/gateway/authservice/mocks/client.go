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
	Calls int
}

func (client *Client) ListUsers(
	ctx context.Context,
	request *userv1.ListUsersRequest,
) (*userv1.ListUsersResponse, error) {
	client.Calls++
	return client.ListUsersFunc(ctx, request)
}
