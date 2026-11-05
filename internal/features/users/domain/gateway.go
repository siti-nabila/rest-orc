package domain

import (
	"context"
	"time"

	"github.com/siti-nabila/rest-orc/pkg/pagination"
)

type (
	ListFilter struct {
		CreatedFrom *time.Time
		CreatedTo   *time.Time
		RoleCodes   []uint64
		Statuses    []string
	}
	ListQuery struct {
		Page   pagination.Query
		Filter ListFilter
	}

	Gateway interface {
		List(ctx context.Context, query ListQuery) (pagination.Page[ListItem], error)
	}
)
