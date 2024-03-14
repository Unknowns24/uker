package uker_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/unknowns24/uker"
)

type testStruct struct {
	Param1 string `uker:"required"`
	Param2 string `uker:"required"`
	Param3 int    `uker:"required"`
	Param4 bool
}

func TestMultiPartFormParser(t *testing.T) {
	// Create a test structure
	test := testStruct{
		Param1: "value1",
		Param2: "value2",
	}

	// Encode the test structure to JSON
	testJson, err := json.Marshal(test)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	// Create a bytes.Buffer object to store the request body
	body := &bytes.Buffer{}

	// Create a multipart.Writer object to write to the request body
	writer := multipart.NewWriter(body)

	// Encode the testJson string in base64
	testBase64 := base64.StdEncoding.EncodeToString(testJson)

	// Add a "data" field with values
	writer.WriteField("data", testBase64)

	// Add a simulated file
	fileWriter, _ := writer.CreateFormFile("file1", "example.txt")
	fileWriter.Write([]byte("simulated file content"))

	// Finalize the multipart form
	writer.Close()

	// Create an HTTP request with the simulated body
	req, err := http.NewRequest("POST", "/", body)
	if err != nil {
		t.Errorf("Error creating HTTP request: %v", err)
	}

	// Set the content header to multipart form content type
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Create a map[string]interface{} to store decoded values
	values := make(map[string]interface{})

	// Create an empty test structure where the final value will be stored
	data := testStruct{}

	// Add the "data" key to the values map
	values["data"] = &data

	// Create a http.ResponseWriter object to capture output
	res := httptest.NewRecorder()

	// Call the MultiPartFormParser function
	files, err := uker.NewHttp(true).MultiPartFormParser(res, req, values, []string{"file1"})
	if err != nil {
		t.Errorf("Error in MultiPartFormParser function: %v", err)
	}

	// Verify that the test structure was filled correctly
	if data.Param1 != "value1" {
		t.Errorf("Expected Param1 to be 'value1', got '%s'", data.Param1)
	}

	// Verify that files were processed correctly (simulated)
	if len(files["file1"]) == 0 {
		t.Errorf("Expected files, got no files")
	}
}

func TestBodyParser(t *testing.T) {
	// Create a test structure
	test := testStruct{
		Param1: "value1",
		Param2: "value2",
		Param3: 0,
		Param4: false,
	}

	// Marshal the test structure to JSON
	testJson, err := json.Marshal(test)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	// Encode the testJson string in base64
	testBase64 := base64.StdEncoding.EncodeToString(testJson)

	// Create a test body with a "data" field containing the base64 encoded JSON
	testBody := map[string]string{
		"data": testBase64,
	}

	// Marshal the test map to JSON
	bodyJson, err := json.Marshal(testBody)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	// Create an HTTP test request with the JSON body
	req, err := http.NewRequest("POST", "/", strings.NewReader(string(bodyJson)))
	if err != nil {
		t.Fatal(err)
	}

	// Set the Content-Type header
	req.Header.Set("Content-Type", "application/json")

	// Create a simulated http.ResponseWriter
	rec := httptest.NewRecorder()

	// Handle the HTTP request with the BodyParser
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create an empty structure to hold files when decoded
		var data testStruct

		// Decode the JSON body of the request and fill the data structure
		err := uker.NewHttp(true).BodyParser(w, r, &data)
		if err != nil {
			t.Errorf("Error: %v", err)
			return
		}

		// Verify that the test structure was filled correctly
		if data.Param1 != "value1" {
			t.Errorf("Expected Param1 to be 'value1', got '%s'", data.Param1)
		}

		// Respond with an HTTP OK status
		w.WriteHeader(http.StatusOK)
	})

	// Serve the HTTP request
	handler.ServeHTTP(rec, req)

	// Verify the response status code
	if status := rec.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestFinalOutPut(t *testing.T) {
	const testMsg = "TEST"

	// Create a http.ResponseWriter to capture output
	res := httptest.NewRecorder()

	// Call the FinalOutPut function with test values
	uker.NewHttp(true).FinalOutPut(res, http.StatusOK, testMsg, map[string]string{"key1": "value1", "key2": "value2"})

	// Verify the response status code
	if res.Code != http.StatusOK {
		t.Errorf("Incorrect response status code. Expected %d, but got %d", http.StatusOK, res.Code)
	}

	// Decode the JSON response
	var baseResponse map[string]interface{}
	if err := json.Unmarshal(res.Body.Bytes(), &baseResponse); err != nil {
		t.Errorf("Error decoding the base response to JSON: %v", err)
	}

	// Verify the response content
	if baseResponse["message"] != testMsg {
		t.Errorf("Incorrect message in the response. Expected '%s', but got '%s'", testMsg, baseResponse["message"])
	}

	// Decode the data section of the JSON response
	encodedData, err := json.Marshal(baseResponse["data"])
	if err != nil {
		t.Errorf("Error encoding data to JSON: %v", err)
	}

	var dataResponse map[string]string
	if err := json.Unmarshal([]byte(encodedData), &dataResponse); err != nil {
		t.Errorf("Error decoding the data content to JSON: %v", err)
	}

	// Verify other additional values if needed
	if dataResponse["key1"] != "value1" {
		t.Errorf("Incorrect value for 'key1' in the response. Expected '%s', but got '%s'", "value1", dataResponse["key1"])
	}

	if dataResponse["key2"] != "value2" {
		t.Errorf("Incorrect value for 'key2' in the response. Expected '%s', but got '%s'", "value2", dataResponse["key2"])
	}
}

