package mocks

import (
	"context"
	profilev1 "github.com/siti-nabila/api-contracts/pb/profile/v1"
)

type ProfileClient struct {
	UpdateFunc func(context.Context, *profilev1.ProfileRequest) (*profilev1.ProfileResponse, error)
	Calls      int
}

func (mock *ProfileClient) UpdateProfile(ctx context.Context, request *profilev1.ProfileRequest) (*profilev1.ProfileResponse, error) {
	mock.Calls++
	return mock.UpdateFunc(ctx, request)
}
