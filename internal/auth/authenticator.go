package auth

import (
	"context"
	"strings"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"google.golang.org/grpc/metadata"
)

const authorizationMetadataKey = "authorization"

type MeClient interface {
	Me(ctx context.Context) (*userv1.UserData, error)
}

type Authenticator struct {
	client MeClient
}

func NewAuthenticator(client MeClient) (*Authenticator, error) {
	if client == nil {
		return nil, dictionary.ErrAuthenticatorClientRequired
	}
	return &Authenticator{client: client}, nil
}

func (authenticator *Authenticator) Authenticate(
	ctx context.Context,
	authorization string,
) (context.Context, error) {
	if ctx == nil {
		return nil, dictionary.ErrAuthenticatorContextRequired
	}
	if strings.TrimSpace(authorization) == "" {
		return nil, common.ErrUnauthorized
	}

	outboundContext := withAuthorization(ctx, authorization)
	user, err := authenticator.client.Me(outboundContext)
	if err != nil {
		return nil, err
	}

	principal := Principal{
		ID:        user.GetId(),
		Email:     user.GetEmail(),
		Fullname:  user.GetFullname(),
		RoleCodes: append([]int32(nil), user.GetRoleCodes()...),
		RoleNames: append([]string(nil), user.GetRoleNames()...),
	}
	return WithPrincipal(outboundContext, principal), nil
}

func withAuthorization(ctx context.Context, authorization string) context.Context {
	outgoingMetadata, _ := metadata.FromOutgoingContext(ctx)
	outgoingMetadata = outgoingMetadata.Copy()
	if outgoingMetadata == nil {
		outgoingMetadata = metadata.MD{}
	}
	outgoingMetadata.Set(authorizationMetadataKey, authorization)
	return metadata.NewOutgoingContext(ctx, outgoingMetadata)
}
