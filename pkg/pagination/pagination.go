package pagination

type SearchMode int32

const (
	SearchModeUnspecified SearchMode = iota
	SearchModeContains
	SearchModePrefix
	SearchModeFullText
	SearchModeTrigram
	SearchModeFullTextTrigram
)

type Sort struct {
	Field string
	Desc  bool
}

type Search struct {
	Fields  []string
	Keyword string
	Mode    SearchMode
}

type Query struct {
	Page   int32
	Limit  int32
	Fields []string
	Sort   []Sort
	Search *Search
	LastID string
	Filter map[string]string
}

type Page[T any] struct {
	Items      []T    `json:"items"`
	Total      int32  `json:"total"`
	Page       int32  `json:"page"`
	Limit      int32  `json:"limit"`
	TotalPages int32  `json:"total_pages"`
	HasNext    bool   `json:"has_next"`
	HasPrev    bool   `json:"has_prev"`
	NextCursor string `json:"next_cursor"`
}
