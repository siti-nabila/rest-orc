package mocks

import "context"

type Authenticator struct {
	AuthenticateFunc func(context.Context, string) (context.Context, error)
	Calls            int
}

func (authenticator *Authenticator) Authenticate(
	ctx context.Context,
	authorization string,
) (context.Context, error) {
	authenticator.Calls++
	return authenticator.AuthenticateFunc(ctx, authorization)
}
