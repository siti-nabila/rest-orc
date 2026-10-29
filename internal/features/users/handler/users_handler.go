package handler

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	requestauth "github.com/siti-nabila/rest-orc/internal/auth"
	"github.com/siti-nabila/rest-orc/internal/features/users/domain"
	"github.com/siti-nabila/rest-orc/internal/response"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/pkg/pagination"
)

const jakartaUTCOffsetSeconds = 7 * 60 * 60

const (
	Route              = "/users"
	ProfileRoute       = "/profile"
	listSuccessMessage = "Users retrieved successfully."
)

type (
	UpdateProfileUseCase interface {
		Execute(context.Context, domain.ProfileActor, domain.UpdateProfileRequest) (domain.UpdateProfileResponse, error)
	}
	ListUseCase interface {
		Execute(
			ctx context.Context,
			query domain.ListQuery,
		) (pagination.Page[domain.ListItem], error)
	}
	Handler struct {
		listUseCase          ListUseCase
		updateProfileUseCase UpdateProfileUseCase
		responseWriter       *response.Writer
	}
)

func (handler *Handler) RegisterAdminRoutes(router fiber.Router) {
	router.Get("", handler.List)
}

func New(
	listUseCase ListUseCase,
	updateProfileUseCase UpdateProfileUseCase,
	responseWriter *response.Writer,
) (*Handler, error) {
	if listUseCase == nil || updateProfileUseCase == nil {
		return nil, dictionary.ErrUsersHandlerUseCaseRequired
	}
	if responseWriter == nil {
		return nil, dictionary.ErrUsersHandlerResponseWriterRequired
	}
	return &Handler{
		listUseCase:          listUseCase,
		updateProfileUseCase: updateProfileUseCase,
		responseWriter:       responseWriter,
	}, nil
}

func (handler *Handler) List(ctx fiber.Ctx) error {
	query, err := listQueryFromRequest(ctx)
	if err != nil {
		return err
	}

	result, err := handler.listUseCase.Execute(ctx.Context(), query)
	if err != nil {
		return err
	}
	return handler.responseWriter.Write(ctx, response.Result{
		Success: response.Success{Message: listSuccessMessage},
		Data:    result,
	})
}

func listQueryFromRequest(ctx fiber.Ctx) (domain.ListQuery, error) {
	page, err := positiveInt32Query(ctx, "page")
	if err != nil {
		return domain.ListQuery{}, err
	}
	limit, err := positiveInt32Query(ctx, "limit")
	if err != nil {
		return domain.ListQuery{}, err
	}
	createdFrom, err := dateQuery(ctx, "created_from", false)
	if err != nil {
		return domain.ListQuery{}, err
	}
	createdTo, err := dateQuery(ctx, "created_to", true)
	if err != nil {
		return domain.ListQuery{}, err
	}
	if createdFrom != nil && createdTo != nil && !createdFrom.Before(*createdTo) {
		return domain.ListQuery{}, fiber.ErrBadRequest
	}
	roleCodes, err := positiveUint64CSVQuery(ctx, "role")
	if err != nil {
		return domain.ListQuery{}, err
	}

	var search *pagination.Search
	if keyword := strings.TrimSpace(ctx.Query("keyword")); keyword != "" {
		search = &pagination.Search{Keyword: keyword}
	}

	return domain.ListQuery{
		Page: pagination.Query{
			Page:   page,
			Limit:  limit,
			Search: search,
			LastID: ctx.Query("last_id"),
		},
		Filter: domain.ListFilter{
			CreatedFrom: createdFrom,
			CreatedTo:   createdTo,
			RoleCodes:   roleCodes,
		},
	}, nil
}

func dateQuery(ctx fiber.Ctx, key string, endExclusive bool) (*time.Time, error) {
	raw := strings.TrimSpace(ctx.Query(key))
	if raw == "" {
		return nil, nil
	}

	location := time.FixedZone("Asia/Jakarta", jakartaUTCOffsetSeconds)
	value, err := time.ParseInLocation(time.DateOnly, raw, location)
	if err != nil {
		return nil, fiber.ErrBadRequest
	}
	if endExclusive {
		value = value.AddDate(0, 0, 1)
	}
	return &value, nil
}

func positiveUint64CSVQuery(ctx fiber.Ctx, key string) ([]uint64, error) {
	raw := strings.TrimSpace(ctx.Query(key))
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	values := make([]uint64, 0, len(parts))
	seen := make(map[uint64]struct{}, len(parts))
	for _, part := range parts {
		value, err := strconv.ParseUint(strings.TrimSpace(part), 10, 64)
		if err != nil || value == 0 {
			return nil, fiber.ErrBadRequest
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	return values, nil
}

func positiveInt32Query(ctx fiber.Ctx, key string) (int32, error) {
	raw := ctx.Query(key)
	if raw == "" {
		return 0, nil
	}

	parsed, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || parsed <= 0 {
		return 0, fiber.ErrBadRequest
	}
	return int32(parsed), nil
}

func (handler *Handler) RegisterAuthenticatedRoutes(router fiber.Router) {
	router.Patch(ProfileRoute, handler.UpdateProfile)
}

func (handler *Handler) UpdateProfile(ctx fiber.Ctx) error {
	principal, ok := requestauth.PrincipalFromContext(ctx.Context())
	if !ok {
		return common.ErrUnauthorized
	}
	var request domain.UpdateProfileRequest
	if err := ctx.Bind().Body(&request); err != nil {
		return fiber.ErrBadRequest
	}
	result, err := handler.updateProfileUseCase.Execute(ctx.Context(), domain.ProfileActor{
		UserID: principal.ID, MustChangePassword: principal.MustChangePassword,
	}, request)
	if err != nil {
		return err
	}
	return handler.responseWriter.Write(ctx, response.Result{
		Success: response.Success{Message: "Profile updated successfully."}, Data: result,
	})
}
