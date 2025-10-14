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

var secret = []byte("integration-secret")

type record struct{}

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(gormtest.DummyDialector{}, &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("open dummy dialector: %v", err)
	}

	return db
}

func setAllowedColumns(t *testing.T, allowed map[string]struct{}) {
	original := pagination.AllowedColumns
	pagination.AllowedColumns = allowed
	t.Cleanup(func() { pagination.AllowedColumns = original })
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

func TestEncodeDecodeCursorSigned(t *testing.T) {
	payload := pagination.CursorPayload{
		Version: 1,
		Sort: []pagination.SortExpression{
			{Field: "created_at", Direction: pagination.DirectionDesc},
			{Field: "id", Direction: pagination.DirectionDesc},
		},
		Filters:   map[string]string{"status_eq": "active"},
		After:     map[string]string{"created_at": time.Now().UTC().Format(time.RFC3339), "id": "01J8"},
		Timestamp: time.Now().Unix(),
	}

	encoded, err := pagination.EncodeCursorSigned(payload, secret)
	if err != nil {
		t.Fatalf("encode cursor signed: %v", err)
	}

	decoded, err := pagination.DecodeCursorSigned(encoded, secret, time.Hour)
	if err != nil {
		t.Fatalf("decode cursor signed: %v", err)
	}

	if decoded.Signature == "" {
		t.Fatalf("expected signature to be preserved")
	}

	if decoded.After["id"] != payload.After["id"] {
		t.Fatalf("expected after id %q, got %q", payload.After["id"], decoded.After["id"])
	}
}

func TestDecodeCursorSignedRejectsTampering(t *testing.T) {
	payload := pagination.CursorPayload{Version: 1}
	encoded, err := pagination.EncodeCursorSigned(payload, secret)
	if err != nil {
		t.Fatalf("encode cursor signed: %v", err)
	}

	tampered := encoded[:len(encoded)-2] + "zz"
	if _, err := pagination.DecodeCursorSigned(tampered, secret, time.Hour); err != pagination.ErrInvalidCursor {
		t.Fatalf("expected invalid cursor error, got %v", err)
	}
}

func TestParseWithoutCursor(t *testing.T) {
	setAllowedColumns(t, nil)

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

func TestParseAllowsUnderscoreFieldFilters(t *testing.T) {
	setAllowedColumns(t, nil)

	values := url.Values{}
	values.Set("document_number_like", "46")

	params, err := pagination.Parse(values)
	if err != nil {
		t.Fatalf("parse params: %v", err)
	}

	if got := params.Filters["document_number_like"]; got != "46" {
		t.Fatalf("expected filter value 46, got %q", got)
	}
}

func TestParseWithCursor(t *testing.T) {
	setAllowedColumns(t, nil)

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

func TestParseWithSecurityBlocksFilterOverrides(t *testing.T) {
	cursorPayload := pagination.CursorPayload{
		Version: 1,
		Sort:    []pagination.SortExpression{{Field: "created_at", Direction: pagination.DirectionDesc}},
		Filters: map[string]string{"status_eq": "active"},
	}

	encoded, err := pagination.EncodeCursorSigned(cursorPayload, secret)
	if err != nil {
		t.Fatalf("encode cursor signed: %v", err)
	}

	values := url.Values{}
	values.Set("cursor", encoded)
	values.Set("status_eq", "active")

	if _, err := pagination.ParseWithSecurity(values, secret, time.Hour); err != pagination.ErrInvalidFilter {
		t.Fatalf("expected filter override error, got %v", err)
	}
}

func TestParseWithSecurityExpiredCursor(t *testing.T) {
	cursorPayload := pagination.CursorPayload{
		Version:   1,
		Sort:      []pagination.SortExpression{{Field: "created_at", Direction: pagination.DirectionDesc}},
		Filters:   map[string]string{"status_eq": "active"},
		Timestamp: time.Now().Add(-2 * time.Hour).Unix(),
	}

	encoded, err := pagination.EncodeCursorSigned(cursorPayload, secret)
	if err != nil {
		t.Fatalf("encode cursor signed: %v", err)
	}

	values := url.Values{}
	values.Set("cursor", encoded)

	if _, err := pagination.ParseWithSecurity(values, secret, time.Hour); err != pagination.ErrCursorExpired {
		t.Fatalf("expected cursor expired error, got %v", err)
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
			"country_like": "US",
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

	if len(stmt.Vars) == 0 {
		t.Fatalf("expected statement vars to include limit")
	}
	if limitValue, ok := stmt.Vars[len(stmt.Vars)-1].(int); !ok || limitValue != params.Limit+1 {
		t.Fatalf("expected final var to be limit+1 (%d), got %v", params.Limit+1, stmt.Vars[len(stmt.Vars)-1])
	}
}

func TestApplyAllowsUnderscoreFieldFilters(t *testing.T) {
	db := openTestDB(t)

	params := pagination.Params{
		Filters: map[string]string{
			"document_number_like": "46",
		},
	}

	query, err := pagination.Apply(db.Table("students"), params)
	if err != nil {
		t.Fatalf("apply params: %v", err)
	}

	stmt := query.Find(&[]struct{}{}).Statement
	sql := stmt.SQL.String()
	if !strings.Contains(sql, "document_number LIKE ?") {
		t.Fatalf("expected LIKE filter to target document_number, got %s", sql)
	}
}

func TestApplyRejectsUnsafeIdentifier(t *testing.T) {
	db := openTestDB(t)
	params := pagination.Params{
		Limit: 5,
		Sort:  []pagination.SortExpression{{Field: "created_at;drop", Direction: pagination.DirectionDesc}},
	}

	if _, err := pagination.Apply(db, params); !errors.Is(err, pagination.ErrInvalidSort) {
		t.Fatalf("expected invalid sort error, got %v", err)
	}
}

func TestApplyHonoursAllowedColumns(t *testing.T) {
	db := openTestDB(t)
	setAllowedColumns(t, map[string]struct{}{"created_at": {}, "id": {}, "status": {}})

	params := pagination.Params{
		Limit: 5,
		Sort: []pagination.SortExpression{
			{Field: "created_at", Direction: pagination.DirectionDesc},
			{Field: "id", Direction: pagination.DirectionDesc},
		},
		Filters: map[string]string{"status_eq": "active"},
	}

	if _, err := pagination.Apply(db, params); err != nil {
		t.Fatalf("unexpected error when identifiers are allowed: %v", err)
	}

	params.Sort = []pagination.SortExpression{{Field: "unknown", Direction: pagination.DirectionDesc}}
	if _, err := pagination.Apply(db, params); !errors.Is(err, pagination.ErrInvalidSort) {
		t.Fatalf("expected invalid sort due to whitelist, got %v", err)
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

func TestBuildNextCursorSignedClonesInput(t *testing.T) {
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

	cursor, err := pagination.BuildNextCursorSigned(params, values, secret)
	if err != nil {
		t.Fatalf("build next cursor signed: %v", err)
	}
	if cursor == "" {
		t.Fatalf("expected next cursor to be generated")
	}

	params.Sort[0].Field = "mutated"
	params.Filters["status_in"] = "changed"
	values["created_at"] = "bad"

	payload, err := pagination.DecodeCursorSigned(cursor, secret, time.Hour)
	if err != nil {
		t.Fatalf("decode cursor signed: %v", err)
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

func TestBuildPrevCursorSigned(t *testing.T) {
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

	cursor, err := pagination.BuildPrevCursorSigned(params, values, secret)
	if err != nil {
		t.Fatalf("build prev cursor signed: %v", err)
	}
	if cursor == "" {
		t.Fatalf("expected prev cursor to be generated")
	}

	payload, err := pagination.DecodeCursorSigned(cursor, secret, time.Hour)
	if err != nil {
		t.Fatalf("decode cursor signed: %v", err)
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
