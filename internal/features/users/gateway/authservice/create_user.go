package authservice

import (
	"context"
	"strings"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"google.golang.org/grpc/metadata"
)

type UserCreationClient interface {
	CreateUser(context.Context, *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error)
}
type UserCreationGateway struct{ client UserCreationClient }

func NewUserCreation(client UserCreationClient) (*UserCreationGateway, error) {
	if client == nil {
		return nil, dictionary.ErrUsersGatewayClientRequired
	}
	return &UserCreationGateway{client: client}, nil
}

// CreateUser copies values already normalized once by the usecase. The backend
// owns account identity, canonical email, password hashing, membership and replay.
func (g *UserCreationGateway) CreateUser(ctx context.Context, request domain.CreateUserRequest) (domain.CreateUserResponse, error) {
	if ctx == nil {
		return domain.CreateUserResponse{}, dictionary.ErrUsersGatewayContextRequired
	}
	if err := domain.ValidateSessionIdempotencyKey(request.IdempotencyKey); err != nil {
		return domain.CreateUserResponse{}, err
	}
	b := request.Body
	if b.CountryCode == nil || *b.CountryCode == "" || b.Email == nil || b.Password == nil || b.Name == nil || b.Phone == nil || (b.Status != nil && *b.Status != "PENDING") {
		return domain.CreateUserResponse{}, common.ErrBadRequest
	}
	country, status := *b.CountryCode, "PENDING"
	// Save expected identity before passing dependency-owned payload pointers.
	email := strings.ToLower(strings.TrimSpace(*b.Email))
	payload := &userv1.CreateUserRequest{Name: *b.Name, Email: *b.Email, Password: *b.Password, Phone: *b.Phone, CountryCode: &country, Status: &status}
	if b.Address != nil {
		payload.Address = *b.Address
	}
	md, _ := metadata.FromOutgoingContext(ctx)
	md = md.Copy()
	md.Delete("version")
	md.Delete("idempotency-key")
	md.Set("idempotency-key", request.IdempotencyKey)
	result, err := g.client.CreateUser(metadata.NewOutgoingContext(ctx, md), payload)
	if err != nil {
		return domain.CreateUserResponse{}, err
	}
	if result == nil {
		return domain.CreateUserResponse{}, common.ErrInternalServerError
	}
	user, err := safeUserDetail(result.User)
	if err != nil {
		return domain.CreateUserResponse{}, err
	}
	if user.Email != email || user.Status != "PENDING" {
		return domain.CreateUserResponse{}, common.ErrInternalServerError
	}
	return domain.CreateUserResponse{User: &user}, nil
}
