package grpcclient

import (
	"context"

	authdictionary "github.com/siti-nabila/api-contracts/pkg/dictionary/auth"
	"google.golang.org/grpc/metadata"
)

const serviceTokenMetadataKey = "x-service-token"

type serviceCredentials struct {
	token string
}

func (c serviceCredentials) RequireTransportSecurity() bool { return true }

func (c serviceCredentials) GetRequestMetadata(ctx context.Context, _ ...string) (map[string]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	md, _ := metadata.FromOutgoingContext(ctx)
	if len(md.Get(serviceTokenMetadataKey)) != 0 {
		return nil, authdictionary.ErrServiceAuthenticationFailed
	}
	return map[string]string{serviceTokenMetadataKey: c.token}, nil
}
