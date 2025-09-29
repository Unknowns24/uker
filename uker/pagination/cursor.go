package pagination

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/unknowns24/uker/internal/base64url"
)

// Direction represents the ordering applied to a sort field.
type Direction string

const (
	// DirectionAsc sorts values in ascending order.
	DirectionAsc Direction = "asc"
	// DirectionDesc sorts values in descending order.
	DirectionDesc Direction = "desc"
)

// SortExpression defines a field and the direction used for cursor pagination.
type SortExpression struct {
	Field     string
	Direction Direction
}

// MarshalJSON encodes the sort expression as the array representation described in the
// pagination documentation, e.g. ["created_at","desc"].
func (s SortExpression) MarshalJSON() ([]byte, error) {
	if s.Field == "" {
		return nil, errors.New("pagination: missing sort field")
	}
	direction := s.Direction
	if direction == "" {
		direction = DirectionDesc
	}
	return json.Marshal([2]string{s.Field, string(direction)})
}

// UnmarshalJSON decodes the array representation into a SortExpression.
func (s *SortExpression) UnmarshalJSON(data []byte) error {
	var raw []string
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("pagination: invalid sort encoding: %w", err)
	}

	if len(raw) == 0 || raw[0] == "" {
		return errors.New("pagination: invalid sort encoding")
	}

	direction := DirectionDesc
	if len(raw) > 1 && raw[1] != "" {
		dir := Direction(raw[1])
		if dir != DirectionAsc && dir != DirectionDesc {
			return errors.New("pagination: invalid sort direction")
		}
		direction = dir
	}

	s.Field = raw[0]
	s.Direction = direction
	return nil
}

// CursorPayload matches the documented cursor schema. It remains transport agnostic and can
// be encoded using EncodeCursor / DecodeCursor.
type CursorPayload struct {
	Version   int               `json:"v"`
	Sort      []SortExpression  `json:"sort,omitempty"`
	Filters   map[string]string `json:"filters,omitempty"`
	After     map[string]string `json:"after,omitempty"`
	Before    map[string]string `json:"before,omitempty"`
	Timestamp int64             `json:"ts,omitempty"`
	Signature string            `json:"sig,omitempty"`
}

// BuildNextCursor constructs a cursor that points to the next page using the provided
// pagination parameters and keyset boundary values. The supplied values map must contain the
// fields referenced by the sort expressions so the subsequent request can continue from the
// last record of the current page.
func BuildNextCursor(params Params, values map[string]string) (string, error) {
	if len(values) == 0 {
		return "", nil
	}

	payload := CursorPayload{
		Sort:    cloneSortExpressions(params.Sort),
		Filters: cloneFilters(params.Filters),
		After:   cloneCursorValues(values),
	}

	return EncodeCursor(payload)
}

// BuildPrevCursor constructs a cursor pointing to the previous page using the provided
// pagination parameters and keyset boundary values. Callers are expected to pass the first
// record of the current page so the API can navigate backwards when the client requests it.
func BuildPrevCursor(params Params, values map[string]string) (string, error) {
	if len(values) == 0 {
		return "", nil
	}

	payload := CursorPayload{
		Sort:    cloneSortExpressions(params.Sort),
		Filters: cloneFilters(params.Filters),
		Before:  cloneCursorValues(values),
	}

	return EncodeCursor(payload)
}

func cloneSortExpressions(src []SortExpression) []SortExpression {
	if len(src) == 0 {
		return nil
	}
	cloned := make([]SortExpression, len(src))
	copy(cloned, src)
	return cloned
}

func cloneFilters(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(src))
	for key, value := range src {
		cloned[key] = value
	}
	return cloned
}

func cloneCursorValues(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(src))
	for key, value := range src {
		cloned[key] = value
	}
	return cloned
}

// EncodeCursor serialises the provided payload following the documented format and returns a
// base64url transport string.
func EncodeCursor(payload CursorPayload) (string, error) {
	if payload.Version == 0 {
		payload.Version = 1
	}
	if payload.Timestamp == 0 {
		payload.Timestamp = time.Now().Unix()
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("pagination: cannot marshal cursor: %w", err)
	}

	return base64url.Encode(string(raw)), nil
}

// DecodeCursor reads the provided base64url string and decodes it into a CursorPayload.
func DecodeCursor(encoded string) (CursorPayload, error) {
	if encoded == "" {
		return CursorPayload{}, errors.New("pagination: empty cursor")
	}

	decoded, err := base64url.Decode(encoded)
	if err != nil {
		return CursorPayload{}, fmt.Errorf("pagination: malformed cursor: %w", err)
	}

	var payload CursorPayload
	if err := json.Unmarshal([]byte(decoded), &payload); err != nil {
		return CursorPayload{}, fmt.Errorf("pagination: invalid cursor payload: %w", err)
	}

	if payload.Version <= 0 {
		return CursorPayload{}, errors.New("pagination: unsupported cursor version")
	}

	for _, sort := range payload.Sort {
		if sort.Field == "" {
			return CursorPayload{}, errors.New("pagination: cursor sort field required")
		}
		if sort.Direction != DirectionAsc && sort.Direction != DirectionDesc {
			return CursorPayload{}, errors.New("pagination: cursor sort direction invalid")
		}
	}

	return payload, nil
}
