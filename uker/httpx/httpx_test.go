package httpx_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/unknowns24/uker/uker/httpx"
)

type testStruct struct {
	Param1 string `uker:"required"`
	Param2 string `uker:"required"`
	Param3 int    `uker:"required"`
	Param4 bool
}

func TestMultiPartFormParser(t *testing.T) {
	test := testStruct{Param1: "value1", Param2: "value2"}

	testJSON, err := json.Marshal(test)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	encoded := base64.StdEncoding.EncodeToString(testJSON)
	writer.WriteField("data", encoded)

	fileWriter, _ := writer.CreateFormFile("file1", "example.txt")
	fileWriter.Write([]byte("simulated file content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	values := make(map[string]any)
	data := testStruct{}
	values["data"] = &data

	files, err := httpx.MultiPartFormParser(req, values, []string{"file1"}, httpx.WithBase64Data())
	if err != nil {
		t.Fatalf("MultiPartFormParser: %v", err)
	}

	if data.Param1 != "value1" {
		t.Fatalf("Param1 = %q", data.Param1)
	}

	if len(files["file1"]) == 0 {
		t.Fatalf("expected files, got none")
	}
}

func TestBodyParser(t *testing.T) {
	test := testStruct{Param1: "value1", Param2: "value2"}
	testJSON, err := json.Marshal(test)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	encoded := base64.StdEncoding.EncodeToString(testJSON)
	body, err := json.Marshal(map[string]string{"data": encoded})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data testStruct
		if err := httpx.BodyParser(r, &data, httpx.WithBase64Data()); err != nil {
			t.Fatalf("BodyParser: %v", err)
		}

		if data.Param1 != "value1" {
			t.Fatalf("Param1 = %q", data.Param1)
		}

		w.WriteHeader(http.StatusOK)
	})

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestBodyParserRootPayload(t *testing.T) {
	test := testStruct{Param1: "value1", Param2: "value2", Param3: 42, Param4: true}
	body, err := json.Marshal(test)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data testStruct
		if err := httpx.BodyParser(r, &data); err != nil {
			t.Fatalf("BodyParser: %v", err)
		}

		if data.Param3 != 42 {
			t.Fatalf("Param3 = %d", data.Param3)
		}
		if !data.Param4 {
			t.Fatalf("Param4 = %v", data.Param4)
		}
	})

	handler.ServeHTTP(httptest.NewRecorder(), req)
}

func TestBodyParserMissingRequired(t *testing.T) {
	test := map[string]any{"Param2": "value2", "Param3": 10}
	testJSON, err := json.Marshal(test)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	encoded := base64.StdEncoding.EncodeToString(testJSON)
	body, err := json.Marshal(map[string]string{"data": encoded})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data testStruct
		if err := httpx.BodyParser(r, &data, httpx.WithBase64Data()); err == nil {
			t.Fatalf("expected error for missing required field")
		}
	})

	handler.ServeHTTP(httptest.NewRecorder(), req)
}

func TestFinalOutput(t *testing.T) {
	rec := httptest.NewRecorder()
	httpx.FinalOutput(rec, http.StatusOK, map[string]string{"key1": "value1"})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if response["key1"] != "value1" {
		t.Fatalf("value = %q", response["key1"])
	}
}

func TestErrorOutput(t *testing.T) {
	const message = "TEST"
	rec := httptest.NewRecorder()
	httpx.ErrorOutput(rec, http.StatusBadRequest, &httpx.ResponseStatus{Type: httpx.Error, Code: message})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}

	if contentType := rec.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("content-type = %s", contentType)
	}

	var status httpx.ResponseStatus
	if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if status.Type != httpx.Error {
		t.Fatalf("type = %s", status.Type)
	}
	if status.Code != message {
		t.Fatalf("code = %s", status.Code)
	}
}
