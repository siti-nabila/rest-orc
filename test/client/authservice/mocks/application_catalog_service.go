package mocks

import (
	"context"

	applicationv1 "github.com/siti-nabila/api-contracts/pb/application/v1"
	"google.golang.org/grpc"
)

type ApplicationCatalogService struct {
	List func(context.Context, *applicationv1.ListApplicationsRequest) (*applicationv1.ListApplicationsResponse, error)
}

func (s *ApplicationCatalogService) ListApplications(c context.Context, r *applicationv1.ListApplicationsRequest, _ ...grpc.CallOption) (*applicationv1.ListApplicationsResponse, error) {
	if s.List == nil {
		panic("unexpected ListApplications")
	}
	return s.List(c, r)
}
