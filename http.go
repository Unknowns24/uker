package uker

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"reflect"
	"strings"

	"github.com/gofiber/fiber/v2"
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
type Http interface {

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

	// Multi part file parser
	//
	// @param files []*multipart.FileHeader: slice with all multipart files to be added as buff
	//
	// @return [][]byte: files buffer
	MultiPartFileToBuff(files []*multipart.FileHeader) [][]byte

	// Multi part file parser
	//
	// @param files []*multipart.FileHeader: slice with all multipart files
	//
	// @return ([][]byte, error): file buffer & error if exists
	FirstMultiPartFileToBuff(files []*multipart.FileHeader) ([][]byte, error)

	// Extract request pagination parameters
	//
	// @param c *fiber.Ctx: fiber request context
	//
	// @return Pagination: Pagination struct with all request params
	ExtractReqPaginationParameters(c *fiber.Ctx) Pagination
}

// Local struct to be implmented
type http_implementation struct{}

// External contructor
func NewHttp(appResponseSuffix string) Http {
	// Make app suffix local to access it from EndOutPut
	appSuffix = appResponseSuffix

	// return implemented local struct
	return &http_implementation{}
}

func (h *http_implementation) EndOutPut(c *fiber.Ctx, resCode int, message string, extraValues map[string]string) error {
	return endOutPut(c, resCode, message, extraValues)
}

func (h *http_implementation) BodyParser(c *fiber.Ctx, requestInterface interface{}) error {
	// Validate if requestInterface is a pointer
	if reflect.ValueOf(requestInterface).Kind() != reflect.Ptr {
		panic(fmt.Errorf("expected %s as requestInterface, %s received", reflect.Ptr, reflect.ValueOf(requestInterface).Kind()))
	}

	var bodyData map[string]string

	// Parse the content sent in the body
	if err := c.BodyParser(&bodyData); err != nil {
		return h.EndOutPut(c, fiber.StatusBadRequest, ERROR_HTTP_INVALID_JSON, nil)
	}

	// Check if the 'data' field exists within the JSON in the body
	if bodyData[REQUEST_KEY_DATA] == "" {
		return h.EndOutPut(c, fiber.StatusBadRequest, ERROR_HTTP_MISSING_DATA, nil)
	}

	// Decode the value of the 'data' field from base64
	decoded, err := base64.StdEncoding.DecodeString(bodyData[REQUEST_KEY_DATA])

	// Check if there was an error while decoding the base64
	if err != nil {
		return h.EndOutPut(c, fiber.StatusBadRequest, ERROR_HTTP_INVALID_BASE64, nil)
	}

	// Parse the JSON encoded in base64
	if err := json.Unmarshal([]byte(string(decoded)), &requestInterface); err != nil {
		return h.EndOutPut(c, fiber.StatusBadRequest, ERROR_HTTP_BAD_REQUEST, nil)
	}

	// Check if required values on valueInterface are not nil
	if existAllRequiredParams := requiredParamsExists(requestInterface); !existAllRequiredParams {
		return h.EndOutPut(c, fiber.StatusBadRequest, ERROR_HTTP_MISSING_PARAMS, nil)
	}

	return nil
}

