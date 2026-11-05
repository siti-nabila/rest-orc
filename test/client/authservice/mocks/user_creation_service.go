package mocks

import (
	"context"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"google.golang.org/grpc"
)

type UserCreationService struct {
	Create func(context.Context, *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error)
}

func (s *UserCreationService) CreateUser(c context.Context, r *userv1.CreateUserRequest, _ ...grpc.CallOption) (*userv1.CreateUserResponse, error) {
	if s.Create == nil {
		panic("unexpected CreateUser")
	}
	return s.Create(c, r)
}
