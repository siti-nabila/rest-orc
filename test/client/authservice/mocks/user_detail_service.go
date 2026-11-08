package mocks

import (
	"context"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"google.golang.org/grpc"
)

type UserDetailService struct {
	Get func(context.Context, *userv1.GetUserRequest) (*userv1.GetUserResponse, error)
}

func (s *UserDetailService) GetUser(c context.Context, r *userv1.GetUserRequest, _ ...grpc.CallOption) (*userv1.GetUserResponse, error) {
	return s.Get(c, r)
}
