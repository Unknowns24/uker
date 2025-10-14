package pagination

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var errNilDB = errors.New("pagination: nil db")

// Apply attaches the provided pagination parameters to the GORM query. It configures
// limit, sorting, filters and cursor bounds so repositories can reuse the behaviour
// consistently across entities.
func Apply(db *gorm.DB, params Params) (*gorm.DB, error) {
	if db == nil {
		return nil, errNilDB
	}

	query := db
	if params.Limit > 0 {
		query = query.Limit(params.Limit + 1)
	}

	// Apply filters first so keyset conditions can rely on consistent aliases.
	for key, raw := range params.Filters {
		idx := strings.LastIndex(key, "_")
		if idx <= 0 || idx == len(key)-1 {
			return nil, ErrInvalidFilter
		}

		field := key[:idx]
		operator := key[idx+1:]
		expr, values, err := buildFilterExpression(field, operator, raw)
		if err != nil {
			return nil, err
		}
		query = query.Where(expr, values...)
	}

	if params.Cursor != nil {
		cursor := params.Cursor
		if len(cursor.After) > 0 && len(cursor.Before) > 0 {
			return nil, ErrInvalidCursor
		}

		if len(cursor.After) > 0 {
			if expr, values, ok, err := buildKeysetPredicateTuple(params.Sort, cursor.After, false); err != nil {
				return nil, err
			} else if ok {
				query = query.Where(expr, values...)
			} else {
				expr, values, err := buildKeysetPredicate(params.Sort, cursor.After, false)
				if err != nil {
					return nil, err
				}
				query = query.Where(expr, values...)
			}
		}

		if len(cursor.Before) > 0 {
			if expr, values, ok, err := buildKeysetPredicateTuple(params.Sort, cursor.Before, true); err != nil {
				return nil, err
			} else if ok {
				query = query.Where(expr, values...)
			} else {
				expr, values, err := buildKeysetPredicate(params.Sort, cursor.Before, true)
				if err != nil {
					return nil, err
				}
				query = query.Where(expr, values...)
			}
		}
	}

	for _, sort := range params.Sort {
		column, err := requireIdent(sort.Field, ErrInvalidSort)
		if err != nil {
			return nil, err
		}
		query = query.Order(clause.OrderByColumn{Column: clause.Column{Name: column}, Desc: sort.Direction == DirectionDesc})
	}

	return query, nil
}

func buildFilterExpression(field, operator, raw string) (string, []any, error) {
	column, err := requireIdent(strings.TrimSpace(field), ErrInvalidFilter)
	if err != nil {
		return "", nil, err
	}

	switch operator {
	case "eq":
		return fmt.Sprintf("%s = ?", column), []any{raw}, nil
	case "neq":
		return fmt.Sprintf("%s <> ?", column), []any{raw}, nil
	case "lt":
		return fmt.Sprintf("%s < ?", column), []any{raw}, nil
	case "lte":
		return fmt.Sprintf("%s <= ?", column), []any{raw}, nil
	case "gt":
		return fmt.Sprintf("%s > ?", column), []any{raw}, nil
	case "gte":
		return fmt.Sprintf("%s >= ?", column), []any{raw}, nil
	case "like":
		return fmt.Sprintf("%s LIKE ?", column), []any{"%" + raw + "%"}, nil
	case "in", "nin":
		values := splitCSV(raw)
		if len(values) == 0 {
			return "", nil, ErrInvalidFilter
		}
		keyword := "IN"
		if operator == "nin" {
			keyword = "NOT IN"
		}
		return fmt.Sprintf("%s %s ?", column, keyword), []any{values}, nil
	default:
		return "", nil, ErrInvalidFilter
	}
}

func splitCSV(raw string) []string {
	segments := strings.Split(raw, ",")
	cleaned := make([]string, 0, len(segments))
	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		cleaned = append(cleaned, segment)
	}
	return cleaned
}

func buildKeysetPredicateTuple(sortExpressions []SortExpression, cursorValues map[string]string, invert bool) (string, []any, bool, error) {
	if len(sortExpressions) != 2 {
		return "", nil, false, nil
	}

	first := sortExpressions[0]
	second := sortExpressions[1]
	if first.Direction != second.Direction {
		return "", nil, false, nil
	}

	firstColumn, err := requireIdent(first.Field, ErrInvalidSort)
	if err != nil {
		return "", nil, false, err
	}
	secondColumn, err := requireIdent(second.Field, ErrInvalidSort)
	if err != nil {
		return "", nil, false, err
	}

	firstValue, okFirst := cursorValues[first.Field]
	secondValue, okSecond := cursorValues[second.Field]
	if !okFirst || !okSecond {
		return "", nil, false, ErrInvalidCursor
	}

	comparator := comparatorFor(first.Direction, invert)
	expr := fmt.Sprintf("(%s, %s) %s (?, ?)", firstColumn, secondColumn, comparator)
	return expr, []any{firstValue, secondValue}, true, nil
}

func buildKeysetPredicate(sortExpressions []SortExpression, cursorValues map[string]string, invert bool) (string, []any, error) {
	if len(sortExpressions) == 0 {
		return "", nil, ErrInvalidCursor
	}

	clauses := make([]string, 0, len(sortExpressions))
	args := make([]any, 0, len(cursorValues)*len(sortExpressions))

	for i := range sortExpressions {
		parts := make([]string, 0, i+1)
		for j := 0; j <= i; j++ {
			sortExpr := sortExpressions[j]
			column, err := requireIdent(sortExpr.Field, ErrInvalidSort)
			if err != nil {
				return "", nil, err
			}

			value, ok := cursorValues[sortExpr.Field]
			if !ok {
				return "", nil, ErrInvalidCursor
			}

			if j == i {
				comparator := comparatorFor(sortExpr.Direction, invert)
				parts = append(parts, fmt.Sprintf("%s %s ?", column, comparator))
				args = append(args, value)
			} else {
				parts = append(parts, fmt.Sprintf("%s = ?", column))
				args = append(args, cursorValues[sortExpr.Field])
			}
		}

		clauses = append(clauses, "("+strings.Join(parts, " AND ")+")")
	}

	return strings.Join(clauses, " OR "), args, nil
}

func comparatorFor(direction Direction, invert bool) string {
	switch direction {
	case DirectionAsc:
		if invert {
			return "<"
		}
		return ">"
	case DirectionDesc:
		if invert {
			return ">"
		}
		return "<"
	default:
		if invert {
			return "<"
		}
		return ">"
	}
}
