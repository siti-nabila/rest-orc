package response

import (
	"context"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/api-contracts/pkg/dictionary"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/api-contracts/pkg/grpcerror"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	contractresponse "github.com/siti-nabila/api-contracts/pkg/response"
	"google.golang.org/grpc/codes"
)

type MappedError struct {
	Status     int
	Body       any
	RetryAfter string
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
	if fields, ok := localFieldErrors(err, language); ok {
		return fieldError(fiber.StatusBadRequest, common.ErrBadRequest.Code(), fields)
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
	if errors.Is(err, context.DeadlineExceeded) {
		return mappedDefinition(common.ErrDeadlineExceeded, language)
	}
	if errors.Is(err, context.Canceled) {
		return mappedDefinition(common.ErrRequestCanceled, language)
	}

	return internalError(language)
}

func (mapper *ErrorMapper) mapGRPCError(
	decoded grpcerror.Decoded,
	language locale.Language,
) MappedError {
	httpStatus, fallback := grpcFallback(decoded.Code)
	code := fallback.Code()
	message := fallback.Message(language)
	if definition, exists := decoded.Definition(mapper.registries...); exists {
		httpStatus = definition.HTTPStatus()
		code = definition.Code()
		message = definition.Message(language)
	}

	if explicitStatus, err := strconv.Atoi(decoded.Metadata["http_status"]); err == nil && explicitStatus >= 400 && explicitStatus <= 599 {
		httpStatus = explicitStatus
	}
	if publicCode := decoded.Metadata["code"]; publicCode != "" {
		code = publicCode
	}
	mapped := descriptionError(httpStatus, code, message)
	if decoded.HasFieldErrors() {
		mapped = fieldError(httpStatus, code, decoded.FieldErrors)
	}
	mapped.RetryAfter = retryAfter(decoded.RetryInfo)
	return mapped
}

func grpcFallback(code codes.Code) (int, *dictionary.Error) {
	switch code {
	case codes.InvalidArgument, codes.FailedPrecondition:
		return fiber.StatusBadRequest, common.ErrBadRequest
	case codes.Unauthenticated:
		return fiber.StatusUnauthorized, common.ErrUnauthorized
	case codes.PermissionDenied:
		return fiber.StatusForbidden, common.ErrForbidden
	case codes.NotFound:
		return fiber.StatusNotFound, common.ErrNotFound
	case codes.AlreadyExists, codes.Aborted:
		return fiber.StatusConflict, common.ErrInternalServerError
	case codes.ResourceExhausted:
		return fiber.StatusTooManyRequests, common.ErrRateLimited
	case codes.Unavailable:
		return fiber.StatusServiceUnavailable, common.ErrServiceUnavailable
	case codes.DeadlineExceeded:
		return fiber.StatusGatewayTimeout, common.ErrDeadlineExceeded
	case codes.Canceled:
		return common.ErrRequestCanceled.HTTPStatus(), common.ErrRequestCanceled
	default:
		return fiber.StatusInternalServerError, common.ErrInternalServerError
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
	case fiber.StatusUnsupportedMediaType:
		return descriptionError(fiber.StatusUnsupportedMediaType, common.ErrBadRequest.Code(), common.ErrBadRequest.Message(language))
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

func fieldError(status int, code string, fields map[string][]string) MappedError {
	return MappedError{
		Status: status,
		Body: contractresponse.FieldError{
			Code:   code,
			Errors: fields,
			Data:   contractresponse.EmptyData(),
		},
	}
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
