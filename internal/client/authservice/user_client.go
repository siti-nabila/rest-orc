package authservice

import (
	"context"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/rest-orc/internal/client/grpcclient"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"google.golang.org/grpc"
)

// UserDetailService isolates the generated GetUser SDK without widening the
// existing public/auth/list service or session interfaces.
type UserDetailService interface {
	GetUser(context.Context, *userv1.GetUserRequest, ...grpc.CallOption) (*userv1.GetUserResponse, error)
}

func NewWithUserDetailService(transport *grpcclient.Client, service Service, profile ProfileService, password PasswordService, session SessionService, self SelfSessionService, admin AdminSessionService, detail UserDetailService) (*Client, error) {
	client, err := NewWithAdminSessionService(transport, service, profile, password, session, self, admin)
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return nil, dictionary.ErrUserDetailServiceRequired
	}
	client.userDetailService = detail
	return client, nil
}

func (c *Client) GetUser(ctx context.Context, request *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	if ctx == nil {
		return nil, dictionary.ErrGRPCAuthRequestContextRequired
	}
	if request == nil {
		return nil, dictionary.ErrGetUserRequestRequired
	}
	child, cancel := c.transport.RequestContext(ctx)
	defer cancel()
	result, err := c.userDetailService.GetUser(child, request)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, dictionary.ErrGetUserResponseRequired
	}
	return result, nil
}

// UserStatusService isolates the generated command without widening read/session interfaces.
type UserStatusService interface {
	ChangeUserStatus(context.Context, *userv1.ChangeUserStatusRequest, ...grpc.CallOption) (*userv1.ChangeUserStatusResponse, error)
}

// NewWithUserStatusService adds an injectable command SDK after existing dependency guards.
// Older constructors retain defaults on the same transport connection.
func NewWithUserStatusService(transport *grpcclient.Client, service Service, profile ProfileService, password PasswordService, session SessionService, self SelfSessionService, admin AdminSessionService, detail UserDetailService, status UserStatusService) (*Client, error) {
	client, err := NewWithUserDetailService(transport, service, profile, password, session, self, admin, detail)
	if err != nil {
		return nil, err
	}
	if status == nil {
		return nil, dictionary.ErrUserStatusServiceRequired
	}
	client.userStatusService = status
	return client, nil
}

func (c *Client) ChangeUserStatus(ctx context.Context, request *userv1.ChangeUserStatusRequest) (*userv1.ChangeUserStatusResponse, error) {
	if ctx == nil {
		return nil, dictionary.ErrGRPCAuthRequestContextRequired
	}
	if request == nil {
		return nil, dictionary.ErrChangeUserStatusRequestRequired
	}
	child, cancel := c.transport.RequestContext(ctx)
	defer cancel()
	result, err := c.userStatusService.ChangeUserStatus(child, request)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, dictionary.ErrChangeUserStatusResponseRequired
	}
	return result, nil
}

// UserProfileService is the target-user UpdateUser command, distinct from the
// self-profile ProfileService and without widening any existing interface.
type UserProfileService interface {
	UpdateUser(context.Context, *userv1.UpdateUserRequest, ...grpc.CallOption) (*userv1.UpdateUserResponse, error)
}

// NewWithUserProfileService adds the command SDK after all existing guards.
// Older constructors keep their defaults on the same reusable connection.
func NewWithUserProfileService(transport *grpcclient.Client, service Service, profile ProfileService, password PasswordService, session SessionService, self SelfSessionService, admin AdminSessionService, detail UserDetailService, status UserStatusService, userProfile UserProfileService) (*Client, error) {
	client, err := NewWithUserStatusService(transport, service, profile, password, session, self, admin, detail, status)
	if err != nil {
		return nil, err
	}
	if userProfile == nil {
		return nil, dictionary.ErrUserProfileServiceRequired
	}
	client.userProfileService = userProfile
	return client, nil
}

func (c *Client) UpdateUser(ctx context.Context, request *userv1.UpdateUserRequest) (*userv1.UpdateUserResponse, error) {
	if ctx == nil {
		return nil, dictionary.ErrGRPCAuthRequestContextRequired
	}
	if request == nil {
		return nil, dictionary.ErrUpdateUserRequestRequired
	}
	child, cancel := c.transport.RequestContext(ctx)
	defer cancel()
	result, err := c.userProfileService.UpdateUser(child, request)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, dictionary.ErrUpdateUserResponseRequired
	}
	return result, nil
}

// UserCreationService is the CreateUser command, distinct from public Register and the
// self-profile ProfileService and without widening any existing interface.
type UserCreationService interface {
	CreateUser(context.Context, *userv1.CreateUserRequest, ...grpc.CallOption) (*userv1.CreateUserResponse, error)
}

// NewWithUserCreationService adds the command SDK after all existing guards.
// Older constructors keep their defaults on the same reusable connection.
func NewWithUserCreationService(transport *grpcclient.Client, service Service, profile ProfileService, password PasswordService, session SessionService, self SelfSessionService, admin AdminSessionService, detail UserDetailService, status UserStatusService, userProfile UserProfileService, userCreation UserCreationService) (*Client, error) {
	client, err := NewWithUserProfileService(transport, service, profile, password, session, self, admin, detail, status, userProfile)
	if err != nil {
		return nil, err
	}
	if userCreation == nil {
		return nil, dictionary.ErrUserCreationServiceRequired
	}
	client.userCreationService = userCreation
	return client, nil
}

func (c *Client) CreateUser(ctx context.Context, request *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
	if ctx == nil {
		return nil, dictionary.ErrGRPCAuthRequestContextRequired
	}
	if request == nil {
		return nil, dictionary.ErrCreateUserRequestRequired
	}
	child, cancel := c.transport.RequestContext(ctx)
	defer cancel()
	result, err := c.userCreationService.CreateUser(child, request)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, dictionary.ErrCreateUserResponseRequired
	}
	return result, nil
}
