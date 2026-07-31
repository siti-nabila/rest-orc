package response

import (
	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/api-contracts/pkg/locale"
	contractresponse "github.com/siti-nabila/api-contracts/pkg/response"
)

const successCode = "SS"

type Success struct {
	Message string
}

type Result struct {
	Success Success
	Data    any
	Err     error
}

type Writer struct {
	errorMapper *ErrorMapper
}

func NewWriter(errorMapper *ErrorMapper) *Writer {
	if errorMapper == nil {
		errorMapper = NewErrorMapper()
	}
	return &Writer{errorMapper: errorMapper}
}

func (writer *Writer) Write(ctx fiber.Ctx, result Result) error {
	if result.Err != nil {
		language := requestLanguage(ctx)
		mapped := writer.errorMapper.Map(result.Err, language)
		return ctx.Status(mapped.Status).JSON(mapped.Body)
	}

	return ctx.Status(fiber.StatusOK).JSON(contractresponse.Success{
		Code:    successCode,
		Message: result.Success.Message,
		Data:    result.Data,
	})
}

func requestLanguage(ctx fiber.Ctx) locale.Language {
	language := ctx.Get(locale.HTTPHeader)
	if language == "" {
		language = ctx.Get(locale.MetadataKey)
	}
	return locale.Parse(language)
}
