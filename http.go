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

// helper structs
type ResponseStatusType string

const (
	ERROR   ResponseStatusType = "error"
	SUCCESS ResponseStatusType = "success"
)

type ResponseStatus struct {
	Type        ResponseStatusType `json:"type"`
	Code        string             `json:"code"`
	Description string             `json:"description,omitempty"`
}

type Response struct {
	Data   interface{}    `json:"data,omitempty"`
	Status ResponseStatus `json:"status"`
}

// Struct with MultiformParser return
type MutiformData struct {
	Values map[string]string
	Files  map[string][]*multipart.FileHeader
}

// Global interface
type Http interface {
	FinalOutPut(w http.ResponseWriter, httpCode int, resStatus *ResponseStatus, extraValues interface{})
	ErrorOutPut(w http.ResponseWriter, httpCode int, resStatus *ResponseStatus)
	BodyParser(w http.ResponseWriter, r *http.Request, requestInterface interface{}) error
	MultiPartFormParser(w http.ResponseWriter, r *http.Request, values map[string]interface{}, files ...string) (map[string][]*multipart.FileHeader, error)
	MultiPartFileToBuff(files []*multipart.FileHeader) [][]byte
	FirstMultiPartFileToBuff(files []*multipart.FileHeader) ([]byte, error)
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

// Create an http response as json string
//
// @param w http.ResponseWriter Current handler writer.
//
// @param resCode int: Http response code.
//
// @param resStatus *ResponseStatus: Response status.
//
// @param extraValues interface{}: interface that will be sended on data field.
func (h *http_implementation) FinalOutPut(w http.ResponseWriter, resCode int, resStatus *ResponseStatus, extraValues interface{}) {
	finalOutPut(w, resCode, resStatus, extraValues)
}

// Create an http error response as json string
//
// @param w http.ResponseWriter Current fiber context.
//
// @param resCode int: Http response code.
//
// @param resStatus *ResponseStatus: Response status.
func (h *http_implementation) ErrorOutPut(w http.ResponseWriter, resCode int, resStatus *ResponseStatus) {
	errorOutPut(w, resCode, resStatus)
}

// Parse request body data
//
// @param c *fiber.Ctx: Current fiber context.
//
// @param requestInterface *interface{}: Interface pointer where parsed data will be stored.
//
// @return error: error if exists
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
	if missingRequiredParamError := requiredParamsExists(requestInterface); missingRequiredParamError != nil {
		return fmt.Errorf("missing required parameter: %s", missingRequiredParamError.Error())
	}

	return nil
}

// Multi part form parser
//
// @param c *fiber.Ctx: current fiber context.
//
// @param values map[string]*interface{}: map with the value to be parsed and the interface pointer to decode it.
//
// @param files []string: string slice with all files that are required of the multipart.
//
// @return (map[string][]*multipart.FileHeader, error): map with all files & error if exists
func (h *http_implementation) MultiPartFormParser(w http.ResponseWriter, r *http.Request, values map[string]interface{}, files ...string) (map[string][]*multipart.FileHeader, error) {
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
		if missingRequiredParamError := requiredParamsExists(valueInterface); missingRequiredParamError != nil {
			return nil, fmt.Errorf("missing required parameters in valueInterface: %s", missingRequiredParamError.Error())
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
	}

	return parsedFiles, nil
}

// Multi part file parser
//
// @param files []*multipart.FileHeader: slice with all multipart files to be added as buff
//
// @return [][]byte: files buffer
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

// Multi part file parser
//
// @param files []*multipart.FileHeader: slice with all multipart files
//
// @return ([]byte, error): file buffer & error if exists
func (h *http_implementation) FirstMultiPartFileToBuff(files []*multipart.FileHeader) ([]byte, error) {
	// Get the first image in case there is more than one
	thisFile, err := files[0].Open()
	if err != nil {
		return nil, fmt.Errorf("cannot open the first file of the slice: %s", err)
	}

	// Create new buffer
	buf := bytes.NewBuffer(nil)

	// Copy file content in the buffer
	if _, err = io.Copy(buf, thisFile); err != nil {
		return nil, fmt.Errorf("cannot copy the first file content to the buffer: %s", err)
	}

	return buf.Bytes(), nil
}

// Extract request pagination parameters
//
// @param r *http.Request: Actual handler http request
//
// @return Pagination: Pagination struct with all request params
func (h *http_implementation) ExtractReqPaginationParameters(r *http.Request) Pagination {
	queryParams := r.URL.Query()
	pagination := Pagination{
		Page:       queryParams.Get(PAGINATION_QUERY_PAGE),
		Sort:       queryParams.Get(PAGINATION_QUERY_SORT),
		SortDir:    queryParams.Get(PAGINATION_QUERY_SORT_DIR),
		Search:     queryParams.Get(PAGINATION_QUERY_SEARCH),
		PerPage:    queryParams.Get(PAGINATION_QUERY_PERPAGE),
		WhereField: queryParams.Get(PAGINATION_QUERY_WHERE_FIELD),
		WhereValue: queryParams.Get(PAGINATION_QUERY_WHERE_VALUE),
	}

	// Validate if search has the base64 prefix and decode it in case of provided
	if pagination.Search != "" && strings.HasPrefix(pagination.Search, "b64!") {
		decoded, err := base64.StdEncoding.DecodeString(strings.Replace(pagination.Search, "b64!", "", -1))
		if err == nil {
			pagination.Search = string(decoded)
		}
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
func finalOutPut(w http.ResponseWriter, resCode int, resStatus *ResponseStatus, extraValues interface{}) {
	// crete the response interface
	res := Response{
		Status: *resStatus,
	}

	// if extra values is not nil -> add it on data field
	if extraValues != nil {
		res.Data = extraValues
	}

	// return error or success code
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resCode)
	json.NewEncoder(w).Encode(res)

	// Flush the response
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func requiredParamsExists(x interface{}) error {
	interfaceType := reflect.TypeOf(x).Elem()
	interfaceValues := reflect.ValueOf(x).Elem()

	for i := 0; i < interfaceType.NumField(); i++ {
		field := interfaceType.Field(i)
		tagValue := field.Tag.Get(UKER_STRUCT_TAG)

		if strings.Contains(tagValue, UKER_STRUCT_TAG_REQUIRED) {
			if interfaceValues.Field(i).Type().Kind() == reflect.String && interfaceValues.Field(i).IsZero() {
				return fmt.Errorf("missing required parameter: %s", field.Name)
			}
		}
	}

	return nil
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
		// If data is not encoded and is not string we have to convert it to string
		if reflect.TypeOf(data).Kind() != reflect.String {
			bodyJson, err := json.Marshal(data)
			if err != nil {
				return fmt.Errorf("error while marshalling JSON: %v", err)
			}

			codedJson = string(bodyJson)
		} else {
			codedJson = data.(string)
		}
	}

	// Parse decoded json string to the specified interface
	if err := json.Unmarshal([]byte(codedJson), structure); err != nil {
		return fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	return nil
}

func errorOutPut(w http.ResponseWriter, resCode int, resStatus *ResponseStatus) {
	// crete the response interface
	res := Response{
		Status: *resStatus,
	}

	// convert response struct into json string
	jsonData, _ := json.Marshal(res)

	// return error
	w.Header().Set("content-type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(resCode)
	fmt.Fprintln(w, string(jsonData))

	// Flush the response
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}
