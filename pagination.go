package uker

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

type Pagination struct {
	Page    string
	Sort    string
	Search  string
	PerPage string
	SortDir string
}

type PaginationOpts struct {
	DB         *gorm.DB
	Join       string
	Where      string
	Select     string
	Result     interface{}
	TableModel interface{}
}

// Server data pagination with specified select
//
// @param db *gorm.DB: Database pointer to perform the pagination.
//
// @param tableModel interface{}: Model of the table to paginate.
//
// @param selectQry string: Select query.
//
// @param condition string: Where condition to add to the pagination if necessary.
//
// @param join string: join condition to add to the pagination if necessary.
//
// @param result interface{}: Interface of wantend result.
//
// @return map[string]interface{}: map with all paginated data
func (p *Pagination) Paginate(opts PaginationOpts) map[string]interface{} {
	// Build a base query without conditions
	query := opts.DB.Model(opts.TableModel)

	// Apply custom select query if provided
	if opts.Select != "" {
		query = query.Select(opts.Select)
	}

	if opts.Where != "" {
		query = query.Where(opts.Where)
	}

	// Apply search if provided
	if p.Search != "" {
		// Get the type of the result to dynamically generate search conditions
		modelType := reflect.TypeOf(opts.Result).Elem().Elem()

		// Start with an empty condition
		searchCondition := ""

		// Iterate over the fields of the model
		for i := 0; i < modelType.NumField(); i++ {
			fieldName := modelType.Field(i).Name

			// Ignore the "id" field
			if strings.ToLower(fieldName) == "id" {
				continue
			}

			// Add a condition for the current field
			if searchCondition == "" {
				searchCondition = fieldName + " LIKE " + "'%%" + p.Search + "%%'"
			} else {
				searchCondition += " OR " + fieldName + " LIKE " + "'%%" + p.Search + "%%'"
			}
		}

		// Combine the search condition with the existing condition using "AND"
		query = query.Where(searchCondition)
	}

	// Apply join if provided
	if opts.Join != "" {
		query = query.Joins(opts.Join)
	}

	// Apply sorting if provided
	if p.Sort != "" {
		if p.SortDir == PAGINATION_ORDER_DESC {
			query = query.Order(fmt.Sprintf("%s %s", p.Sort, strings.ToUpper(PAGINATION_ORDER_DESC)))
		} else {
			query = query.Order(p.Sort)
		}
	}

	// Convert URL parameters to integers
	page, err := strconv.Atoi(p.Page)
	if err != nil {
		page = 1
	}

	perPage, err := strconv.Atoi(p.PerPage)
	if err != nil {
		perPage = 10
	}

	// Perform the query and count the total records
	var total int64
	query.Count(&total)

	// Calculate the number of pages and adjust the requested page if necessary
	lastPage := int(math.Ceil(float64(total) / float64(perPage)))
	if page > lastPage {
		page = lastPage
	}

	// Perform pagination
	query.Limit(perPage).Offset((page - 1) * perPage).Find(opts.Result)

	return map[string]interface{}{
		"page":      page,
		"total":     total,
		"per_page":  perPage,
		"last_page": lastPage,
		"data":      opts.Result,
	}
}
