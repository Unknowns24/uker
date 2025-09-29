package pagination

// PagingResponse describes the response envelope expected by clients consuming cursor
// pagination. The structure mirrors the documentation example.
type PagingResponse[T any] struct {
	Data   []T         `json:"data"`
	Paging PagingBlock `json:"paging"`
}

// PagingBlock embeds metadata about the delivered page.
type PagingBlock struct {
	Limit      int    `json:"limit"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor,omitempty"`
	PrevCursor string `json:"prev_cursor,omitempty"`
}

// NewPage creates a paging response with the provided data slice and metadata. The function is
// generic to avoid unnecessary allocations when building API responses.
func NewPage[T any](data []T, limit int, hasMore bool, nextCursor, prevCursor string) PagingResponse[T] {
	// Defensive copy so callers can continue re-using their input slice without exposing
	// future mutations.
	copied := make([]T, len(data))
	copy(copied, data)

	return PagingResponse[T]{
		Data: copied,
		Paging: PagingBlock{
			Limit:      limit,
			HasMore:    hasMore,
			NextCursor: nextCursor,
			PrevCursor: prevCursor,
		},
	}
}
