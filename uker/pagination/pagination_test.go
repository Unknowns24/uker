package pagination_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/unknowns24/uker/uker/pagination"
)

func TestEncodeDecodeCursor(t *testing.T) {
	payload := pagination.CursorPayload{
		Version:   1,
		Sort:      []pagination.SortExpression{{Field: "created_at", Direction: pagination.DirectionDesc}, {Field: "id", Direction: pagination.DirectionDesc}},
		Filters:   map[string]string{"status_in": "scheduled,ongoing"},
		After:     map[string]string{"created_at": time.Now().UTC().Format(time.RFC3339), "id": "01J8"},
		Timestamp: time.Now().Unix(),
	}

	encoded, err := pagination.EncodeCursor(payload)
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}

	decoded, err := pagination.DecodeCursor(encoded)
	if err != nil {
		t.Fatalf("decode cursor: %v", err)
	}

	if decoded.Version != payload.Version {
		t.Fatalf("expected version %d, got %d", payload.Version, decoded.Version)
	}
	if len(decoded.Sort) != len(payload.Sort) {
		t.Fatalf("expected %d sort expressions, got %d", len(payload.Sort), len(decoded.Sort))
	}
	if decoded.Filters["status_in"] != payload.Filters["status_in"] {
		t.Fatalf("filters mismatch: %s", decoded.Filters["status_in"])
	}
}

func TestParseWithoutCursor(t *testing.T) {
	values := url.Values{}
	values.Set("limit", "25")
	values.Set("sort", "created_at:desc")
	values.Set("status_in", "scheduled,ongoing")
	values.Set("origin_eq", "ROS")

	params, err := pagination.Parse(values)
	if err != nil {
		t.Fatalf("parse params: %v", err)
	}

	if params.Limit != 25 {
		t.Fatalf("expected limit 25, got %d", params.Limit)
	}

	if len(params.Sort) != 2 {
		t.Fatalf("expected ID to be appended, got %d sorts", len(params.Sort))
	}

	if _, ok := params.Filters["status_in"]; !ok {
		t.Fatalf("expected status_in filter to be present")
	}
	if _, ok := params.Filters["origin_eq"]; !ok {
		t.Fatalf("expected origin_eq filter to be present")
	}
}

func TestParseWithCursor(t *testing.T) {
	cursorPayload := pagination.CursorPayload{
		Version: 1,
		Sort:    []pagination.SortExpression{{Field: "created_at", Direction: pagination.DirectionDesc}, {Field: "id", Direction: pagination.DirectionDesc}},
		Filters: map[string]string{"status_in": "scheduled,ongoing"},
	}

	encoded, err := pagination.EncodeCursor(cursorPayload)
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}

	values := url.Values{}
	values.Set("cursor", encoded)

	params, err := pagination.Parse(values)
	if err != nil {
		t.Fatalf("parse params: %v", err)
	}

	if params.RawCursor == "" {
		t.Fatalf("expected raw cursor to be set")
	}
	if params.Cursor == nil {
		t.Fatalf("expected cursor payload to be populated")
	}
	if len(params.Filters) != 1 {
		t.Fatalf("expected filters from cursor to be included")
	}
}

func TestParseInvalidLimit(t *testing.T) {
	values := url.Values{}
	values.Set("limit", "101")

	if _, err := pagination.Parse(values); err != pagination.ErrLimitOutOfRange {
		t.Fatalf("expected limit out of range, got %v", err)
	}
}

func TestParseInvalidSortOverride(t *testing.T) {
	cursorPayload := pagination.CursorPayload{
		Version: 1,
		Sort:    []pagination.SortExpression{{Field: "created_at", Direction: pagination.DirectionDesc}},
	}
	encoded, err := pagination.EncodeCursor(cursorPayload)
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}

	values := url.Values{}
	values.Set("cursor", encoded)
	values.Set("sort", "created_at:asc")

	if _, err := pagination.Parse(values); err != pagination.ErrInvalidSort {
		t.Fatalf("expected invalid sort override, got %v", err)
	}
}

func TestParseInvalidFilter(t *testing.T) {
	values := url.Values{}
	values.Set("status_between", "one,two")

	if _, err := pagination.Parse(values); err != pagination.ErrInvalidFilter {
		t.Fatalf("expected invalid filter error, got %v", err)
	}
}
