package uker

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"reflect"
	"strings"
)

// New http struct config
type NewHttpParameters struct {
	EncodeBody bool
}

// helper struct
type response struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message"`
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
	// @param w http.ResponseWriter Current fiber context.
	//
	// @param resCode int: Http response code.
	//
	// @param message string: Response message.
	//
	// @param extraValues map[string]string: map with all extras key, value that response need to return.
	//
	// @return error: return fiber response
	FinalOutPut(w http.ResponseWriter, resCode int, message string, extraValues interface{})

	// Parse request body data
	//
	// @param c *fiber.Ctx: Current fiber context.
	//
	// @param requestInterface *interface{}: Interface pointer where parsed data will be stored.
	//
	// @return error: error if exists
	BodyParser(w http.ResponseWriter, r *http.Request, requestInterface interface{}) error

	// Multi part form parser
	//
	// @param c *fiber.Ctx: current fiber context.
	//
	// @param values map[string]*interface{}: map with the value to be parsed and the interface pointer to decode it.
	//
	// @param files []string: string slice with all files that are required of the multipart.
	//
	// @return (map[string][]*multipart.FileHeader, error): map with all files & error if exists
	MultiPartFormParser(w http.ResponseWriter, r *http.Request, values map[string]interface{}, files []string) (map[string][]*multipart.FileHeader, error)

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
	ExtractReqPaginationParameters(r *http.Request) Pagination
}

// Local struct to be implmented
type http_implementation struct {
	Base64Data bool
}

// External contructor
func NewHttp(encodedData bool) Http {
	// return implemented local struct
	return &http_implementation{
		Base64Data: encodedData,
	}
}

func (h *http_implementation) FinalOutPut(w http.ResponseWriter, resCode int, message string, extraValues interface{}) {
	finalOutPut(w, resCode, message, extraValues)
}

func (h *http_implementation) BodyParser(w http.ResponseWriter, r *http.Request, requestInterface interface{}) error {
	// Validate if requestInterface is a pointer
	if reflect.ValueOf(requestInterface).Kind() != reflect.Ptr {
		panic(fmt.Errorf("expected %s as requestInterface, %s received", reflect.Ptr, reflect.ValueOf(requestInterface).Kind()))
	}

	var bodyData map[string]interface{}

	// Read the body of the HTTP Request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return errors.New(ERROR_HTTP_BAD_REQUEST)
	}

	// Parse the JSON encoded in base64
	if err := json.Unmarshal(body, &bodyData); err != nil {
		return errors.New(ERROR_HTTP_BAD_REQUEST)
	}

	err = decodeDataIfEncoded(bodyData["data"], h.Base64Data, &requestInterface)
	if err != nil {
		return err
	}

	// Check if required values on valueInterface are not nil
	if existAllRequiredParams := requiredParamsExists(requestInterface); !existAllRequiredParams {
		return errors.New(ERROR_HTTP_MISSING_PARAMS)
	}

	return nil
}

func (h *http_implementation) MultiPartFormParser(w http.ResponseWriter, r *http.Request, values map[string]interface{}, files []string) (map[string][]*multipart.FileHeader, error) {
	err := r.ParseMultipartForm(10 << 20) // 10MB max
	if err != nil {
		return nil, fmt.Errorf("error parsing multipart form: %v", err)
	}

	// Parse every requested value on the values map
	for value, valueInterface := range values {
		if reflect.ValueOf(valueInterface).Kind() != reflect.Ptr {
			return nil, fmt.Errorf("expected %s as value interface, %s received", reflect.Ptr, reflect.ValueOf(valueInterface).Kind())
		}

		// Get requested FormValue value if exists inside of the multiform
		valueData := r.FormValue(value)

		err := decodeDataIfEncoded(valueData, h.Base64Data, &valueInterface)
		if err != nil {
			return nil, err
		}

		// Check if required values on valueInterface are not nil
		if existAllRequiredParams := requiredParamsExists(valueInterface); !existAllRequiredParams {
			return nil, fmt.Errorf("missing required parameters in valueInterface")
		}
	}

	// Map with all requested files that will be returned
	parsedFiles := map[string][]*multipart.FileHeader{}

	// Parse every requested file on the Files string slice
	for _, file := range files {
		MultipartFormFileHeaders := r.MultipartForm.File[file]
		if MultipartFormFileHeaders != nil {
			parsedFiles[file] = MultipartFormFileHeaders
			continue
		}

		return nil, fmt.Errorf("missing file for key %s", file)
	}

	return parsedFiles, nil
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

func (h *http_implementation) ExtractReqPaginationParameters(r *http.Request) Pagination {
	queryParams := r.URL.Query()
	pagination := Pagination{
		Search:  queryParams.Get(PAGINATION_QUERY_SEARCH),
		Sort:    queryParams.Get(PAGINATION_QUERY_SORT),
		SortDir: queryParams.Get(PAGINATION_QUERY_SORT_DIR),
		Page:    queryParams.Get(PAGINATION_QUERY_PAGE),
		PerPage: queryParams.Get(PAGINATION_QUERY_PERPAGE),
	}

	// Si los valores por defecto no están definidos, asigna los valores por defecto
	if pagination.SortDir == "" {
		pagination.SortDir = PAGINATION_ORDER_ASC
	}
	if pagination.Page == "" {
		pagination.Page = "1"
	}
	if pagination.PerPage == "" {
		pagination.PerPage = "10"
	}

	return pagination
}

// Declaring this local function tu use on all utility files
func finalOutPut(w http.ResponseWriter, resCode int, message string, extraValues interface{}) {
	// crete the response interface
	res := response{
		Code:    resCode,
		Message: message,
	}

	// if extra values is not nil -> add it on data field
	if extraValues != nil {
		res.Data = extraValues
	}

	// return error or success code
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resCode)
	json.NewEncoder(w).Encode(res)
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

func decodeDataIfEncoded(data any, encoded bool, structure *interface{}) error {
	// Check if the 'data' field exists within the JSON in the body
	if data == nil {
		return errors.New(ERROR_HTTP_MISSING_DATA)
	}

	codedJson := ""

	if encoded {
		// Decode the value of the 'data' field from base64
		decoded, err := base64.StdEncoding.DecodeString(data.(string))

		// Check if there was an error while decoding the base64
		if err != nil {
			return errors.New(ERROR_HTTP_INVALID_BASE64)
		}

		codedJson = string(decoded)
	} else {
		bodyJson, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("error while marshalling JSON: %v", err)
		}

		codedJson = string(bodyJson)
	}

	// Parse decoded json string to the specified interface
	if err := json.Unmarshal([]byte(codedJson), structure); err != nil {
		return fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	return nil
}
