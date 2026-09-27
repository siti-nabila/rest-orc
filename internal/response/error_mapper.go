package response

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/api-contracts/pkg/dictionary"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/api-contracts/pkg/grpcerror"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	contractresponse "github.com/siti-nabila/api-contracts/pkg/response"
	"google.golang.org/grpc/codes"
)

type MappedError struct {
	Status int
	Body   any
}

type ErrorMapper struct {
	registries []dictionary.Registry
}

func NewErrorMapper(registries ...dictionary.Registry) *ErrorMapper {
	return &ErrorMapper{
		registries: append([]dictionary.Registry(nil), registries...),
	}
}

func (mapper *ErrorMapper) Map(
	err error,
	language locale.Language,
) MappedError {
	if decoded, ok := grpcerror.Decode(err); ok {
		return mapper.mapGRPCError(decoded, language)
	}

	if applicationError, ok := errors.AsType[*dictionary.Error](err); ok {
		return descriptionError(
			applicationError.HTTPStatus(),
			applicationError.Code(),
			applicationError.Message(language),
		)
	}

	if fiberError, ok := errors.AsType[*fiber.Error](err); ok {
		return mapper.mapFiberError(fiberError, language)
	}

	return internalError(language)
}

func (mapper *ErrorMapper) mapGRPCError(
	decoded grpcerror.Decoded,
	language locale.Language,
) MappedError {
	if decoded.HasFieldErrors() {
		return MappedError{
			Status: common.ErrBadRequest.HTTPStatus(),
			Body: contractresponse.FieldError{
				Code:   common.ErrBadRequest.Code(),
				Errors: decoded.FieldErrors,
				Data:   contractresponse.EmptyData(),
			},
		}
	}

	if definition, exists := decoded.Definition(mapper.registries...); exists {
		return descriptionError(
			definition.HTTPStatus(),
			definition.Code(),
			definition.Message(language),
		)
	}

	return mapper.mapUnregisteredGRPCCode(decoded.Code, language)
}

func (mapper *ErrorMapper) mapUnregisteredGRPCCode(
	code codes.Code,
	language locale.Language,
) MappedError {
	switch code {
	case codes.InvalidArgument:
		return mappedDefinition(common.ErrBadRequest, language)
	case codes.Unauthenticated:
		return mappedDefinition(common.ErrUnauthorized, language)
	case codes.PermissionDenied:
		return mappedDefinition(common.ErrForbidden, language)
	case codes.NotFound:
		return mappedDefinition(common.ErrNotFound, language)
	case codes.Unavailable:
		return mappedDefinition(common.ErrServiceUnavailable, language)
	case codes.DeadlineExceeded:
		return mappedDefinition(common.ErrDeadlineExceeded, language)
	default:
		return internalError(language)
	}
}

func (mapper *ErrorMapper) mapFiberError(
	err *fiber.Error,
	language locale.Language,
) MappedError {
	switch err.Code {
	case fiber.StatusBadRequest:
		return mappedDefinition(common.ErrBadRequest, language)
	case fiber.StatusUnauthorized:
		return mappedDefinition(common.ErrUnauthorized, language)
	case fiber.StatusForbidden:
		return mappedDefinition(common.ErrForbidden, language)
	case fiber.StatusNotFound:
		return mappedDefinition(common.ErrEndpointNotFound, language)
	default:
		return internalError(language)
	}
}

func mappedDefinition(
	err *dictionary.Error,
	language locale.Language,
) MappedError {
	return descriptionError(
		err.HTTPStatus(),
		err.Code(),
		err.Message(language),
	)
}

func internalError(language locale.Language) MappedError {
	return mappedDefinition(common.ErrInternalServerError, language)
}

func descriptionError(status int, code string, description string) MappedError {
	return MappedError{
		Status: status,
		Body: contractresponse.DescriptionError{
			Code: code,
			Errors: contractresponse.Description{
				Description: description,
			},
			Data: contractresponse.EmptyData(),
		},
	}
}
