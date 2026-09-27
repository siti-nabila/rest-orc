package auth

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

type Authenticator interface {
	Authenticate(
		ctx context.Context,
		authorization string,
	) (context.Context, error)
}

type Middleware struct {
	authenticator Authenticator
}

func New(authenticator Authenticator) (*Middleware, error) {
	if authenticator == nil {
		return nil, dictionary.ErrAuthenticationMiddlewareRequired
	}
	return &Middleware{authenticator: authenticator}, nil
}

func (middleware *Middleware) Handle(ctx fiber.Ctx) error {
	requestContext, err := middleware.authenticator.Authenticate(
		ctx.Context(),
		ctx.Get(fiber.HeaderAuthorization),
	)
	if err != nil {
		return err
	}

	ctx.SetContext(requestContext)
	return ctx.Next()
}
