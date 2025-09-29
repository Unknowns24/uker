package pagination_test

import (
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/unknowns24/uker/uker/pagination"
	"gorm.io/gorm"
	gormtest "gorm.io/gorm/utils/tests"
)

type record struct{}

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(gormtest.DummyDialector{}, &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("open dummy dialector: %v", err)
	}

	return db
}

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

func TestApplyBuildsQuery(t *testing.T) {
	db := openTestDB(t)
	params := pagination.Params{
		Limit: 10,
		Sort: []pagination.SortExpression{
			{Field: "created_at", Direction: pagination.DirectionDesc},
			{Field: "id", Direction: pagination.DirectionDesc},
		},
		Filters: map[string]string{
			"status_eq":    "active",
			"origin_in":    "web,app",
			"country_like": "US%",
		},
		Cursor: &pagination.CursorPayload{After: map[string]string{
			"created_at": time.Now().UTC().Format(time.RFC3339),
			"id":         "01J8",
		}},
	}

	query, err := pagination.Apply(db.Model(&record{}), params)
	if err != nil {
		t.Fatalf("apply params: %v", err)
	}

	stmt := query.Find(&[]record{}).Statement
	if stmt.SQL.Len() == 0 {
		t.Fatalf("expected statement to generate SQL")
	}
	sql := stmt.SQL.String()
	if !strings.Contains(sql, "LIMIT ?") {
		t.Fatalf("expected LIMIT placeholder, got %s", sql)
	}
	if !strings.Contains(sql, "ORDER BY") {
		t.Fatalf("expected ORDER BY clause, got %s", sql)
	}
	if !strings.Contains(sql, "created_at") || !strings.Contains(sql, "id") {
		t.Fatalf("expected sort fields in SQL, got %s", sql)
	}
	if !strings.Contains(sql, "status = ?") {
		t.Fatalf("expected status filter, got %s", sql)
	}
	if !strings.Contains(sql, "origin IN") {
		t.Fatalf("expected IN filter, got %s", sql)
	}
	if !strings.Contains(sql, "country LIKE ?") {
		t.Fatalf("expected LIKE filter, got %s", sql)
	}

	if len(stmt.Vars) == 0 || stmt.Vars[len(stmt.Vars)-1] != params.Limit {
		t.Fatalf("expected limit binding at the end, got %v", stmt.Vars)
	}
}

func TestApplyBeforeCursor(t *testing.T) {
	db := openTestDB(t)

	params := pagination.Params{
		Limit: 5,
		Sort: []pagination.SortExpression{
			{Field: "id", Direction: pagination.DirectionDesc},
		},
		Cursor: &pagination.CursorPayload{Before: map[string]string{"id": "100"}},
	}

	query, err := pagination.Apply(db.Table("records"), params)
	if err != nil {
		t.Fatalf("apply params: %v", err)
	}

	stmt := query.Find(&[]struct{}{}).Statement
	if stmt.SQL.Len() == 0 {
		t.Fatalf("expected statement to generate SQL")
	}
	sql := stmt.SQL.String()
	if !strings.Contains(sql, "id > ?") {
		t.Fatalf("expected inverted comparator for before cursor, got %s", sql)
	}
}

func TestApplyMissingCursorField(t *testing.T) {
	db := openTestDB(t)

	params := pagination.Params{
		Sort:   []pagination.SortExpression{{Field: "created_at", Direction: pagination.DirectionDesc}},
		Cursor: &pagination.CursorPayload{After: map[string]string{"id": "1"}},
	}

	if _, err := pagination.Apply(db, params); !errors.Is(err, pagination.ErrInvalidCursor) {
		t.Fatalf("expected invalid cursor error, got %v", err)
	}
}

func TestBuildNextCursor(t *testing.T) {
	params := pagination.Params{
		Sort: []pagination.SortExpression{
			{Field: "created_at", Direction: pagination.DirectionDesc},
			{Field: "id", Direction: pagination.DirectionDesc},
		},
		Filters: map[string]string{"status_in": "active,inactive"},
	}
	values := map[string]string{
		"created_at": time.Date(2024, 8, 20, 12, 0, 0, 0, time.UTC).Format(time.RFC3339),
		"id":         "01J8",
	}

	cursor, err := pagination.BuildNextCursor(params, values)
	if err != nil {
		t.Fatalf("build next cursor: %v", err)
	}
	if cursor == "" {
		t.Fatalf("expected next cursor to be generated")
	}

	// Mutate sources after building to ensure internal copies were produced.
	params.Sort[0].Field = "mutated"
	params.Filters["status_in"] = "changed"
	values["created_at"] = "bad"

	payload, err := pagination.DecodeCursor(cursor)
	if err != nil {
		t.Fatalf("decode cursor: %v", err)
	}

	if payload.After["created_at"] != "2024-08-20T12:00:00Z" {
		t.Fatalf("unexpected after value: %s", payload.After["created_at"])
	}
	if payload.Sort[0].Field != "created_at" {
		t.Fatalf("expected sort field to remain created_at, got %s", payload.Sort[0].Field)
	}
	if payload.Filters["status_in"] != "active,inactive" {
		t.Fatalf("expected filters to be copied, got %s", payload.Filters["status_in"])
	}
}

func TestBuildPrevCursor(t *testing.T) {
	params := pagination.Params{
		Sort: []pagination.SortExpression{
			{Field: "created_at", Direction: pagination.DirectionDesc},
			{Field: "id", Direction: pagination.DirectionDesc},
		},
		Filters: map[string]string{"origin_eq": "web"},
	}
	values := map[string]string{
		"created_at": time.Date(2024, 8, 18, 9, 0, 0, 0, time.UTC).Format(time.RFC3339),
		"id":         "01J7",
	}

	cursor, err := pagination.BuildPrevCursor(params, values)
	if err != nil {
		t.Fatalf("build prev cursor: %v", err)
	}
	if cursor == "" {
		t.Fatalf("expected prev cursor to be generated")
	}

	payload, err := pagination.DecodeCursor(cursor)
	if err != nil {
		t.Fatalf("decode cursor: %v", err)
	}

	if _, ok := payload.Before["created_at"]; !ok {
		t.Fatalf("expected before map to contain created_at")
	}
	if len(payload.After) != 0 {
		t.Fatalf("expected after map to be empty")
	}
	if payload.Filters["origin_eq"] != "web" {
		t.Fatalf("expected filters to match, got %s", payload.Filters["origin_eq"])
	}
}

func TestBuildCursorEmptyValues(t *testing.T) {
	params := pagination.Params{}

	next, err := pagination.BuildNextCursor(params, nil)
	if err != nil {
		t.Fatalf("build next cursor with nil values: %v", err)
	}
	if next != "" {
		t.Fatalf("expected empty next cursor when no values are provided")
	}

	prev, err := pagination.BuildPrevCursor(params, map[string]string{})
	if err != nil {
		t.Fatalf("build prev cursor with empty values: %v", err)
	}
	if prev != "" {
		t.Fatalf("expected empty prev cursor when no values are provided")
	}
}