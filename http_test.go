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
	// Crear una estructura de prueba
	test := testStruct{
		Param1: "value1",
		Param2: "value2",
	}

	// Codificar la estructura de prueba a JSON
	testJson, err := json.Marshal(test)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	// Crear un objeto bytes.Buffer para almacenar el cuerpo de la solicitud
	body := &bytes.Buffer{}

	// Crear un objeto multipart.Writer para escribir en el cuerpo de la solicitud
	writer := multipart.NewWriter(body)

	// Codificar el JSON de prueba en base64
	testBase64 := base64.StdEncoding.EncodeToString(testJson)

	// Agregar un campo "data" con los valores
	writer.WriteField("data", testBase64)

	// Agregar un archivo simulado
	fileWriter, _ := writer.CreateFormFile("file1", "example.txt")
	fileWriter.Write([]byte("simulated file content"))

	// Finalizar el formulario multipart
	writer.Close()

	// Crear una solicitud HTTP con el cuerpo simulado
	req, err := http.NewRequest("POST", "/", body)
	if err != nil {
		t.Errorf("Error creando la solicitud HTTP: %v", err)
	}

	// Establecer el encabezado de contenido en el tipo de contenido del formulario multipart
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Crear un objeto map[string]interface{} para almacenar los valores decodificados
	values := make(map[string]interface{})

	// Crear una estructura de prueba vacía donde se almacenará el valor final
	data := testStruct{}

	// Agregar la clave "data" al mapa de valores
	values["data"] = &data

	// Crear un objeto http.ResponseWriter para capturar la salida
	// de la función MultiPartFormParser
	res := httptest.NewRecorder()

	// Llamar a la función MultiPartFormParser
	files, err := uker.NewHttp(true).MultiPartFormParser(res, req, values, []string{"file1"})
	if err != nil {
		t.Errorf("Error en la función MultiPartFormParser: %v", err)
	}

	// Verificar que la estructura de prueba se haya llenado correctamente
	if data.Param1 != "value1" {
		t.Errorf("Se esperaba que Param1 fuera 'value1', se obtuvo '%s'", data.Param1)
	}

	// Verificar que los archivos se hayan procesado correctamente (simulados)
	if len(files["file1"]) == 0 {
		t.Errorf("Se esperaban archivos, no se obtuvieron archivos")
	}
}

func TestBodyParser(t *testing.T) {
	// Crear una estructura de prueba
	test := testStruct{
		Param1: "value1",
		Param2: "value2",
		Param3: 0,
		Param4: false,
	}

	// Marshalizar la estructura de prueba a JSON
	testJson, err := json.Marshal(test)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	// Codificar la cadena JSON de prueba en base64
	testBase64 := base64.StdEncoding.EncodeToString(testJson)

	// Crear un cuerpo de prueba con un campo "data" que tenga el JSON codificado en base64
	testBody := map[string]string{
		"data": testBase64,
	}

	// Marshalizar el mapa de prueba a JSON
	bodyJson, err := json.Marshal(testBody)
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	// Crear una solicitud HTTP de prueba con el cuerpo JSON
	req, err := http.NewRequest("POST", "/", strings.NewReader(string(bodyJson)))
	if err != nil {
		t.Fatal(err)
	}

	// Configurar la cabecera Content-Type
	req.Header.Set("Content-Type", "application/json")

	// Crear un registrador de respuesta HTTP simulado
	rec := httptest.NewRecorder()

	// Manejar la solicitud HTTP con el BodyParser
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Crear una estructura vacía para guardar los archivos cuando se decodifiquen
		var data testStruct

		// Decodificar el cuerpo JSON de la solicitud y llenar la estructura de datos
		err := uker.NewHttp(true).BodyParser(w, r, &data)
		if err != nil {
			t.Errorf("Error: %v", err)
			return
		}

		// Verificar que la estructura de prueba se llenó correctamente
		if data.Param1 != "value1" {
			t.Errorf("Expected Param1 to be 'value1', got '%s'", data.Param1)
		}

		// Responder con un estado HTTP OK
		w.WriteHeader(http.StatusOK)
	})

	// Servir la solicitud HTTP
	handler.ServeHTTP(rec, req)

	// Verificar el código de estado de la respuesta
	if status := rec.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestFinalOutPut(t *testing.T) {
	const testMsg = "TEST"

	// Crear un objeto http.ResponseWriter para capturar la salida
	res := httptest.NewRecorder()

	// Llamar a la función FinalOutPut con valores de prueba
	uker.NewHttp(true).FinalOutPut(res, http.StatusOK, testMsg, map[string]string{"key1": "value1", "key2": "value2"})

	// Verificar el código de estado de la respuesta
	if res.Code != http.StatusOK {
		t.Errorf("Código de estado de respuesta incorrecto. Se esperaba %d, pero se obtuvo %d", http.StatusOK, res.Code)
	}

	// Decodificar la respuesta JSON
	var baseResponse map[string]interface{}
	if err := json.Unmarshal(res.Body.Bytes(), &baseResponse); err != nil {
		t.Errorf("Error decodificando la respuesta base a JSON: %v", err)
	}

	// Decodificar la sección de datos de la respuesta JSON
	encodedData, err := json.Marshal(baseResponse["data"])
	if err != nil {
		t.Errorf("Error codificando datos a JSON: %v", err)
	}

	var dataResponse map[string]string
	if err := json.Unmarshal([]byte(encodedData), &dataResponse); err != nil {
		t.Errorf("Error decodificando el contenido de los datos a JSON: %v", err)
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

func TestExtractReqPaginationParameters(t *testing.T) {
	// Crear una solicitud HTTP con diferentes parámetros de consulta

	fullTestReq, err := http.NewRequest("GET", "/?search=ss&page=2&sort=id&per_page=5&sort_dir=desc", nil)
	if err != nil {
		t.Errorf("Error creando la solicitud HTTP: %v", err)
	}

	emptyReq, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Errorf("Error creando la solicitud HTTP: %v", err)
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
