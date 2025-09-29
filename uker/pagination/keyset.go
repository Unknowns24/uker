package pagination

import (
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultLimit is applied when the request does not provide an explicit limit.
	DefaultLimit = 25
	// MaxLimit guards the API from overly expensive page sizes.
	MaxLimit = 100
)

var (
	// ErrInvalidCursor is returned when the provided cursor cannot be decoded or validated.
	ErrInvalidCursor = errors.New("pagination: invalid cursor")
	// ErrCursorExpired indicates that the cursor signature is valid but its TTL elapsed.
	ErrCursorExpired = errors.New("pagination: cursor expired")
	// ErrInvalidSort indicates that the sort query parameter is malformed or unsupported.
	ErrInvalidSort = errors.New("pagination: invalid sort")
	// ErrInvalidFilter reports that one of the query filters does not follow the supported format.
	ErrInvalidFilter = errors.New("pagination: invalid filter")
	// ErrLimitOutOfRange signals that the requested limit is outside the allowed bounds.
	ErrLimitOutOfRange = errors.New("pagination: limit out of range")
	// ErrInvalidIdentifier reports a disallowed or malformed column identifier.
	ErrInvalidIdentifier = errors.New("pagination: invalid identifier")
)

var allowedFilterOperators = map[string]struct{}{
	"eq":   {},
	"neq":  {},
	"lt":   {},
	"lte":  {},
	"gt":   {},
	"gte":  {},
	"like": {},
	"in":   {},
	"nin":  {},
}

// Params encapsulates the parsed pagination request.
type Params struct {
	Limit     int
	Sort      []SortExpression
	Filters   map[string]string
	Cursor    *CursorPayload
	RawCursor string
}

type cursorDecoder func(string) (CursorPayload, error)

// Parse reads the provided query values and generates pagination parameters following the
// documented cursor pagination contract.
func Parse(values url.Values) (Params, error) {
	return parse(values, DecodeCursor)
}

// ParseWithSecurity mirrors Parse but verifies signed cursors using the supplied secret and TTL.
func ParseWithSecurity(values url.Values, secret []byte, ttl time.Duration) (Params, error) {
	if len(secret) == 0 {
		return Params{}, ErrInvalidCursor
	}

	decoder := func(raw string) (CursorPayload, error) {
		return DecodeCursorSigned(raw, secret, ttl)
	}

	return parse(values, decoder)
}

func parse(values url.Values, decoder cursorDecoder) (Params, error) {
	params := Params{
		Limit:   DefaultLimit,
		Filters: map[string]string{},
	}

	if limit := values.Get("limit"); limit != "" {
		parsed, err := strconv.Atoi(limit)
		if err != nil || parsed <= 0 {
			return Params{}, ErrLimitOutOfRange
		}
		if parsed > MaxLimit {
			return Params{}, ErrLimitOutOfRange
		}
		params.Limit = parsed
	}

	if cursor := values.Get("cursor"); cursor != "" {
		if decoder == nil {
			return Params{}, ErrInvalidCursor
		}
		payload, err := decoder(cursor)
		if err != nil {
			if errors.Is(err, ErrCursorExpired) {
				return Params{}, err
			}
			return Params{}, ErrInvalidCursor
		}
		params.Cursor = &payload
		params.RawCursor = cursor

		if len(payload.Filters) > 0 {
			for key, value := range payload.Filters {
				params.Filters[key] = value
			}
		}

		if len(payload.Sort) > 0 {
			params.Sort = append(params.Sort, payload.Sort...)
		}
	}

	sortParam := values.Get("sort")
	if sortParam != "" && params.Cursor != nil && len(params.Cursor.Sort) > 0 {
		// Query attempts to override cursor sort; this is not allowed because cursors
		// must remain consistent between requests. We surface a consistent error.
		return Params{}, ErrInvalidSort
	}

	if sortParam != "" && (params.Cursor == nil || len(params.Cursor.Sort) == 0) {
		sorts, err := parseSort(sortParam)
		if err != nil {
			return Params{}, err
		}
		params.Sort = sorts
	}

	if len(params.Sort) == 0 {
		params.Sort = []SortExpression{{Field: "id", Direction: DirectionDesc}}
	}
	ensureIDSort(&params.Sort)

	filters, err := parseFilters(values)
	if err != nil {
		return Params{}, err
	}
	if params.Cursor != nil && len(filters) > 0 {
		return Params{}, ErrInvalidFilter
	}

	for key, value := range filters {
		params.Filters[key] = value
	}

	return params, nil
}

func parseSort(raw string) ([]SortExpression, error) {
	if raw == "" {
		return nil, nil
	}

	segments := strings.Split(raw, ",")
	expressions := make([]SortExpression, 0, len(segments))
	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		parts := strings.Split(segment, ":")
		field := strings.TrimSpace(parts[0])
		identifier, err := requireIdent(field, ErrInvalidSort)
		if err != nil {
			return nil, err
		}

		direction := DirectionDesc
		if len(parts) > 1 {
			dir := Direction(strings.ToLower(strings.TrimSpace(parts[1])))
			switch dir {
			case DirectionAsc, DirectionDesc:
				direction = dir
			default:
				return nil, ErrInvalidSort
			}
		}

		expressions = append(expressions, SortExpression{Field: identifier, Direction: direction})
	}

	if len(expressions) == 0 {
		return nil, ErrInvalidSort
	}

	return expressions, nil
}

func parseFilters(values url.Values) (map[string]string, error) {
	filters := map[string]string{}
	for key, rawValues := range values {
		if key == "limit" || key == "cursor" || key == "sort" {
			continue
		}
		if len(rawValues) == 0 {
			continue
		}

		field, operator, ok := strings.Cut(key, "_")
		if !ok || field == "" || operator == "" {
			return nil, ErrInvalidFilter
		}
		if _, allowed := allowedFilterOperators[operator]; !allowed {
			return nil, ErrInvalidFilter
		}

		identifier, err := requireIdent(strings.TrimSpace(field), ErrInvalidFilter)
		if err != nil {
			return nil, err
		}

		combined := strings.Join(rawValues, ",")
		filters[identifier+"_"+operator] = combined
	}

	return filters, nil
}

func ensureIDSort(sortExpressions *[]SortExpression) {
	sortSlice := *sortExpressions
	hasID := false
	for _, entry := range sortSlice {
		if strings.EqualFold(entry.Field, "id") {
			hasID = true
			break
		}
	}
	if !hasID {
		sortSlice = append(sortSlice, SortExpression{Field: "id", Direction: DirectionDesc})
	}

	// Normalise to guarantee deterministic order for callers and to avoid duplicated id
	// expressions (e.g. when provided by the cursor).
	unique := make([]SortExpression, 0, len(sortSlice))
	seen := map[string]struct{}{}
	for _, entry := range sortSlice {
		key := strings.ToLower(entry.Field)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, entry)
	}

	*sortExpressions = unique
	sort.SliceStable(*sortExpressions, func(i, j int) bool {
		if strings.EqualFold((*sortExpressions)[i].Field, "id") {
			return false
		}
		if strings.EqualFold((*sortExpressions)[j].Field, "id") {
			return true
		}
		return i < j
	})
}
