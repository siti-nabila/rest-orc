package mocks

import (
	"context"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"google.golang.org/grpc"
)

type UserStatusService struct {
	Change func(context.Context, *userv1.ChangeUserStatusRequest) (*userv1.ChangeUserStatusResponse, error)
}

func (s *UserStatusService) ChangeUserStatus(c context.Context, r *userv1.ChangeUserStatusRequest, _ ...grpc.CallOption) (*userv1.ChangeUserStatusResponse, error) {
	if s.Change == nil {
		panic("unexpected ChangeUserStatus")
	}
	return s.Change(c, r)
}