func TestErrorOutPut(t *testing.T) {
	const testMsg = "TEST"

	// Create a http.ResponseWriter to capture output
	res := httptest.NewRecorder()

	// Call the ErrorOutPut function with test values
	uker.NewHttp(true).ErrorOutPut(res, http.StatusOK, testMsg)

	// Verify the response status code
	if res.Code != http.StatusOK {
		t.Errorf("Incorrect response status code. Expected %d, but got %d", http.StatusOK, res.Code)
	}

	contentType := res.Header().Get("content-type")
	if contentType != "application/json" {
		t.Errorf("Incorrect content type. Expected application/json, but got %s", contentType)
	}

	// Decode the JSON response
	var baseResponse map[string]interface{}
	if err := json.Unmarshal(res.Body.Bytes(), &baseResponse); err != nil {
		t.Errorf("Error decoding the base response to JSON: %v", err)
	}

	// Verify the response content
	if baseResponse["message"] != testMsg {
		t.Errorf("Incorrect message in the response. Expected '%s', but got '%s'", testMsg, baseResponse["message"])
	}
}

func TestExtractReqPaginationParameters(t *testing.T) {
	// Create an HTTP request with different query parameters

	fullTestReq, err := http.NewRequest("GET", "/?search=ss&page=2&sort=id&per_page=5&sort_dir=desc", nil)
	if err != nil {
		t.Errorf("Error creating HTTP request: %v", err)
	}

	emptyReq, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Errorf("Error creating HTTP request: %v", err)
	}

	tests := []struct {
		name string
		desc string
		req  *http.Request
		want uker.Pagination
	}{
		{
			name: "full test",
			desc: "test pagination with all parameters",
			req:  fullTestReq,
			want: uker.Pagination{Search: "ss", Sort: "id", SortDir: uker.PAGINATION_ORDER_DESC, Page: "2", PerPage: "5"},
		},
		{
			name: "empty test",
			desc: "test pagination with none parameters",
			req:  emptyReq,
			want: uker.Pagination{SortDir: uker.PAGINATION_ORDER_ASC, Page: "1", PerPage: "10"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uker.NewHttp(true).ExtractReqPaginationParameters(tt.req)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExtractReqPaginationParameters() = %v, want %v", got, tt.want)
			}
		})
	}
}
