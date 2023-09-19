package uker

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"mime/multipart"
	"reflect"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// request constants
const (
	request_key_data    = "data"
	request_key_message = "message"
)

// pagination constants
const (
	pagination_order_asc      = "asc"
	pagination_order_desc     = "desc"
	pagination_query_sort     = "sort"
	pagination_query_page     = "page"
	pagination_query_search   = "search"
	pagination_query_per_page = "per_page"
	pagination_query_sort_dir = "sort_dir"
)

// Variable to store application response sufix
var appSuffix string

// helper struct
type response struct {
	Code int               `json:"code"`
	Data map[string]string `json:"data"`
}

// Struct with MultiformParser return
type MutiformData struct {
	Values map[string]string
	Files  map[string][]*multipart.FileHeader
}

// Global interface
type http interface {
	// Server data pagination
	//
	// @param c *fiber.Ctx: current fiber context.
	//
	// @param db *gorm.DB: Database pointer to perform the pagination.
	//
	// @param tableName string: Name of the table to paginate.
	//
	// @param condition string: Where condition to add to the pagination if necessary.
	//
	// @param result interface{}: Interface of wantend result.
	//
	// @return (fiber.Map, error): map with all paginated data & error if exists
	Paginate(c *fiber.Ctx, db *gorm.DB, tableName string, condition string, result interface{}) (fiber.Map, error)

	// Create a fiber response as json string
	//
	// @param c *fiber.Ctx: Current fiber context.
	//
	// @param resCode int: Http response code.
	//
	// @param message string: Response message.
	//
	// @param extraValues map[string]string: map with all extras key, value that response need to return.
	//
	// @return error: return fiber response
	EndOutPut(c *fiber.Ctx, resCode int, message string, extraValues map[string]string) error

	// Parse request body data
	//
	// @param c *fiber.Ctx: Current fiber context.
	//
	// @param requestInterface *interface{}: Interface pointer where parsed data will be stored.
	//
	// @return error: error if exists
	BodyParser(c *fiber.Ctx, requestInterface interface{}) error

	// Multi part form parser
	//
	// @param c *fiber.Ctx: current fiber context.
	//
	// @param values map[string]*interface{}: map with the value to be parsed and the interface pointer to decode it.
	//
	// @param files []string: string slice with all files that are required of the multipart.
	//
	// @return (map[string][]*multipart.FileHeader, error): map with all files & error if exists
	MultiPartFormParser(ctx *fiber.Ctx, values map[string]interface{}, files []string) (map[string][]*multipart.FileHeader, error)
}

// Local struct to be implmented
type http_implementation struct{}

// External contructor
func NewHttp(appResponseSuffix string) http {
	// Make app suffix local to access it from EndOutPut
	appSuffix = appResponseSuffix

	// return implemented local struct
	return &http_implementation{}
}

