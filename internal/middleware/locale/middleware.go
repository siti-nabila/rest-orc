package locale

import (
	"github.com/gofiber/fiber/v3"
	contractlocale "github.com/siti-nabila/api-contracts/pkg/locale"
	"google.golang.org/grpc/metadata"
)

// Handle normalizes the request language and forwards it to downstream gRPC
// services without discarding existing outgoing metadata.
func Handle(ctx fiber.Ctx) error {
	language := contractlocale.Parse(ctx.Get(contractlocale.HTTPHeader))
	outgoingMetadata, _ := metadata.FromOutgoingContext(ctx.Context())
	outgoingMetadata = outgoingMetadata.Copy()
	if outgoingMetadata == nil {
		outgoingMetadata = metadata.MD{}
	}
	outgoingMetadata.Set(contractlocale.MetadataKey, string(language))
	requestContext := metadata.NewOutgoingContext(ctx.Context(), outgoingMetadata)
	ctx.SetContext(requestContext)
	return ctx.Next()
}
