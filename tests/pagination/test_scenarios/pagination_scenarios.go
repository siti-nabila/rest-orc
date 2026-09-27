package test_scenarios

import (
	"encoding/json"
	"testing"

	"github.com/siti-nabila/rest-orc/pkg/pagination"
	"github.com/siti-nabila/rest-orc/tests/shared/testutils"
)

const emptyPageJSON = `{"items":[],"total":0,"page":1,"limit":10,"total_pages":0,"has_next":false,"has_prev":false,"next_cursor":""}`

func All() []testutils.Scenario {
	return []testutils.Scenario{
		{
			Name: "page JSON keeps empty and false paginator fields",
			Run:  pageJSONKeepsZeroValues,
		},
	}
}

func pageJSONKeepsZeroValues(t *testing.T) {
	// Arrange
	page := pagination.Page[string]{
		Items: make([]string, 0),
		Page:  1,
		Limit: 10,
	}

	// Act
	encoded, err := json.Marshal(page)

	// Assert
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if string(encoded) != emptyPageJSON {
		t.Errorf("page JSON = %s, want %s", encoded, emptyPageJSON)
	}
}
