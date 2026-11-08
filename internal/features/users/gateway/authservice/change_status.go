package authservice

import (
	"context"
	"strconv"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"google.golang.org/grpc/metadata"
)

type UserStatusClient interface {
	ChangeUserStatus(context.Context, *userv1.ChangeUserStatusRequest) (*userv1.ChangeUserStatusResponse, error)
}
type UserStatusGateway struct{ client UserStatusClient }

func NewUserStatus(client UserStatusClient) (*UserStatusGateway, error) {
	if client == nil {
		return nil, dictionary.ErrUsersGatewayClientRequired
	}
	return &UserStatusGateway{client: client}, nil
}
func (g *UserStatusGateway) ChangeUserStatus(ctx context.Context, request domain.ChangeUserStatusRequest) (domain.ChangeUserStatusResponse, error) {
	if ctx == nil {
		return domain.ChangeUserStatusResponse{}, dictionary.ErrUsersGatewayContextRequired
	}
	r, err := request.NormalizeAndValidate()
	if err != nil {
		return domain.ChangeUserStatusResponse{}, err
	}
	md, _ := metadata.FromOutgoingContext(ctx)
	md = md.Copy()
	md.Delete("version")
	md.Delete("idempotency-key")
	md.Set("idempotency-key", r.IdempotencyKey)
	// Optional values belong to this RPC, independent of caller-owned pointers.
	status, version := *r.Status, r.Version
	result, err := g.client.ChangeUserStatus(metadata.NewOutgoingContext(ctx, md), &userv1.ChangeUserStatusRequest{UserId: r.UserID, Status: &status, Reason: *r.Reason, ExpectedUserVersion: &version})
	if err != nil {
		return domain.ChangeUserStatusResponse{}, err
	}
	if result == nil || domain.ValidateUserDetailID(result.UserId) != nil || result.UserId != r.UserID || !domain.ValidUserStatus(result.Status) || result.Version == 0 || result.Version > 9223372036854775807 {
		return domain.ChangeUserStatusResponse{}, common.ErrInternalServerError
	}
	return domain.ChangeUserStatusResponse{UserID: result.UserId, Status: result.Status, Version: strconv.FormatUint(result.Version, 10), Changed: result.Changed}, nil
}
