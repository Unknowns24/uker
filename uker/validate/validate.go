package validate

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

const (
	tagName          = "uker"
	tagRequiredValue = "required"
)

// NotEmpty validates that the provided string is not empty.
func NotEmpty(value string) error {
	if value == "" {
		return errors.New("value cannot be empty")
	}
	return nil
}

// MinLength validates that the provided string has at least the given length.
func MinLength(value string, length int) error {
	if len(value) < length {
		return errors.New("value shorter than allowed")
	}
	return nil
}

// RequiredFields checks that struct fields tagged as required are present in the decoded body.
func RequiredFields(target any, body map[string]any) error {
	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Ptr || value.IsNil() {
		return errors.New("target must be a non-nil pointer")
	}

	elem := value.Elem()
	if elem.Kind() != reflect.Struct {
		return errors.New("target must point to a struct")
	}

	for i := 0; i < elem.NumField(); i++ {
		field := elem.Type().Field(i)
		if !strings.Contains(field.Tag.Get(tagName), tagRequiredValue) {
			continue
		}

		jsonKey := field.Name
		if tag := field.Tag.Get("json"); tag != "" {
			jsonKey = strings.Split(tag, ",")[0]
			if jsonKey == "" {
				jsonKey = field.Name
			}
		}

		rawValue, ok := body[jsonKey]
		if !ok || rawValue == nil {
			return fmt.Errorf("missing required parameter: %s", field.Name)
		}

		fieldValue := elem.Field(i)
		if fieldValue.Kind() == reflect.String && fieldValue.IsZero() {
			return fmt.Errorf("missing required parameter: %s", field.Name)
		}
	}

	return nil
}
