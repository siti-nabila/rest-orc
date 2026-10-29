package mocks

import (
	"context"
	profilev1 "github.com/siti-nabila/api-contracts/pb/profile/v1"
	"google.golang.org/grpc"
)

type ProfileService struct {
	UpdateFunc func(context.Context, *profilev1.ProfileRequest, ...grpc.CallOption) (*profilev1.ProfileResponse, error)
	Calls      int
}

func (mock *ProfileService) UpdateProfile(ctx context.Context, request *profilev1.ProfileRequest, opts ...grpc.CallOption) (*profilev1.ProfileResponse, error) {
	mock.Calls++
	return mock.UpdateFunc(ctx, request, opts...)
}
