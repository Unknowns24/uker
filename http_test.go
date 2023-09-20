package uker_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/unknowns24/uker"
	"github.com/valyala/fasthttp"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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
	files, err := uker.NewHttp("").MultiPartFormParser(c, values, []string{"file1", "file2"})
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
	err = uker.NewHttp("").BodyParser(c, &data)
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
	err := uker.NewHttp("").EndOutPut(c, fiber.StatusOK, testMsg, map[string]string{"key1": "value1", "key2": "value2"})

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

func TestPaginate(t *testing.T) {
	type TestProduct struct {
		Id    uint   `json:"id" gorm:"primary_key"`
		State uint   `json:"state"`
		Name  string `json:"name" gorm:"unique"`
		Desc  string `json:"description"`
	}

	// Create a GORM DB instance with a SQLite driver
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		t.Fatalf("Error creating GORM DB: %v", err)
	}

	// Create a Fiber app
	app := fiber.New()

	// Create a simulated Fiber context
	c := app.AcquireCtx(&fasthttp.RequestCtx{})

	// Import table
	db.AutoMigrate(&TestProduct{})

	// create test products
	tp1 := TestProduct{Name: "tp1", State: 1, Desc: "ssa"}
	tp2 := TestProduct{Name: "tp2", State: 0}
	tp3 := TestProduct{Name: "tp3ss", State: 1}
	tp4 := TestProduct{Name: "ss", State: 2}

	db.Create(&tp1)
	db.Create(&tp2)
	db.Create(&tp3)
	db.Create(&tp4)

	var result []TestProduct

	// Call the Paginate function
	paginationResult, err := uker.NewHttp("").Paginate(c, db, "test_products", "state != 0", &result)

	// Check if there is an error
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	// Check if the pagination result is not nil
	if paginationResult == nil {
		t.Errorf("Pagination result is nil")
	}

	// Check if the pagination result has the correct keys
	expectedKeys := []string{"page", "total", "per_page", "last_page", "data"}
	for _, key := range expectedKeys {
		if _, ok := paginationResult[key]; !ok {
			t.Errorf("Pagination result does not have key: %s", key)
		}
	}

	// Get all products inside data
	jsonData, _ := json.Marshal(paginationResult["data"])

	var resProducts []TestProduct
	json.Unmarshal(jsonData, &resProducts)

	for _, product := range resProducts {
		if product.State == 0 {
			t.Error("Pagination result where codition does not work, unexpected product state returned")
		}
	}

	// Set query params to search
	c.Request().URI().SetQueryString(fmt.Sprintf("%s=%s&%s=1&%s=%s&%s=%s", uker.PAGINATION_QUERY_SEARCH, "ss", uker.PAGINATION_QUERY_PERPAGE, uker.PAGINATION_QUERY_SORT, "id", uker.PAGINATION_QUERY_SORT_DIR, uker.PAGINATION_ORDER_DESC))

	// test if params are working
	var result2 []TestProduct

	// Call the Paginate function
	paginationResult2, err := uker.NewHttp("").Paginate(c, db, "test_products", "state != 2", &result2)

	// Check if there is an error
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	if paginationResult2["last_page"] != 2 {
		t.Error("Per page not working")
	}

	if paginationResult2["total"] != int64(2) {
		t.Errorf("Some clause is not working, expected total 2 -> %s received", paginationResult2["total"])
	}
}
