package authservice

import (
	"context"
	"strconv"

	profilev1 "github.com/siti-nabila/api-contracts/pb/profile/v1"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type ProfileClient interface {
	UpdateProfile(context.Context, *profilev1.ProfileRequest) (*profilev1.ProfileResponse, error)
}

type ProfileGateway struct{ client ProfileClient }

func NewProfile(client ProfileClient) (*ProfileGateway, error) {
	if client == nil {
		return nil, dictionary.ErrUsersGatewayClientRequired
	}
	return &ProfileGateway{client: client}, nil
}

func (gateway *ProfileGateway) UpdateProfile(ctx context.Context, request domain.UpdateProfileRequest) (domain.UpdateProfileResponse, error) {
	if ctx == nil {
		return domain.UpdateProfileResponse{}, dictionary.ErrUsersGatewayContextRequired
	}
	if !request.HasFields() {
		return domain.UpdateProfileResponse{}, common.ErrBadRequest
	}
	payload := &profilev1.ProfileRequest{FieldMask: &fieldmaskpb.FieldMask{}}
	if request.Name != nil {
		payload.Name = *request.Name
		payload.FieldMask.Paths = append(payload.FieldMask.Paths, "name")
	}
	if request.Address != nil {
		payload.Address = *request.Address
		payload.FieldMask.Paths = append(payload.FieldMask.Paths, "address")
	}
	if request.Phone != nil {
		payload.Phone = *request.Phone
		payload.FieldMask.Paths = append(payload.FieldMask.Paths, "phone")
	}
	if request.CountryCode != nil {
		country := *request.CountryCode
		payload.CountryCode = &country
		payload.FieldMask.Paths = append(payload.FieldMask.Paths, "country_code")
	}
	result, err := gateway.client.UpdateProfile(ctx, payload)
	if err != nil {
		return domain.UpdateProfileResponse{}, err
	}
	if result == nil || result.Profile == nil || result.Profile.Id == 0 || result.Profile.UserId == 0 || result.Version == 0 {
		return domain.UpdateProfileResponse{}, common.ErrInternalServerError
	}
	profile := result.Profile
	version := strconv.FormatUint(result.Version, 10)
	return domain.UpdateProfileResponse{
		Changed: result.Changed,
		Version: &version,
		Profile: &domain.Profile{
			ID:          profile.Id,
			UserID:      profile.UserId,
			Name:        profile.Name,
			Address:     profile.Address,
			Phone:       profile.Phone,
			CountryCode: profile.CountryCode,
		},
	}, nil
}
