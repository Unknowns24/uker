package pagination

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
	"unicode"
)

// ErrNilCursorExtractor is returned when BuildPage requires cursor boundary values
// but no extractor function was provided.
var ErrNilCursorExtractor = errors.New("pagination: nil cursor extractor")

type cursorEncodeFunc func(CursorPayload) (string, error)

// CursorExtractor defines the function signature used by BuildPage to obtain the
// boundary values for the first and last records of a result set. The returned
// map should contain the fields referenced in Params.Sort so subsequent cursors
// can be generated.
type CursorExtractor[T any] func(item T) (map[string]string, error)

// BuildPage constructs a PagingResponse using the provided query parameters and
// result slice. The function expects the slice to contain up to limit+1
// elements, where the extra record is used to determine whether there is
// another page available. Callers must provide an extractor that converts a
// record into the cursor value map understood by BuildNextCursor/BuildPrevCursor.
func BuildPage[T any](params Params, results []T, limit int, extract CursorExtractor[T]) (PagingResponse[T], error) {
	encode := func(payload CursorPayload) (string, error) {
		return EncodeCursor(payload)
	}
	return buildPageWithEncoders(params, results, limit, extract, encode, encode)
}

// BuildPageSigned behaves like BuildPage but signs generated cursors with the provided secret.
func BuildPageSigned[T any](params Params, results []T, limit int, extract CursorExtractor[T], secret []byte) (PagingResponse[T], error) {
	if len(secret) == 0 {
		return PagingResponse[T]{}, errors.New("pagination: missing cursor signing secret")
	}

	encode := func(payload CursorPayload) (string, error) {
		return EncodeCursorSigned(payload, secret)
	}
	return buildPageWithEncoders(params, results, limit, extract, encode, encode)
}

func buildPageWithEncoders[T any](params Params, results []T, limit int, extract CursorExtractor[T], encodeNext, encodePrev cursorEncodeFunc) (PagingResponse[T], error) {
	if limit < 0 {
		limit = 0
	}

	items := results
	hasMore := false
	if len(results) > limit {
		hasMore = true
		if limit < len(results) {
			items = results[:limit]
		}
	}

	needsExtractor := (hasMore && limit > 0) || (params.Cursor != nil && len(items) > 0)
	var err error
	if extract == nil && needsExtractor {
		extract, err = newAutoCursorExtractor[T](params, items)
		if err != nil {
			return PagingResponse[T]{}, err
		}
	}

	if encodeNext == nil || encodePrev == nil {
		return PagingResponse[T]{}, errors.New("pagination: nil cursor encoder")
	}

	var nextCursor string
	if hasMore && limit > 0 {
		if extract == nil {
			return PagingResponse[T]{}, ErrNilCursorExtractor
		}

		cursorValues, err := extract(items[len(items)-1])
		if err != nil {
			return PagingResponse[T]{}, err
		}

		payload, err := buildNextCursorPayload(params, cursorValues)
		if err != nil {
			return PagingResponse[T]{}, err
		}
		if payload != nil {
			nextCursor, err = encodeNext(*payload)
			if err != nil {
				return PagingResponse[T]{}, err
			}
		}
	}

	var prevCursor string
	if params.Cursor != nil {
		if len(items) == 0 {
			prevCursor = params.RawCursor
		} else {
			if extract == nil {
				return PagingResponse[T]{}, ErrNilCursorExtractor
			}

			cursorValues, err := extract(items[0])
			if err != nil {
				return PagingResponse[T]{}, err
			}

			payload, err := buildPrevCursorPayload(params, cursorValues)
			if err != nil {
				return PagingResponse[T]{}, err
			}
			if payload != nil {
				prevCursor, err = encodePrev(*payload)
				if err != nil {
					return PagingResponse[T]{}, err
				}
			}
		}
	}

	return NewPage(items, limit, hasMore, nextCursor, prevCursor), nil
}

