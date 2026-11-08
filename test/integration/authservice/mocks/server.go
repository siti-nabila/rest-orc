package mocks

import (
	"context"
	"strings"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	usersfixtures "github.com/siti-nabila/rest-orc/test/integration/users/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Server struct {
	userv1.UnimplementedUserServiceServer
}

func (*Server) Register(context.Context, *userv1.AuthRequest) (*userv1.UserTokenResponse, error) {
	return fixtures.RegistrationResponse(), nil
}

func (*Server) Login(context.Context, *userv1.AuthRequest) (*userv1.UserTokenResponse, error) {
	return fixtures.LoginTokenResponse(), nil
}

func (*Server) Me(context.Context, *emptypb.Empty) (*userv1.UserData, error) {
	return usersfixtures.MeResponse(), nil
}

func (*Server) ListUsers(context.Context, *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error) {
	return usersfixtures.ListUsersResponse(), nil
}

func (*Server) GetUser(_ context.Context, request *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	return fixtures.UserDetailResponse(request.Id), nil
}

func (*Server) ChangeUserStatus(_ context.Context, r *userv1.ChangeUserStatusRequest) (*userv1.ChangeUserStatusResponse, error) {
	return &userv1.ChangeUserStatusResponse{UserId: r.UserId, Status: r.GetStatus(), Version: r.GetExpectedUserVersion(), Changed: false}, nil
}

func (*Server) UpdateUser(_ context.Context, r *userv1.UpdateUserRequest) (*userv1.UpdateUserResponse, error) {
	user := fixtures.UserDetailResponse(r.UserId).User
	user.Version = r.GetExpectedUserVersion()
	return &userv1.UpdateUserResponse{User: user, Changed: false}, nil
}

func (*Server) CreateUser(_ context.Context, r *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
	user := fixtures.UserDetailResponse(23).User
	user.Email = strings.ToLower(strings.TrimSpace(r.Email))
	user.Status = r.GetStatus()
	return &userv1.CreateUserResponse{User: user}, nil
}

func (*Server) ListUserRoles(_ context.Context, r *userv1.ListUserRolesRequest) (*userv1.ListUserRolesResponse, error) {
	return fixtures.UserRoleList(r.UserId), nil
}
func (*Server) GrantUserRoles(_ context.Context, r *userv1.GrantUserRolesRequest) (*userv1.GrantUserRolesResponse, error) {
	return &userv1.GrantUserRolesResponse{UserId: r.UserId, Version: r.GetExpectedUserVersion()}, nil
}
func (*Server) RevokeUserRoles(_ context.Context, r *userv1.RevokeUserRolesRequest) (*userv1.RevokeUserRolesResponse, error) {
	return &userv1.RevokeUserRolesResponse{UserId: r.UserId, Version: r.GetExpectedUserVersion()}, nil
}

func (*Server) ListUserApplicationAccess(_ context.Context, r *userv1.ListUserApplicationAccessRequest) (*userv1.ListUserApplicationAccessResponse, error) {
	return fixtures.UserAccessHistory(r.UserId), nil
}
