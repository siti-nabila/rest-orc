package mocks

import (
	"context"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
)

type ProfileUseCase struct {
	ExecuteFunc func(context.Context, domain.ProfileActor, domain.UpdateProfileRequest) (domain.UpdateProfileResponse, error)
	Calls       int
}

func (mock *ProfileUseCase) Execute(ctx context.Context, actor domain.ProfileActor, request domain.UpdateProfileRequest) (domain.UpdateProfileResponse, error) {
	mock.Calls++
	return mock.ExecuteFunc(ctx, actor, request)
}
