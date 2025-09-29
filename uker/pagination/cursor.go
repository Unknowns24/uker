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
