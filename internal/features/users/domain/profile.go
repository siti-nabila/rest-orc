package domain

import "context"

type UpdateProfileRequest struct {
	Name        *string `json:"name"`
	Phone       *string `json:"phone"`
	CountryCode *string `json:"country_code"`
	Address     *string `json:"address"`
}

func (request UpdateProfileRequest) HasFields() bool {
	return request.Name != nil || request.Address != nil || request.Phone != nil || request.CountryCode != nil
}

type ProfileActor struct {
	UserID             uint64
	MustChangePassword bool
}

type Profile struct {
	ID          uint64 `json:"id"`
	UserID      uint64 `json:"user_id"`
	Name        string `json:"name"`
	Address     string `json:"address"`
	Phone       string `json:"phone"`
	CountryCode string `json:"country_code"`
}

type UpdateProfileResponse struct {
	Changed bool     `json:"changed"`
	Version *string  `json:"version,omitempty"`
	Profile *Profile `json:"profile,omitempty"`
}

type ProfileUpdater interface {
	UpdateProfile(context.Context, UpdateProfileRequest) (UpdateProfileResponse, error)
}