func (h *http_implementation) MultiPartFormParser(ctx *fiber.Ctx, values map[string]interface{}, files []string) (map[string][]*multipart.FileHeader, error) {
	// Get MultiparForm pointer
	MultipartForm, err := ctx.MultipartForm()
	if err != nil {
		return nil, h.EndOutPut(ctx, fiber.StatusBadRequest, ERROR_HTTP_MULTIPARTFORM_INVALID_FORM, nil)
	}

	// Parse every requested value on the values map
	for value, valueInterface := range values {
		if reflect.ValueOf(valueInterface).Kind() != reflect.Ptr {
			panic(fmt.Errorf("expected %s as value interface, %s received", reflect.Ptr, reflect.ValueOf(valueInterface).Kind()))
		}

		// Get requested FormValue value if exists inside of the multiform
		valueData := ctx.FormValue(value, "")

		// Check if field exists
		if valueData == "" {
			return nil, h.EndOutPut(ctx, fiber.StatusBadRequest, ERROR_HTTP_BAD_REQUEST, nil)
		}

		// Decoding base64 multiform value data
		decoded, err := base64.StdEncoding.DecodeString(valueData)

		// Check if error happens on base64 decoding
		if err != nil {
			return nil, h.EndOutPut(ctx, fiber.StatusBadRequest, ERROR_HTTP_INVALID_BASE64, nil)
		}

		// Parse decoded json string to the specified interface
		if err := json.Unmarshal(decoded, &valueInterface); err != nil {
			return nil, h.EndOutPut(ctx, fiber.StatusBadRequest, ERROR_HTTP_INVALID_JSON, nil)
		}

		// Check if required values on valueInterface are not nil
		if existAllRequiredParams := requiredParamsExists(valueInterface); !existAllRequiredParams {
			return nil, h.EndOutPut(ctx, fiber.StatusBadRequest, ERROR_HTTP_MISSING_PARAMS, nil)
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

		return nil, h.EndOutPut(ctx, fiber.StatusBadRequest, ERROR_HTTP_MULTIPARTFORM_MISSING_FILES, nil)
	}

	return ParsedFiles, nil
}

func (h *http_implementation) MultiPartFileToBuff(files []*multipart.FileHeader) [][]byte {
	filesBuffers := make([][]byte, len(files))

	for k, file := range files {
		thisFile, err := file.Open()
		if err != nil {
			continue
		}

		buf := bytes.NewBuffer(nil)
		_, err = io.Copy(buf, thisFile)

		if err != nil {
			continue
		}

		filesBuffers[k] = buf.Bytes()
	}

	return filesBuffers
}

func (h *http_implementation) FirstMultiPartFileToBuff(files []*multipart.FileHeader) ([][]byte, error) {
	fileBuff := make([][]byte, 1)

	// Get the first image in case there is more than one
	thisFile, err := files[0].Open()
	if err != nil {
		return nil, fmt.Errorf("cannot open the first file of the slice: %s", err)
	}

	buf := bytes.NewBuffer(nil)
	_, err = io.Copy(buf, thisFile)

	if err != nil {
		return nil, fmt.Errorf("cannot copy the first file content to the buffer: %s", err)
	}

	fileBuff[0] = buf.Bytes()

	return fileBuff, nil
}

func (h *http_implementation) ExtractReqPaginationParameters(c *fiber.Ctx) Pagination {
	return Pagination{
		Search:  c.Query(PAGINATION_QUERY_SEARCH),
		Sort:    c.Query(PAGINATION_QUERY_SORT),
		SortDir: c.Query(PAGINATION_QUERY_SORT_DIR, PAGINATION_ORDER_ASC),
		Page:    c.Query(PAGINATION_QUERY_PAGE, "1"),
		PerPage: c.Query(PAGINATION_QUERY_PERPAGE, "10"),
	}
}

// Declaring this local function tu use on all utility files
func endOutPut(c *fiber.Ctx, resCode int, message string, extraValues map[string]string) error {
	// if extra values is nil -> set it as an empty map[string]string
	if extraValues == nil {
		extraValues = map[string]string{}
	}

	// Add message to the map
	extraValues[REQUEST_KEY_MESSAGE] = fmt.Sprintf("%s%s", message, appSuffix)

	// Encode response as json
	jsonData, _ := json.Marshal(response{Data: extraValues, Code: resCode})

	// return error or success code
	return c.Status(resCode).SendString(string(jsonData))
}

func requiredParamsExists(x interface{}) bool {
	interfaceType := reflect.TypeOf(x).Elem()
	interfaceValues := reflect.ValueOf(x).Elem()

	for i := 0; i < interfaceType.NumField(); i++ {
		field := interfaceType.Field(i)
		tagValue := field.Tag.Get(UKER_STRUCT_TAG)

		if strings.Contains(tagValue, UKER_STRUCT_TAG_REQUIRED) {
			if interfaceValues.Field(i).Type().Kind() == reflect.String && interfaceValues.Field(i).IsZero() {
				return false
			}
		}
	}

	return true
}