func (h *http_implementation) Paginate(c *fiber.Ctx, db *gorm.DB, tableName string, condition string, result interface{}) (fiber.Map, error) {
	// Build a base query without conditions
	query := db.Model(result).Table(tableName)

	if condition != "" {
		query = query.Where(condition)
	}

	// Apply search if provided
	if search := c.Query(pagination_query_search); search != "" {
		// Search conditions for non-ID fields
		query = query.Where("NOT id = ? AND (column1 LIKE ? OR column2 LIKE ?)", 0, "%"+search+"%", "%"+search+"%")
	}

	// Apply sorting if provided
	if sort := c.Query(pagination_query_sort); sort != "" {
		if sortDir := c.Query(pagination_query_sort_dir, pagination_order_asc); sortDir == pagination_order_desc {
			query = query.Order(fmt.Sprintf("%s %s", sort, strings.ToUpper(pagination_order_desc)))
		} else {
			query = query.Order(sort)
		}
	}

	// Convert URL parameters to integers
	page, err1 := strconv.Atoi(c.Query(pagination_query_page, "1"))
	perPage, err2 := strconv.Atoi(c.Query(pagination_query_per_page, "10"))

	if err1 != nil || err2 != nil {
		return nil, endOutPut(c, fiber.StatusBadRequest, ERROR_HTTP_BAD_REQUEST, nil)
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
	var paginatedResult interface{}
	query.Limit(perPage).Offset((page - 1) * perPage).Find(paginatedResult)

	return fiber.Map{
		"page":      page,
		"total":     total,
		"per_page":  perPage,
		"last_page": lastPage,
		"data":      paginatedResult,
	}, nil
}

func (h *http_implementation) EndOutPut(c *fiber.Ctx, resCode int, message string, extraValues map[string]string) error {
	return endOutPut(c, resCode, message, extraValues)
}

func (h *http_implementation) BodyParser(c *fiber.Ctx, requestInterface interface{}) error {
	// Validate if requestInterface is a pointer
	if reflect.ValueOf(requestInterface).Kind() != reflect.Ptr {
		return fmt.Errorf("expected %s as requestInterface, %s received", reflect.Ptr, reflect.ValueOf(requestInterface).Kind())
	}

	var bodyData map[string]string

	// Parse the content sent in the body
	if err := c.BodyParser(&bodyData); err != nil {
		return endOutPut(c, fiber.StatusBadRequest, ERROR_HTTP_INVALID_JSON, nil)
	}

	// Check if the 'data' field exists within the JSON in the body
	if bodyData[request_key_data] == "" {
		return endOutPut(c, fiber.StatusBadRequest, ERROR_HTTP_MISSING_DATA, nil)
	}

	// Decode the value of the 'data' field from base64
	decoded, err := base64.StdEncoding.DecodeString(bodyData[request_key_data])

	// Check if there was an error while decoding the base64
	if err != nil {
		return endOutPut(c, fiber.StatusBadRequest, ERROR_HTTP_INVALID_BASE64, nil)
	}

	// Parse the JSON encoded in base64
	if err := json.Unmarshal([]byte(string(decoded)), &requestInterface); err != nil {
		return endOutPut(c, fiber.StatusBadRequest, ERROR_HTTP_BAD_REQUEST, nil)
	}

	return nil
}

func (h *http_implementation) MultiPartFormParser(ctx *fiber.Ctx, values map[string]interface{}, files []string) (map[string][]*multipart.FileHeader, error) {
	// Get MultiparForm pointer
	MultipartForm, err := ctx.MultipartForm()
	if err != nil {
		return nil, endOutPut(ctx, fiber.StatusBadRequest, ERROR_HTTP_MULTIPARTFORM_INVALID_FORM, nil)
	}

	// Parse every requested value on the values map
	for value, valueInterface := range values {
		if reflect.ValueOf(valueInterface).Kind() != reflect.Ptr {
			return nil, fmt.Errorf("expected %s as value interface, %s received", reflect.Ptr, reflect.ValueOf(valueInterface).Kind())
		}

		// Get requested FormValue value if exists inside of the multiform
		valueData := ctx.FormValue(value, "")
		// Check if field exists
		if valueData == "" {
			return nil, endOutPut(ctx, fiber.StatusBadRequest, ERROR_HTTP_BAD_REQUEST, nil)
		}

		// Decoding base64 multiform value data
		decoded, err := base64.StdEncoding.DecodeString(valueData)

		// Check if error happens on base64 decoding
		if err != nil {
			return nil, endOutPut(ctx, fiber.StatusBadRequest, ERROR_HTTP_INVALID_BASE64, nil)
		}

		// Parse decoded json string to the specified interface
		if err := json.Unmarshal(decoded, &valueInterface); err != nil {
			return nil, endOutPut(ctx, fiber.StatusBadRequest, ERROR_HTTP_INVALID_JSON, nil)
		}
	}

	// Map with all requested files that will be returned
	ParsedFiles := map[string][]*multipart.FileHeader{}

	// Parse every requested file on the Files string slice
	for _, file := range files {
		if MultipartFormFile := MultipartForm.File[file]; MultipartFormFile != nil {
			ParsedFiles[file] = MultipartFormFile
			continue
		}

		return nil, endOutPut(ctx, fiber.StatusBadRequest, ERROR_HTTP_MULTIPARTFORM_MISSING_FILES, nil)
	}

	return ParsedFiles, nil
}

// Declaring this local function tu use on all utility files
func endOutPut(c *fiber.Ctx, resCode int, message string, extraValues map[string]string) error {
	// if extra values is nil -> set it as an empty map[string]string
	if extraValues == nil {
		extraValues = map[string]string{}
	}

	// Add message to the map
	extraValues[request_key_message] = fmt.Sprintf("%s%s", message, appSuffix)

	// Encode response as json
	jsonData, _ := json.Marshal(response{Data: extraValues, Code: resCode})

	// return error or success code
	return c.Status(resCode).SendString(string(jsonData))
}