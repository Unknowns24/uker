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
	Param1 string
	Param2 string
}

func TestMultiPartFormParser(t *testing.T) {
	test := testStruct{
		Param1: "value1",
		Param2: "value2",
	}

	testJson, err := json.Marshal(test)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	// Crear un objeto fasthttp.RequestCtx simulado
	ctx := &fasthttp.RequestCtx{}

	// Crear un cuerpo de formulario multiparte simulado
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Encoding base64 multiform value data
	testBase64 := base64.StdEncoding.EncodeToString(testJson)

	// Agregar un campo "data" con valores
	writer.WriteField("data", testBase64)

	// Agregar un archivo simulado
	fileWriter, _ := writer.CreateFormFile("file1", "example.txt")
	fileWriter.Write([]byte("simulated file content"))

	// Finalizar el formulario multiparte
	writer.Close()

	// Configurar el cuerpo del contexto con el formulario simulado
	ctx.Request.SetBody(body.Bytes())
	ctx.Request.Header.SetContentType(writer.FormDataContentType())

	// Crear app de fiber
	app := fiber.New()

	// Agregarle el contexto de la request a la app de fiber
	c := app.AcquireCtx(ctx)

	// Create a new map with string (value to search on the multipartform) as key and the interface (where decoded value will stored) as value
	values := make(map[string]interface{})

	// Create an empty test structure where final value will be stored
	data := testStruct{}

	// Add row to test to values map
	values["data"] = &data

	// Llamar a la función MultiPartFormParser
	files, err := uker.NewHttp("").MultiPartFormParser(c, values, []string{"file1", "file2"})
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	// Verificar que la estructura createAppRequest fue llenada correctamente
	if data.Param1 != "value1" {
		t.Errorf("Expected Name to be 'value1', got '%s'", data.Param1)
	}

	// Verificar que los archivos se hayan procesado correctamente (simulados)
	if len(files["file1"]) != 0 || len(files["file2"]) != 0 {
		t.Errorf("Expected files, got no files")
	}
}

func TestBodyParser(t *testing.T) {
	// Create test structure
	test := testStruct{
		Param1: "value1",
		Param2: "value2",
	}

	// Marshal test structure to json
	testJson, err := json.Marshal(test)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	// Encoding on base64 testJson string
	testBase64 := base64.StdEncoding.EncodeToString(testJson)

	// Create test body with a data field which have the base64 encoded testJson
	testBody := map[string]string{
		"data": testBase64,
	}

	// Marshal testBody map to json
	bodyJson, err := json.Marshal(testBody)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	// Crear un objeto fasthttp.RequestCtx simulado
	ctx := &fasthttp.RequestCtx{}

	// Configurar el cuerpo del contexto con el JSON simulado
	ctx.Request.SetBody([]byte(bodyJson))
	ctx.Request.Header.SetContentType(fiber.MIMEApplicationJSON)

	// Crear app de fiber
	app := fiber.New()

	// Agregarle el contexto de la request a la app de fiber
	c := app.AcquireCtx(ctx)

	// Create an empty test structure to save files when decoded
	data := testStruct{}

	// Llamar a la función BodyParser
	err = uker.NewHttp("").BodyParser(c, &data)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	// Verificar que la estructura createAppRequest fue llenada correctamente
	if data.Param1 != "value1" {
		t.Errorf("Expected Param1 to be 'value1', got '%s'", data.Param1)
	}
}
