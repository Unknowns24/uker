package uker_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"mime/multipart"
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

	// Crear un contexto simulado de Fiber
	c := app.AcquireCtx(&fasthttp.RequestCtx{})

	// Llamar a la función endOutPut con los valores de prueba
	err := uker.NewHttp("").EndOutPut(c, fiber.StatusOK, testMsg, map[string]string{"key1": "value1", "key2": "value2"})

	// Verificar si hay un error
	if err != nil {
		t.Errorf("Error en la función endOutPut: %v", err)
	}

	// Verificar el código de respuesta
	if c.Context().Response.StatusCode() != fiber.StatusOK {
		t.Errorf("Código de respuesta incorrecto. Se esperaba %d, pero se obtuvo %d", fiber.StatusOK, c.Context().Response.StatusCode())
	}

	// Decodificar la respuesta JSON
	var baseResonse map[string]interface{}
	if err := json.Unmarshal(c.Context().Response.Body(), &baseResonse); err != nil {
		t.Errorf("Error al decodificar la respuesta base en JSON: %v", err)
	}

	encodedData, err := json.Marshal(baseResonse["data"])
	if err != nil {
		t.Errorf("Error al codificat data en JSON: %v", err)
	}

	var dataResponse map[string]string
	if err := json.Unmarshal([]byte(encodedData), &dataResponse); err != nil {
		t.Errorf("Error al decodificar el contenido de data en JSON: %v", err)
	}

	// Verificar el contenido de la respuesta
	if dataResponse["message"] != testMsg {
		t.Errorf("Mensaje incorrecto en la respuesta. Se esperaba '%s', pero se obtuvo '%s'", testMsg, dataResponse["message"])
	}

	// Verificar otros valores adicionales si es necesario
	if dataResponse["key1"] != "value1" {
		t.Errorf("Valor incorrecto para 'key1' en la respuesta. Se esperaba '%s', pero se obtuvo '%s'", "value1", dataResponse["key1"])
	}

	if dataResponse["key2"] != "value2" {
		t.Errorf("Valor incorrecto para 'key2' en la respuesta. Se esperaba '%s', pero se obtuvo '%s'", "value2", dataResponse["key2"])
	}
}
