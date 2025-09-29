package businessrepo

import (
	"net/url"
	"testing"
	"time"

	"github.com/unknowns24/uker/uker/pagination"
)

func TestBuildPage_FirstPageWithoutCursor(t *testing.T) {
	raw := url.Values{}
	raw.Set("limit", "2")
	raw.Set("sort", "created_at:asc")

	params, err := pagination.Parse(raw)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	members := []BusinessMember{
		{ID: "mem-1", CreatedAt: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)},
		{ID: "mem-2", CreatedAt: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)},
		{ID: "mem-3", CreatedAt: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)},
	}

	page, err := pagination.BuildPage[BusinessMember](params, members, params.Limit, nil)
	if err != nil {
		t.Fatalf("BuildPage returned error: %v", err)
	}

	if len(page.Data) != 2 {
		t.Fatalf("expected 2 items, got %d", len(page.Data))
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

	params, err := pagination.Parse(raw)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	firstMembers := []BusinessMember{
		{ID: "mem-1", CreatedAt: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)},
		{ID: "mem-2", CreatedAt: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)},
		{ID: "mem-3", CreatedAt: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)},
	}

	firstPage, err := pagination.BuildPage[BusinessMember](params, firstMembers, params.Limit, nil)
	if err != nil {
		t.Fatalf("BuildPage (first page) returned error: %v", err)
	}

	if firstPage.Paging.NextCursor == "" {
		t.Fatalf("expected next cursor on first page")
	}

	secondRaw := url.Values{}
	secondRaw.Set("cursor", firstPage.Paging.NextCursor)
	secondRaw.Set("limit", "2")

	secondParams, err := pagination.Parse(secondRaw)
	if err != nil {
		t.Fatalf("Parse (second page) returned error: %v", err)
	}

	secondMembers := []BusinessMember{{ID: "mem-3", CreatedAt: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)}}

	secondPage, err := pagination.BuildPage[BusinessMember](secondParams, secondMembers, secondParams.Limit, nil)
	if err != nil {
		t.Fatalf("BuildPage (second page) returned error: %v", err)
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

	payload, err := pagination.DecodeCursor(secondPage.Paging.PrevCursor)
	if err != nil {
		t.Fatalf("failed to decode prev cursor: %v", err)
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

	page, err := pagination.BuildPage[BusinessMember](params, nil, params.Limit, nil)
	if err != nil {
		t.Fatalf("BuildPage returned error: %v", err)
	}

	if page.Paging.PrevCursor != params.RawCursor {
		t.Fatalf("expected prev cursor to reuse raw cursor when no results, got %q", page.Paging.PrevCursor)
	}
}

func TestBuildPage_RequiresExtractorWhenNeeded(t *testing.T) {
	params := pagination.Params{
		Limit: 1,
		Sort:  []pagination.SortExpression{{Field: "non_existent", Direction: pagination.DirectionAsc}},
	}

	members := []BusinessMember{{ID: "mem-1", CreatedAt: time.Now().UTC()}, {ID: "mem-2", CreatedAt: time.Now().UTC()}}

	if _, err := pagination.BuildPage[BusinessMember](params, members, params.Limit, nil); err == nil {
		t.Fatalf("expected error when automatic extraction cannot find field")
	}
}
