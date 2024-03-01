package uker_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"reflect"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/unknowns24/uker"
	"github.com/valyala/fasthttp"
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

	// Marshal the test structure to JSON
	testJson, err := json.Marshal(test)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	// Create a simulated fasthttp.RequestCtx object
	ctx := &fasthttp.RequestCtx{}

	// Create a simulated multipart form body
	body := &bytes.Buffer{}
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

	// Set the body of the context with the simulated form
	ctx.Request.SetBody(body.Bytes())
	ctx.Request.Header.SetContentType(writer.FormDataContentType())

	// Create a Fiber app
	app := fiber.New()

	// Acquire the request context into the Fiber app
	c := app.AcquireCtx(ctx)

	// Create a new map with string (value to search in the multipart form) as keys and interface (where the decoded value will be stored) as values
	values := make(map[string]interface{})

	// Create an empty test structure where the final value will be stored
	data := testStruct{}

	// Add the "data" key to the values map
	values["data"] = &data

	// Call the MultiPartFormParser function
	files, err := uker.NewHttp(&uker.NewHttpParameters{
		EncodeBody: true,
	}).MultiPartFormParser(c, values, []string{"file1", "file2"})
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	// Verify that the testStruct was filled correctly
	if data.Param1 != "value1" {
		t.Errorf("Expected Param1 to be 'value1', got '%s'", data.Param1)
	}

	// Verify that the files were processed correctly (simulated)
	if len(files["file1"]) != 0 || len(files["file2"]) != 0 {
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

	// Create a test body with a "data" field that has the base64 encoded testJson
	testBody := map[string]string{
		"data": testBase64,
	}

	// Marshal the testBody map to JSON
	bodyJson, err := json.Marshal(testBody)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	// Create a simulated fasthttp.RequestCtx object
	ctx := &fasthttp.RequestCtx{}

	// Set the body of the context with the simulated JSON
	ctx.Request.SetBody([]byte(bodyJson))
	ctx.Request.Header.SetContentType(fiber.MIMEApplicationJSON)

	// Create a Fiber app
	app := fiber.New()

	// Acquire the request context into the Fiber app
	c := app.AcquireCtx(ctx)

	// Create an empty test structure to save files when decoded
	data := testStruct{}

	// Call the BodyParser function
	err = uker.NewHttp(&uker.NewHttpParameters{
		EncodeBody: true,
	}).BodyParser(c, &data)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	if c.Response().StatusCode() != fiber.StatusOK {
		t.Errorf("Error: %v", c.Response())
	}

	// Verify that the testStruct was filled correctly
	if data.Param1 != "value1" {
		t.Errorf("Expected Param1 to be 'value1', got '%s'", data.Param1)
	}
}

func TestEndOutPut(t *testing.T) {
	const testMsg = "TEST"

	// Create a Fiber app
	app := fiber.New()

	// Create a simulated Fiber context
	c := app.AcquireCtx(&fasthttp.RequestCtx{})

	// Call the endOutPut function with test values
	err := uker.NewHttp(&uker.NewHttpParameters{
		EncodeBody: true,
	}).EndOutPut(c, fiber.StatusOK, testMsg, map[string]string{"key1": "value1", "key2": "value2"})

	// Check for an error
	if err != nil {
		t.Errorf("Error in endOutPut function: %v", err)
	}

	// Check the response status code
	if c.Context().Response.StatusCode() != fiber.StatusOK {
		t.Errorf("Incorrect response status code. Expected %d, but got %d", fiber.StatusOK, c.Context().Response.StatusCode())
	}

	// Decode the JSON response
	var baseResponse map[string]interface{}
	if err := json.Unmarshal(c.Context().Response.Body(), &baseResponse); err != nil {
		t.Errorf("Error decoding the base response to JSON: %v", err)
	}

	encodedData, err := json.Marshal(baseResponse["data"])
	if err != nil {
		t.Errorf("Error encoding data to JSON: %v", err)
	}

	var dataResponse map[string]string
	if err := json.Unmarshal([]byte(encodedData), &dataResponse); err != nil {
		t.Errorf("Error decoding the data content to JSON: %v", err)
	}

	// Verify the response content
	if dataResponse["message"] != testMsg {
		t.Errorf("Incorrect message in the response. Expected '%s', but got '%s'", testMsg, dataResponse["message"])
	}

	// Verify other additional values if needed
	if dataResponse["key1"] != "value1" {
		t.Errorf("Incorrect value for 'key1' in the response. Expected '%s', but got '%s'", "value1", dataResponse["key1"])
	}

	if dataResponse["key2"] != "value2" {
		t.Errorf("Incorrect value for 'key2' in the response. Expected '%s', but got '%s'", "value2", dataResponse["key2"])
	}
}

func TestExtractReqPaginationParameters(t *testing.T) {
	type args struct {
		c *fiber.Ctx
	}

	// Create a Fiber app
	app := fiber.New()

	// Create a simulated Fiber context
	sortTestCtx := app.AcquireCtx(&fasthttp.RequestCtx{})
	sortTestCtx.Request().URI().SetQueryString(fmt.Sprintf("%s=%s&%s=%s", uker.PAGINATION_QUERY_SORT, "id", uker.PAGINATION_QUERY_SORT_DIR, uker.PAGINATION_ORDER_DESC))

	fullTestCtx := app.AcquireCtx(&fasthttp.RequestCtx{})
	fullTestCtx.Request().URI().SetQueryString(fmt.Sprintf("%s=%s&%s=2&%s=5&%s=%s&%s=%s", uker.PAGINATION_QUERY_SEARCH, "ss", uker.PAGINATION_QUERY_PAGE, uker.PAGINATION_QUERY_PERPAGE, uker.PAGINATION_QUERY_SORT, "id", uker.PAGINATION_QUERY_SORT_DIR, uker.PAGINATION_ORDER_DESC))

	emptyCtx := app.AcquireCtx(&fasthttp.RequestCtx{})

	tests := []struct {
		name string
		desc string
		args args
		want uker.Pagination
	}{
		{
			name: "full test",
			desc: "test pagination with all parameter",
			args: args{
				c: fullTestCtx,
			},
			want: uker.Pagination{Search: "ss", Sort: "id", SortDir: uker.PAGINATION_ORDER_DESC, Page: "2", PerPage: "5"},
		},
		{
			name: "empty test",
			desc: "test pagination with none parameter",
			args: args{
				c: emptyCtx,
			},
			want: uker.Pagination{SortDir: uker.PAGINATION_ORDER_ASC, Page: "1", PerPage: "10"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := uker.NewHttp(&uker.NewHttpParameters{
				EncodeBody: true,
			}).ExtractReqPaginationParameters(tt.args.c); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExtractReqPaginationParameters() = %v, want %v", got, tt.want)
			}
		})
	}
}
