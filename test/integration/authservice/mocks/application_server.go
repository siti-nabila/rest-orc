package mocks

import (
	"context"

	applicationv1 "github.com/siti-nabila/api-contracts/pb/application/v1"
	shared "github.com/siti-nabila/rest-orc/test/shared/fixtures"
)

type ApplicationServer struct {
	applicationv1.UnimplementedApplicationServiceServer
}

func (*ApplicationServer) ListApplications(context.Context, *applicationv1.ListApplicationsRequest) (*applicationv1.ListApplicationsResponse, error) {
	return shared.ApplicationList(), nil
}

func (*ApplicationServer) GrantApplicationAccess(_ context.Context, r *applicationv1.GrantApplicationAccessRequest) (*applicationv1.GrantApplicationAccessResponse, error) {
	s := "ACTIVE"
	return &applicationv1.GrantApplicationAccessResponse{UserId: r.UserId, ApplicationId: 5, AccessStatus: &s, Version: *r.ExpectedUserVersion}, nil
}
func (*ApplicationServer) RevokeApplicationAccess(_ context.Context, r *applicationv1.RevokeApplicationAccessRequest) (*applicationv1.RevokeApplicationAccessResponse, error) {
	s := "REVOKED"
	return &applicationv1.RevokeApplicationAccessResponse{UserId: r.UserId, ApplicationId: 5, AccessStatus: &s, Version: *r.ExpectedUserVersion}, nil
}
