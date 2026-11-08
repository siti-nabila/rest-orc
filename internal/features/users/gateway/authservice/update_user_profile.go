package authservice

import (
	"context"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type UserProfileClient interface {
	UpdateUser(context.Context, *userv1.UpdateUserRequest) (*userv1.UpdateUserResponse, error)
}

type UserProfileGateway struct{ client UserProfileClient }

func NewUserProfile(client UserProfileClient) (*UserProfileGateway, error) {
	if client == nil {
		return nil, dictionary.ErrUsersGatewayClientRequired
	}
	return &UserProfileGateway{client: client}, nil
}

// UpdateUserProfile consumes values normalized once by the usecase. Repeating
// phone normalization here could strip a second legitimate prefix or zero.
func (g *UserProfileGateway) UpdateUserProfile(ctx context.Context, request domain.UpdateUserProfileRequest) (domain.UpdateUserProfileResponse, error) {
	if ctx == nil {
		return domain.UpdateUserProfileResponse{}, dictionary.ErrUsersGatewayContextRequired
	}
	if err := request.ValidatePreconditions(); err != nil {
		return domain.UpdateUserProfileResponse{}, err
	}
	if !request.Profile.HasFields() {
		return domain.UpdateUserProfileResponse{}, common.ErrBadRequest
	}
	payload := userProfilePayload(request)
	md, _ := metadata.FromOutgoingContext(ctx)
	md = md.Copy()
	md.Delete("version")
	md.Delete("idempotency-key")
	if request.IdempotencyKey != nil {
		md.Set("idempotency-key", *request.IdempotencyKey)
	}
	result, err := g.client.UpdateUser(metadata.NewOutgoingContext(ctx, md), payload)
	if err != nil {
		return domain.UpdateUserProfileResponse{}, err
	}
	if result == nil {
		return domain.UpdateUserProfileResponse{}, common.ErrInternalServerError
	}
	user, err := safeUserDetail(result.User)
	if err != nil {
		return domain.UpdateUserProfileResponse{}, err
	}
	if user.ID != request.UserID {
		return domain.UpdateUserProfileResponse{}, common.ErrInternalServerError
	}
	return domain.UpdateUserProfileResponse{Changed: result.Changed, User: &user}, nil
}

func userProfilePayload(request domain.UpdateUserProfileRequest) *userv1.UpdateUserRequest {
	version := request.Version
	payload := &userv1.UpdateUserRequest{UserId: request.UserID, ExpectedUserVersion: &version, FieldMask: &fieldmaskpb.FieldMask{}}
	if request.Profile.Name != nil {
		value := *request.Profile.Name
		payload.Name = &value
		payload.FieldMask.Paths = append(payload.FieldMask.Paths, "name")
	}
	if request.Profile.Address != nil {
		value := *request.Profile.Address
		payload.Address = &value
		payload.FieldMask.Paths = append(payload.FieldMask.Paths, "address")
	}
	if request.Profile.Phone != nil {
		value := *request.Profile.Phone
		payload.Phone = &value
		payload.FieldMask.Paths = append(payload.FieldMask.Paths, "phone")
	}
	if request.Profile.CountryCode != nil {
		value := *request.Profile.CountryCode
		payload.CountryCode = &value
		payload.FieldMask.Paths = append(payload.FieldMask.Paths, "country_code")
	}
	return payload
}
