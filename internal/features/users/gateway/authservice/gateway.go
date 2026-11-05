package authservice

import (
	"context"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/pkg/pagination"
)

type Client interface {
	ListUsers(ctx context.Context, request *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error)
}

type Gateway struct {
	client Client
}

func New(client Client) (*Gateway, error) {
	if client == nil {
		return nil, dictionary.ErrUsersGatewayClientRequired
	}
	return &Gateway{client: client}, nil
}

func (gateway *Gateway) List(
	ctx context.Context,
	query domain.ListQuery,
) (pagination.Page[domain.ListItem], error) {
	if ctx == nil {
		return pagination.Page[domain.ListItem]{},
			dictionary.ErrUsersGatewayContextRequired
	}

	request := listRequest{query: query}
	response, err := gateway.client.ListUsers(ctx, request.protobuf())
	if err != nil {
		return pagination.Page[domain.ListItem]{}, err
	}
	if response == nil {
		return pagination.Page[domain.ListItem]{},
			dictionary.ErrGRPCAuthListUsersResponseRequired
	}
	return (listResponse{value: response}).page()
}
