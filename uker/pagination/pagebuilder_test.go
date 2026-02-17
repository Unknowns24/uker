package pagination_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/unknowns24/uker/uker/pagination"
)

var pageSecret = []byte("pagebuilder-secret")

type member struct {
	ID        string    `gorm:"column:id"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

type baseModel struct {
	ID string `gorm:"column:id"`
}

type order struct {
	baseModel
	CreatedAt time.Time `gorm:"column:created_at"`
}

func TestBuildPage_FirstPageWithoutCursor(t *testing.T) {
	raw := url.Values{}
	raw.Set("limit", "2")
	raw.Set("sort", "created_at:asc")

	params, err := pagination.ParseWithSecurity(raw, pageSecret, time.Hour)
	if err != nil {
		t.Fatalf("parse params: %v", err)
	}

	members := []member{
		{ID: "mem-1", CreatedAt: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)},
		{ID: "mem-2", CreatedAt: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)},
		{ID: "mem-3", CreatedAt: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)},
	}

	pageTotal := int64(len(members))
	page, err := pagination.BuildPageSigned[member](params, members, params.Limit, pageTotal, nil, pageSecret)
	if err != nil {
		t.Fatalf("BuildPage returned error: %v", err)
	}

	if len(page.Data) != params.Limit {
		t.Fatalf("expected %d items, got %d", params.Limit, len(page.Data))
	}

	if page.Paging.Total != pageTotal {
		t.Fatalf("expected total %d, got %d", pageTotal, page.Paging.Total)
	}

	if !page.Paging.HasMore {
		t.Fatalf("expected hasMore to be true")
	}

	if page.Paging.NextCursor == "" {
		t.Fatalf("expected next cursor to be generated")
	}

	if page.Paging.PrevCursor != "" {
		t.Fatalf("expected prev cursor to be empty on first page")
	}
}

func TestBuildPage_GeneratesPrevCursorForSubsequentPage(t *testing.T) {
	raw := url.Values{}
	raw.Set("limit", "2")
	raw.Set("sort", "created_at:asc")

	params, err := pagination.ParseWithSecurity(raw, pageSecret, time.Hour)
	if err != nil {
		t.Fatalf("parse params: %v", err)
	}

	firstMembers := []member{
		{ID: "mem-1", CreatedAt: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)},
		{ID: "mem-2", CreatedAt: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)},
		{ID: "mem-3", CreatedAt: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)},
	}

	firstTotal := int64(len(firstMembers))
	firstPage, err := pagination.BuildPageSigned[member](params, firstMembers, params.Limit, firstTotal, nil, pageSecret)
	if err != nil {
		t.Fatalf("BuildPage (first page) returned error: %v", err)
	}

	if firstPage.Paging.NextCursor == "" {
		t.Fatalf("expected next cursor on first page")
	}

	secondRaw := url.Values{}
	secondRaw.Set("cursor", firstPage.Paging.NextCursor)

	secondParams, err := pagination.ParseWithSecurity(secondRaw, pageSecret, time.Hour)
	if err != nil {
		t.Fatalf("parse (second page) params: %v", err)
	}

	secondMembers := []member{{ID: "mem-3", CreatedAt: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)}}

	secondPage, err := pagination.BuildPageSigned[member](secondParams, secondMembers, secondParams.Limit, firstTotal, nil, pageSecret)
	if err != nil {
		t.Fatalf("BuildPage (second page) returned error: %v", err)
	}

	if secondPage.Paging.Total != firstTotal {
		t.Fatalf("expected total %d on second page, got %d", firstTotal, secondPage.Paging.Total)
	}

	if secondPage.Paging.HasMore {
		t.Fatalf("expected hasMore to be false on final page")
	}

	if secondPage.Paging.NextCursor != "" {
		t.Fatalf("expected empty next cursor on final page")
	}

	if secondPage.Paging.PrevCursor == "" {
		t.Fatalf("expected prev cursor to be generated on subsequent page")
	}

	payload, err := pagination.DecodeCursorSigned(secondPage.Paging.PrevCursor, pageSecret, time.Hour)
	if err != nil {
		t.Fatalf("decode prev cursor: %v", err)
	}

	if payload.Before["id"] != secondMembers[0].ID {
		t.Fatalf("expected prev cursor id %q, got %q", secondMembers[0].ID, payload.Before["id"])
	}
}

func TestBuildPage_NoResultsKeepsPrevCursor(t *testing.T) {
	params := pagination.Params{
		Limit: 2,
		Cursor: &pagination.CursorPayload{
			Sort: []pagination.SortExpression{{Field: "id", Direction: pagination.DirectionDesc}},
		},
		RawCursor: "dummy-cursor",
		Sort:      []pagination.SortExpression{{Field: "id", Direction: pagination.DirectionDesc}},
	}

	page, err := pagination.BuildPageSigned[member](params, nil, params.Limit, 0, nil, pageSecret)
	if err != nil {
		t.Fatalf("BuildPage returned error: %v", err)
	}

	if page.Paging.Total != 0 {
		t.Fatalf("expected total 0 when no results, got %d", page.Paging.Total)
	}

	if page.Paging.PrevCursor != params.RawCursor {
		t.Fatalf("expected prev cursor to reuse raw cursor when no results, got %q", page.Paging.PrevCursor)
	}
}

func TestBuildPage_AutomaticExtractorMissingField(t *testing.T) {
	params := pagination.Params{
		Limit: 1,
		Sort:  []pagination.SortExpression{{Field: "non_existent", Direction: pagination.DirectionAsc}},
	}

	members := []member{{ID: "mem-1", CreatedAt: time.Now().UTC()}, {ID: "mem-2", CreatedAt: time.Now().UTC()}}

	if _, err := pagination.BuildPageSigned[member](params, members, params.Limit, int64(len(members)), nil, pageSecret); err == nil {
		t.Fatalf("expected error when automatic extraction cannot resolve sort field")
	}
}

func TestBuildPage_AutomaticExtractorEmbeddedField(t *testing.T) {
	params := pagination.Params{
		Limit: 1,
		Sort:  []pagination.SortExpression{{Field: "id", Direction: pagination.DirectionAsc}},
	}

	orders := []order{{baseModel: baseModel{ID: "ord-1"}, CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}, {baseModel: baseModel{ID: "ord-2"}, CreatedAt: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)}}

	page, err := pagination.BuildPageSigned[order](params, orders, params.Limit, int64(len(orders)), nil, pageSecret)
	if err != nil {
		t.Fatalf("BuildPage returned error: %v", err)
	}

	if page.Paging.NextCursor == "" {
		t.Fatalf("expected next cursor to be generated using embedded id field")
	}

	payload, err := pagination.DecodeCursorSigned(page.Paging.NextCursor, pageSecret, time.Hour)
	if err != nil {
		t.Fatalf("decode next cursor: %v", err)
	}

	if got := payload.After["id"]; got != orders[0].ID {
		t.Fatalf("expected next cursor to include id %q, got %q", orders[0].ID, got)
	}
}

func TestBuildPage_BeforeCursorRestoresForwardPaging(t *testing.T) {
	params := pagination.Params{
		Limit: 3,
		Sort:  []pagination.SortExpression{{Field: "id", Direction: pagination.DirectionDesc}},
		Cursor: &pagination.CursorPayload{
			Before: map[string]string{"id": "mem-7"},
		},
	}

	// Simula resultados obtenidos por Apply al navegar hacia atrás:
	// ORDER asc + limit+1. BuildPage debe reordenarlos en el orden original (desc).
	results := []member{
		{ID: "mem-8", CreatedAt: time.Date(2024, 1, 1, 8, 0, 0, 0, time.UTC)},
		{ID: "mem-9", CreatedAt: time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)},
		{ID: "mem-10", CreatedAt: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)},
	}

	page, err := pagination.BuildPageSigned[member](params, results, params.Limit, 10, nil, pageSecret)
	if err != nil {
		t.Fatalf("BuildPage returned error: %v", err)
	}

	if len(page.Data) != 3 {
		t.Fatalf("expected 3 items, got %d", len(page.Data))
	}
	if page.Data[0].ID != "mem-10" || page.Data[2].ID != "mem-8" {
		t.Fatalf("expected results to be reversed back to desc order, got first=%s last=%s", page.Data[0].ID, page.Data[2].ID)
	}
	if !page.Paging.HasMore {
		t.Fatalf("expected hasMore true when returning from a previous cursor")
	}
	if page.Paging.NextCursor == "" {
		t.Fatalf("expected next cursor to navigate forward again")
	}
	if page.Paging.PrevCursor != "" {
		t.Fatalf("expected prev cursor to be empty at the first page boundary")
	}
}