func newAutoCursorExtractor[T any](params Params, items []T) (CursorExtractor[T], error) {
	if len(params.Sort) == 0 {
		return func(T) (map[string]string, error) {
			return nil, nil
		}, nil
	}

	structType, err := inferStructType[T](items)
	if err != nil {
		return nil, err
	}

	accessors, err := buildFieldAccessors(structType, params.Sort)
	if err != nil {
		return nil, err
	}

	return func(item T) (map[string]string, error) {
		value := reflect.ValueOf(item)
		if !value.IsValid() {
			return nil, errors.New("pagination: cannot extract cursor values from invalid item")
		}

		// Handle pointers to the underlying struct.
		if value.Kind() == reflect.Pointer {
			if value.IsNil() {
				return nil, errors.New("pagination: cannot extract cursor values from nil pointer item")
			}
			value = value.Elem()
		}

		if value.Kind() != reflect.Struct {
			return nil, fmt.Errorf("pagination: automatic cursor extraction expects struct items, got %s", value.Kind())
		}

		cursor := make(map[string]string, len(accessors))
		for _, accessor := range accessors {
			field := value.Field(accessor.index)
			encoded, err := formatCursorValue(field)
			if err != nil {
				return nil, fmt.Errorf("pagination: %s", err)
			}
			cursor[accessor.sortField] = encoded
		}

		return cursor, nil
	}, nil
}

func inferStructType[T any](items []T) (reflect.Type, error) {
	for _, item := range items {
		value := reflect.ValueOf(item)
		if !value.IsValid() {
			continue
		}
		typ := value.Type()
		if typ.Kind() == reflect.Pointer {
			typ = typ.Elem()
		}
		if typ.Kind() == reflect.Struct {
			return typ, nil
		}
	}

	var zero T
	typ := reflect.TypeOf(zero)
	if typ == nil {
		return nil, errors.New("pagination: cannot infer result type for automatic cursor extraction")
	}
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("pagination: automatic cursor extraction requires struct results, got %s", typ.Kind())
	}
	return typ, nil
}

type fieldAccessor struct {
	sortField string
	index     int
}

func buildFieldAccessors(structType reflect.Type, sorts []SortExpression) ([]fieldAccessor, error) {
	lookup := map[string]int{}
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if !field.IsExported() {
			continue
		}

		for _, alias := range fieldAliases(field) {
			key := strings.ToLower(alias)
			if _, exists := lookup[key]; !exists {
				lookup[key] = i
			}
		}
	}

	accessors := make([]fieldAccessor, 0, len(sorts))
	for _, sort := range sorts {
		key := strings.ToLower(sort.Field)
		index, ok := lookup[key]
		if !ok {
			return nil, fmt.Errorf("pagination: cannot find field %q in %s for automatic cursor extraction", sort.Field, structType.Name())
		}
		accessors = append(accessors, fieldAccessor{sortField: sort.Field, index: index})
	}

	return accessors, nil
}

func fieldAliases(field reflect.StructField) []string {
	aliases := []string{field.Name, toSnake(field.Name)}

	if jsonTag := field.Tag.Get("json"); jsonTag != "" {
		name := strings.Split(jsonTag, ",")[0]
		if name != "" && name != "-" {
			aliases = append(aliases, name)
		}
	}

	if dbTag := field.Tag.Get("db"); dbTag != "" {
		aliases = append(aliases, strings.Split(dbTag, ",")[0])
	}

	if gormTag := field.Tag.Get("gorm"); gormTag != "" {
		for _, part := range strings.Split(gormTag, ";") {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "column:") {
				aliases = append(aliases, strings.TrimPrefix(part, "column:"))
			}
		}
	}

	return aliases
}

func formatCursorValue(value reflect.Value) (string, error) {
	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return "", nil
		}
		value = value.Elem()
	}

	if !value.IsValid() {
		return "", nil
	}

	if value.Type() == reflect.TypeOf(time.Time{}) {
		if !value.CanInterface() {
			return "", errors.New("time field is not accessible")
		}
		t := value.Interface().(time.Time)
		return t.UTC().Format(time.RFC3339), nil
	}

	if value.CanInterface() {
		if stringer, ok := value.Interface().(fmt.Stringer); ok {
			return stringer.String(), nil
		}
	}

	switch value.Kind() {
	case reflect.String:
		return value.String(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Bool:
		if value.CanInterface() {
			return fmt.Sprintf("%v", value.Interface()), nil
		}
	}

	if value.CanInterface() {
		return fmt.Sprintf("%v", value.Interface()), nil
	}

	return "", errors.New("unsupported field type for cursor encoding")
}

func toSnake(raw string) string {
	if raw == "" {
		return ""
	}

	var builder strings.Builder
	builder.Grow(len(raw) + len(raw)/2)
	for i, runeValue := range raw {
		if unicode.IsUpper(runeValue) {
			if i > 0 {
				builder.WriteByte('_')
			}
			builder.WriteRune(unicode.ToLower(runeValue))
		} else {
			builder.WriteRune(runeValue)
		}
	}
	return builder.String()
}
