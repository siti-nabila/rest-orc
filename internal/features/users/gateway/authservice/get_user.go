package authservice

import (
	"context"
	"strconv"
	"time"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"google.golang.org/grpc/metadata"
)

type UserDetailClient interface {
	GetUser(context.Context, *userv1.GetUserRequest) (*userv1.GetUserResponse, error)
}

type UserDetailGateway struct{ client UserDetailClient }

func NewUserDetail(client UserDetailClient) (*UserDetailGateway, error) {
	if client == nil {
		return nil, dictionary.ErrUsersGatewayClientRequired
	}
	return &UserDetailGateway{client: client}, nil
}

func (g *UserDetailGateway) GetUser(ctx context.Context, target uint64) (domain.UserDetail, error) {
	if ctx == nil {
		return domain.UserDetail{}, dictionary.ErrUsersGatewayContextRequired
	}
	if err := domain.ValidateUserDetailID(target); err != nil {
		return domain.UserDetail{}, err
	}
	md, _ := metadata.FromOutgoingContext(ctx)
	md = md.Copy()
	md.Delete("version")
	md.Delete("idempotency-key")
	result, err := g.client.GetUser(metadata.NewOutgoingContext(ctx, md), &userv1.GetUserRequest{Id: target})
	if err != nil {
		return domain.UserDetail{}, err
	}
	return userDetail(result)
}

func userDetail(result *userv1.GetUserResponse) (domain.UserDetail, error) {
	if result == nil || result.User == nil {
		return domain.UserDetail{}, common.ErrInternalServerError
	}
	return safeUserDetail(result.User)
}

func safeUserDetail(user *userv1.User) (domain.UserDetail, error) {
	if user == nil {
		return domain.UserDetail{}, common.ErrInternalServerError
	}
	if domain.ValidateUserDetailID(user.Id) != nil || user.Version == 0 || user.Version > 9223372036854775807 || user.CreatedAt == nil || user.CreatedAt.CheckValid() != nil {
		return domain.UserDetail{}, common.ErrInternalServerError
	}
	switch user.Status {
	case "PENDING", "ACTIVE", "SUSPENDED", "DEACTIVATED":
	default:
		return domain.UserDetail{}, common.ErrInternalServerError
	}
	detail := domain.UserDetail{
		ID: user.Id, Email: user.Email, Name: user.Name, Address: user.Address,
		Phone: user.PhoneNo, CountryCode: user.CountryCode, Status: user.Status,
		Version: strconv.FormatUint(user.Version, 10), CreatedAt: user.CreatedAt.AsTime().UTC().Format(time.RFC3339Nano),
		MustChangePassword: user.MustChangePassword,
		Roles:              make([]domain.UserDetailRole, len(user.Roles)),
		ApplicationAccess:  make([]domain.UserDetailApplicationAccess, len(user.ApplicationAccess)),
	}
	for i, role := range user.Roles {
		if role == nil || domain.ValidateUserDetailID(role.RoleId) != nil {
			return domain.UserDetail{}, common.ErrInternalServerError
		}
		detail.Roles[i] = domain.UserDetailRole{RoleID: role.RoleId, Code: role.Code, Name: role.Name, Status: role.Status, RoleStatus: role.RoleStatus}
	}
	for i, app := range user.ApplicationAccess {
		if app == nil || domain.ValidateUserDetailID(app.ApplicationId) != nil {
			return domain.UserDetail{}, common.ErrInternalServerError
		}
		detail.ApplicationAccess[i] = domain.UserDetailApplicationAccess{ApplicationID: app.ApplicationId, Code: app.Code, Name: app.Name, Status: app.Status, ApplicationStatus: app.ApplicationStatus}
	}
	return detail, nil
}
