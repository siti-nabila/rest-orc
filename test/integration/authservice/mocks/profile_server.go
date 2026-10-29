package mocks

import (
	"context"
	profilev1 "github.com/siti-nabila/api-contracts/pb/profile/v1"
	usersfixtures "github.com/siti-nabila/rest-orc/test/integration/users/fixtures"
)

type ProfileServer struct {
	profilev1.UnimplementedProfileServiceServer
}

func (*ProfileServer) UpdateProfile(context.Context, *profilev1.ProfileRequest) (*profilev1.ProfileResponse, error) {
	return usersfixtures.ProfileResponse(), nil
}
