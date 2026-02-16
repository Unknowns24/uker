package businessrepo

import (
	"net/url"
	"testing"
	"time"

	"github.com/unknowns24/uker/uker/pagination"
)

var testSecret = []byte("test-secret")

func TestBuildPage_FirstPageWithoutCursor(t *testing.T) {
	raw := url.Values{}
	raw.Set("limit", "2")
	raw.Set("sort", "created_at:asc")

	params, err := pagination.ParseWithSecurity(raw, testSecret, time.Hour)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	members := []BusinessMember{
		{ID: "mem-1", CreatedAt: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)},
		{ID: "mem-2", CreatedAt: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)},
		{ID: "mem-3", CreatedAt: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)},
	}

	page, err := pagination.BuildPageSigned[BusinessMember](params, members, params.Limit, int64(len(members)), nil, testSecret)
	if err != nil {
		t.Fatalf("BuildPage returned error: %v", err)
	}

	if len(page.Data) != 2 {
		t.Fatalf("expected 2 items, got %d", len(page.Data))
	}

	if page.Paging.Total != int64(len(members)) {
		t.Fatalf("expected total %d, got %d", len(members), page.Paging.Total)
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

	params, err := pagination.ParseWithSecurity(raw, testSecret, time.Hour)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	firstMembers := []BusinessMember{
		{ID: "mem-1", CreatedAt: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)},
		{ID: "mem-2", CreatedAt: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)},
		{ID: "mem-3", CreatedAt: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)},
	}

	firstPage, err := pagination.BuildPageSigned[BusinessMember](params, firstMembers, params.Limit, int64(len(firstMembers)), nil, testSecret)
	if err != nil {
		t.Fatalf("BuildPage (first page) returned error: %v", err)
	}

	if firstPage.Paging.NextCursor == "" {
		t.Fatalf("expected next cursor on first page")
	}

	secondRaw := url.Values{}
	secondRaw.Set("cursor", firstPage.Paging.NextCursor)

	secondParams, err := pagination.ParseWithSecurity(secondRaw, testSecret, time.Hour)
	if err != nil {
		t.Fatalf("Parse (second page) returned error: %v", err)
	}

	secondMembers := []BusinessMember{{ID: "mem-3", CreatedAt: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)}}

	secondPage, err := pagination.BuildPageSigned[BusinessMember](secondParams, secondMembers, secondParams.Limit, int64(len(firstMembers)), nil, testSecret)
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

	if secondPage.Paging.Total != int64(len(firstMembers)) {
		t.Fatalf("expected total %d on second page, got %d", len(firstMembers), secondPage.Paging.Total)
	}

	payload, err := pagination.DecodeCursorSigned(secondPage.Paging.PrevCursor, testSecret, time.Hour)
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

	page, err := pagination.BuildPageSigned[BusinessMember](params, nil, params.Limit, 0, nil, testSecret)
	if err != nil {
		t.Fatalf("BuildPage returned error: %v", err)
	}

	if page.Paging.PrevCursor != params.RawCursor {
		t.Fatalf("expected prev cursor to reuse raw cursor when no results, got %q", page.Paging.PrevCursor)
	}

	if page.Paging.Total != 0 {
		t.Fatalf("expected total 0 when no results, got %d", page.Paging.Total)
	}
}

func TestBuildPage_RequiresExtractorWhenNeeded(t *testing.T) {
	params := pagination.Params{
		Limit: 1,
		Sort:  []pagination.SortExpression{{Field: "non_existent", Direction: pagination.DirectionAsc}},
	}

	members := []BusinessMember{{ID: "mem-1", CreatedAt: time.Now().UTC()}, {ID: "mem-2", CreatedAt: time.Now().UTC()}}

	if _, err := pagination.BuildPageSigned[BusinessMember](params, members, params.Limit, int64(len(members)), nil, testSecret); err == nil {
		t.Fatalf("expected error when automatic extraction cannot find field")
	}
}
