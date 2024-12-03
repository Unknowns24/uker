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
	DB              *gorm.DB
	Join            string
	Where           string
	Select          string
	SortPfx         string
	SearchPfx       string
	DynamicWhere    bool
	DynamicWherePfx string
	Result          interface{}
	TableModel      interface{}
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
		if opts.DynamicWherePfx != "" {
			query = query.Where(fmt.Sprintf("%s.%s = ?", opts.DynamicWherePfx, p.WhereField), p.WhereValue)
		} else {
			query = query.Where(fmt.Sprintf("%s = ?", p.WhereField), p.WhereValue)
		}
	}

	// Apply search if provided
	if p.Search != "" {
		// Get the type of the result to dynamically generate search conditions
		modelType := reflect.TypeOf(opts.Result).Elem().Elem()

		// Start with an empty condition
		searchCondition, searchValues := p.attachSearch(modelType, opts.SearchPfx)

		// Combine the search condition with the existing condition using "AND"
		query = query.Where(searchCondition, searchValues...)
	}

	// Apply join if provided
	if opts.Join != "" {
		query = query.Joins(opts.Join)
	}

	// Apply sorting
	p.SortDir = strings.ToLower(p.SortDir)
	if p.SortDir != "asc" && p.SortDir != "desc" {
		p.SortDir = "asc"
	}

	if p.Sort != "" {
		sortField := p.Sort
		if opts.SortPfx != "" {
			sortField = fmt.Sprintf("%s.%s", opts.SortPfx, p.Sort)
		}
		query = query.Order(fmt.Sprintf("%s %s", sortField, strings.ToUpper(p.SortDir)))
	}

	// Convert URL parameters to integers
	page, err := strconv.Atoi(p.Page)
	if err != nil || page < 1 {
		page = 1
	}

	perPage, err := strconv.Atoi(p.PerPage)
	if err != nil || perPage < 1 {
		perPage = 10
	} else if perPage > 100 { // max limit of perPage
		perPage = 100
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

func (p *Pagination) attachSearch(modelType reflect.Type, searchPrefix string) (string, []interface{}) {
	searchCondition := ""
	var searchValues []interface{}

	// Split the search value by ";"
	searchTerms := strings.Split(p.Search, ";")

	// Iterate over the split search terms
	for _, searchTerm := range searchTerms {
		searchTerm = strings.TrimSpace(searchTerm) // Remove unnecessary spaces
		if searchTerm == "" {
			continue
		}

		// Generate the search condition for the current search term
		currentCondition := ""

		// Iterate over the fields of the model
		for i := 0; i < modelType.NumField(); i++ {
			tagValue := modelType.Field(i).Tag.Get(UKER_STRUCT_TAG)
			gormTagValues := modelType.Field(i).Tag.Get("gorm")

			// Skip the field if it has the tag `uker:"-"` or `gorm:"foreignkey"`
			if strings.Contains(tagValue, "-") || strings.Contains(strings.ToLower(gormTagValues), "foreignkey") {
				continue
			}

			// Skip the field if it is a slice
			if modelType.Field(i).Type.Kind() == reflect.Slice {
				continue
			}

			// Recursively process embedded anonymous struct fields
			if modelType.Field(i).Type.Kind() == reflect.Struct && modelType.Field(i).Anonymous {
				subCondition, subValues := p.attachSearch(modelType.Field(i).Type, searchPrefix)
				if currentCondition == "" {
					currentCondition = subCondition
				} else {
					currentCondition += " OR " + subCondition
				}
				searchValues = append(searchValues, subValues...)
				continue
			}

			// Convert the field name to a SQL-friendly format
			fieldName := modelType.Field(i).Name
			sqlFieldWords := Str().SplitByUpperCase(fieldName)
			fieldName = strings.ToLower(strings.Join(sqlFieldWords, "_"))

			// Special case for the "ID" field
			if modelType.Field(i).Name == "ID" {
				fieldName = "id"
			}

			// Add a prefix to the field name if specified
			if searchPrefix != "" {
				fieldName = fmt.Sprintf("%s.%s", searchPrefix, fieldName)
			}

			// Add a condition for the current field
			if currentCondition == "" {
				currentCondition = fmt.Sprintf("%s LIKE ?", fieldName)
			} else {
				currentCondition += fmt.Sprintf(" OR %s LIKE ?", fieldName)
			}
			searchValues = append(searchValues, "%"+searchTerm+"%")
		}

		// Add the generated condition for this search term
		if currentCondition != "" {
			if searchCondition == "" {
				searchCondition = fmt.Sprintf("(%s)", currentCondition)
			} else {
				searchCondition += fmt.Sprintf(" AND (%s)", currentCondition)
			}
		}
	}

	return searchCondition, searchValues
}
