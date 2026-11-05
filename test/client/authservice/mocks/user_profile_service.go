package mocks

import (
	"context"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"google.golang.org/grpc"
)

type UserProfileService struct {
	Update func(context.Context, *userv1.UpdateUserRequest) (*userv1.UpdateUserResponse, error)
}

func (s *UserProfileService) UpdateUser(c context.Context, r *userv1.UpdateUserRequest, _ ...grpc.CallOption) (*userv1.UpdateUserResponse, error) {
	if s.Update == nil {
		panic("unexpected UpdateUser")
	}
	return s.Update(c, r)
}
