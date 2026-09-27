package grpcauth

import (
	"context"
	"fmt"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/pkg/pagination"
)

type Client interface {
	ListUsers(ctx context.Context, request *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error)
	Register(ctx context.Context, request *userv1.AuthRequest) (*userv1.UserTokenResponse, error)
	Login(ctx context.Context, request *userv1.AuthRequest) (*userv1.UserTokenResponse, error)
}

type Repository struct {
	client Client
}

func New(client Client) (*Repository, error) {
	if client == nil {
		return nil, dictionary.ErrUsersRepositoryClientRequired
	}
	return &Repository{client: client}, nil
}

func (repository *Repository) List(
	ctx context.Context,
	query domain.ListQuery,
) (pagination.Page[domain.ListItem], error) {
	if ctx == nil {
		return pagination.Page[domain.ListItem]{},
			dictionary.ErrUsersRepositoryContextRequired
	}

	request := listRequest{query: query}
	response, err := repository.client.ListUsers(ctx, request.protobuf())
	if err != nil {
		return pagination.Page[domain.ListItem]{}, err
	}
	if response == nil {
		return pagination.Page[domain.ListItem]{},
			dictionary.ErrGRPCAuthListUsersResponseRequired
	}
	return (listResponse{value: response}).page()
}

func (r *Repository) Register(ctx context.Context, request domain.AuthRequest) (domain.AuthResponse, error) {
	if ctx == nil {
		return domain.AuthResponse{}, dictionary.ErrUsersRepositoryContextRequired
	}

	fmt.Println("------- masuk repo register -----")

	protobufRequest := &userv1.AuthRequest{
		Email:    request.Email,
		Password: request.Password,
	}
	response, err := r.client.Register(ctx, protobufRequest)
	if err != nil {
		return domain.AuthResponse{}, err
	}
	if response == nil {
		return domain.AuthResponse{}, dictionary.ErrGRPCAuthRegisterResponseRequired
	}
	return domain.AuthResponse{
		Token: response.Token,
	}, nil
}

func (r *Repository) Login(ctx context.Context, request domain.AuthRequest) (domain.AuthResponse, error) {
	if ctx == nil {
		return domain.AuthResponse{}, dictionary.ErrUsersRepositoryContextRequired
	}
	fmt.Println("------- masuk repo login -----")
	protobufRequest := &userv1.AuthRequest{
		Email:    request.Email,
		Password: request.Password,
	}
	response, err := r.client.Login(ctx, protobufRequest)
	if err != nil {
		return domain.AuthResponse{}, err
	}
	if response == nil {
		return domain.AuthResponse{}, dictionary.ErrGRPCAuthLoginResponseRequired
	}
	return domain.AuthResponse{
		Token: response.Token,
	}, nil
}
