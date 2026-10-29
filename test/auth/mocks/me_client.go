package mocks

import (
	"context"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
)

type MeClient struct {
	MeFunc func(context.Context) (*userv1.UserData, error)
	Calls  int
}

func (client *MeClient) Me(ctx context.Context) (*userv1.UserData, error) {
	client.Calls++
	return client.MeFunc(ctx)
}
