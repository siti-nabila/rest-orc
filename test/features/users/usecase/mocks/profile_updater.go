package mocks

import (
	"context"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
)

type ProfileUpdater struct {
	UpdateFunc func(context.Context, domain.UpdateProfileRequest) (domain.UpdateProfileResponse, error)
	Calls      int
}

func (mock *ProfileUpdater) UpdateProfile(ctx context.Context, request domain.UpdateProfileRequest) (domain.UpdateProfileResponse, error) {
	mock.Calls++
	return mock.UpdateFunc(ctx, request)
}
