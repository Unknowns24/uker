package httpx

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

	"github.com/unknowns24/uker/uker/validate"
)

type parserConfig struct {
	base64Data bool
}

// ParserOption allows configuring the behaviour of the request helpers.
type ParserOption func(*parserConfig)

// WithBase64Data enables base64 decoding for the `data` field before unmarshalling.
func WithBase64Data() ParserOption {
	return func(cfg *parserConfig) {
		cfg.base64Data = true
	}
}

func newParserConfig(opts ...ParserOption) parserConfig {
	cfg := parserConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return cfg
}

// BodyParser decodes the request body into the provided structure, validating
// `uker:"required"` tags are present in the payload.
func BodyParser(r *http.Request, target any, opts ...ParserOption) error {
	if reflect.ValueOf(target).Kind() != reflect.Ptr {
		panic(fmt.Errorf("expected pointer, got %s", reflect.ValueOf(target).Kind()))
	}

	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		return errors.New("cannot read request body")
	}

	payload := map[string]any{}
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return errors.New("error happend on json unmarshal of request body")
	}

	cfg := newParserConfig(opts...)

	dataFields, err := decodeDataField(payload[requestKeyData], target, cfg.base64Data)
	if err != nil {
		return err
	}

	return validate.RequiredFields(target, dataFields)
}

// MultiPartFormParser decodes the provided values and returns the received files.
func MultiPartFormParser(r *http.Request, values map[string]any, files []string, opts ...ParserOption) (map[string][]*multipart.FileHeader, error) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		return nil, fmt.Errorf("error parsing multipart form: %v", err)
	}

	cfg := newParserConfig(opts...)

	for key, target := range values {
		if reflect.ValueOf(target).Kind() != reflect.Ptr {
			return nil, fmt.Errorf("expected pointer for value %s", key)
		}

		value := r.FormValue(key)
		dataFields, err := decodeDataField(value, target, cfg.base64Data)
		if err != nil {
			return nil, err
		}

		if err := validate.RequiredFields(target, dataFields); err != nil {
			return nil, fmt.Errorf("missing required parameters in valueInterface: %s", err.Error())
		}
	}

	parsedFiles := map[string][]*multipart.FileHeader{}
	for _, file := range files {
		if fileHeaders := r.MultipartForm.File[file]; fileHeaders != nil {
			parsedFiles[file] = fileHeaders
		}
	}

	return parsedFiles, nil
}

// MultiPartFileToBuff returns the byte buffer of each file in the slice.
func MultiPartFileToBuff(files []*multipart.FileHeader) [][]byte {
	buffers := make([][]byte, len(files))
	for i, file := range files {
		fh, err := file.Open()
		if err != nil {
			continue
		}

		buf := bytes.NewBuffer(nil)
		if _, err := io.Copy(buf, fh); err != nil {
			continue
		}

		buffers[i] = buf.Bytes()
	}

	return buffers
}

// FirstMultiPartFileToBuff returns the content of the first file in the slice.
func FirstMultiPartFileToBuff(files []*multipart.FileHeader) ([]byte, error) {
	fh, err := files[0].Open()
	if err != nil {
		return nil, fmt.Errorf("cannot open the first file of the slice: %s", err)
	}

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, fh); err != nil {
		return nil, fmt.Errorf("cannot copy the first file content to the buffer: %s", err)
	}

	return buf.Bytes(), nil
}

func decodeDataField(raw any, target any, base64Data bool) (map[string]any, error) {
	if raw == nil {
		return nil, errors.New("missing field 'data' inside of the request")
	}

	var payload string
	switch value := raw.(type) {
	case string:
		payload = value
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("error while marshalling JSON: %v", err)
		}
		payload = string(encoded)
	}

	if base64Data {
		decoded, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return nil, errors.New("malformed base64 on data field")
		}
		payload = string(decoded)
	}

	if err := json.Unmarshal([]byte(payload), target); err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %v", err)
	}
	dataFields := map[string]any{}
	if err := json.Unmarshal([]byte(payload), &dataFields); err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	return dataFields, nil
}
