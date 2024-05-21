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
	Page       string
	Sort       string
	Search     string
	PerPage    string
	SortDir    string
	WhereField string
	WhereValue string
}

type PaginationOpts struct {
	DB           *gorm.DB
	Join         string
	Where        string
	Select       string
	SortPfx      string
	SearchPfx    string
	DynamicWhere bool
	Result       interface{}
	TableModel   interface{}
}

type PaginationResult struct {
	Info    PaginationResultInfo `json:"info"`
	Results interface{}          `json:"results"`
}

type PaginationResultInfo struct {
	Page     int `json:"page"`
	Total    int `json:"total"`
	PerPage  int `json:"per_page"`
	LastPage int `json:"last_page"`
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
// @return PaginationResult: structure with pagination data
func (p *Pagination) Paginate(opts PaginationOpts) PaginationResult {
	// Build a base query without conditions
	query := opts.DB.Model(opts.TableModel)

	// Apply custom select query if provided
	if opts.Select != "" {
		query = query.Select(opts.Select)
	}

	if opts.Where != "" {
		query = query.Where(opts.Where)
	}

	if opts.DynamicWhere && p.WhereField != "" && p.WhereValue != "" {
		query = query.Where(fmt.Sprintf("%s = ?", p.WhereField), p.WhereValue)
	}

	// Apply search if provided
	if p.Search != "" {
		// Get the type of the result to dynamically generate search conditions
		modelType := reflect.TypeOf(opts.Result).Elem().Elem()

		// Start with an empty condition
		searchCondition := p.attachSearch(modelType, opts.SearchPfx)

		// Combine the search condition with the existing condition using "AND"
		query = query.Where(searchCondition)
	}

	// Apply join if provided
	if opts.Join != "" {
		query = query.Joins(opts.Join)
	}

	// Apply sorting if provided
	if p.Sort != "" {
		if opts.SortPfx != "" {
			p.Sort = fmt.Sprintf("%s.%s", opts.SortPfx, p.Sort)
		}

		if strings.ToLower(p.SortDir) == PAGINATION_ORDER_DESC {
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

	return PaginationResult{
		Results: opts.Result,
		Info: PaginationResultInfo{
			Page:     page,
			Total:    int(total),
			PerPage:  perPage,
			LastPage: lastPage,
		},
	}
}

func (p *Pagination) attachSearch(modelType reflect.Type, searchPrefix string) string {
	searchCondition := ""

	// Iterate over the fields of the model
	for i := 0; i < modelType.NumField(); i++ {
		tagValue := modelType.Field(i).Tag.Get(UKER_STRUCT_TAG)
		gormTagValues := modelType.Field(i).Tag.Get("gorm")

		if strings.Contains(tagValue, "-") || strings.Contains(strings.ToLower(gormTagValues), "foreignkey") {
			continue
		}

		if modelType.Field(i).Type.Kind() == reflect.Slice {
			continue
		}

		if modelType.Field(i).Type.Kind() == reflect.Struct && modelType.Field(i).Anonymous {
			subSearchCondition := p.attachSearch(modelType.Field(i).Type, searchPrefix)
			if searchCondition == "" {
				searchCondition = subSearchCondition
			} else {
				searchCondition += " OR " + subSearchCondition
			}
			continue
		}

		fieldName := modelType.Field(i).Name

		sqlFieldWords := Str().SplitByUpperCase(fieldName)
		fieldName = strings.ToLower(strings.Join(sqlFieldWords, "_"))

		if modelType.Field(i).Name == "ID" {
			fieldName = "id"
		}

		if searchPrefix != "" {
			fieldName = fmt.Sprintf("%s.%s", searchPrefix, fieldName)
		}

		// Add a condition for the current field
		if searchCondition == "" {
			searchCondition = fieldName + " LIKE " + "'%%" + p.Search + "%%'"
		} else {
			searchCondition += " OR " + fieldName + " LIKE " + "'%%" + p.Search + "%%'"
		}
	}

	return searchCondition
}
