package authservice

import (
	"context"

	authv1 "github.com/siti-nabila/api-contracts/pb/auth/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"google.golang.org/grpc/metadata"
)

type SessionClient interface {
	RefreshSession(context.Context, *authv1.RefreshSessionRequest) (*userv1.UserTokenResponse, error)
	Logout(context.Context, *authv1.LogoutRequest) (*authv1.LogoutResponse, error)
}
type SessionGateway struct{ client SessionClient }

func NewSession(client SessionClient) (*SessionGateway, error) {
	if client == nil {
		return nil, dictionary.ErrAuthGatewayClientRequired
	}
	return &SessionGateway{client: client}, nil
}

func (g *SessionGateway) Refresh(ctx context.Context, token string) (domain.SessionResult, error) {
	if ctx == nil {
		return domain.SessionResult{}, dictionary.ErrAuthGatewayContextRequired
	}
	if token == "" {
		return domain.SessionResult{}, dictionary.ErrSessionRequestRequired
	}
	result, err := g.client.RefreshSession(credentialContext(ctx, ""), &authv1.RefreshSessionRequest{RefreshToken: token})
	if err != nil {
		return domain.SessionResult{}, err
	}
	return sessionResult(result)
}

func (g *SessionGateway) Logout(ctx context.Context, proof domain.LogoutCredential) (domain.LogoutResponse, error) {
	if ctx == nil {
		return domain.LogoutResponse{}, dictionary.ErrAuthGatewayContextRequired
	}
	if !proof.Valid() {
		return domain.LogoutResponse{}, dictionary.ErrSessionRequestRequired
	}
	request := &authv1.LogoutRequest{}
	if proof.RefreshToken != "" {
		value := proof.RefreshToken
		request.RefreshToken = &value
	}
	result, err := g.client.Logout(credentialContext(ctx, proof.Authorization), request)
	if err != nil {
		return domain.LogoutResponse{}, err
	}
	if result == nil {
		return domain.LogoutResponse{}, dictionary.ErrSessionResultInvalid
	}
	return domain.LogoutResponse{Changed: result.Changed}, nil
}

func credentialContext(ctx context.Context, authorization string) context.Context {
	md, _ := metadata.FromOutgoingContext(ctx)
	md = md.Copy()
	if md == nil {
		md = metadata.MD{}
	}
	md.Delete("authorization")
	if authorization != "" {
		md.Set("authorization", authorization)
	}
	return metadata.NewOutgoingContext(ctx, md)
}

func sessionResult(result *userv1.UserTokenResponse) (domain.SessionResult, error) {
	if result == nil || result.Token == "" || result.RefreshToken == "" || result.SessionId == "" || result.AccessExpiresAt == nil || result.SessionExpiresAt == nil || result.AccessExpiresAt.CheckValid() != nil || result.SessionExpiresAt.CheckValid() != nil {
		return domain.SessionResult{}, dictionary.ErrSessionResultInvalid
	}
	access, session := result.AccessExpiresAt.AsTime().UTC(), result.SessionExpiresAt.AsTime().UTC()
	if access.After(session) {
		return domain.SessionResult{}, dictionary.ErrSessionResultInvalid
	}
	return domain.SessionResult{Token: result.Token, RefreshToken: result.RefreshToken, SessionID: result.SessionId, AccessExpiresAt: access, SessionExpiresAt: session, MustChangePassword: result.MustChangePassword}, nil
}
