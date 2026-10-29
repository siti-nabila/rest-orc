package mocks

import (
	"context"
	profilev1 "github.com/siti-nabila/api-contracts/pb/profile/v1"
	"google.golang.org/grpc"
)

type ProfileService struct {
	UpdateFunc func(context.Context, *profilev1.ProfileRequest) (*profilev1.ProfileResponse, error)
}

func (mock *ProfileService) UpdateProfile(ctx context.Context, request *profilev1.ProfileRequest, _ ...grpc.CallOption) (*profilev1.ProfileResponse, error) {
	return mock.UpdateFunc(ctx, request)
}
