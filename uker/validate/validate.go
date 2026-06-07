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
	value, err := targetValue(target)
	if err != nil {
		return err
	}

	elem := value.Elem()
	if elem.Kind() != reflect.Struct {
		return errors.New("target must point to a struct")
	}

	return requiredFieldsForStruct(elem, body, "")
}

// RequiredFieldsFromPayload checks required fields in structs and slices of structs.
func RequiredFieldsFromPayload(target any, payload any) error {
	value, err := targetValue(target)
	if err != nil {
		return err
	}

	return requiredFieldsForValue(value.Elem(), payload, "")
}

func targetValue(target any) (reflect.Value, error) {
	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Ptr || value.IsNil() {
		return reflect.Value{}, errors.New("target must be a non-nil pointer")
	}

	return value, nil
}

func requiredFieldsForValue(value reflect.Value, payload any, path string) error {
	value = indirectValue(value)
	if !value.IsValid() {
		return fmt.Errorf("missing required parameter at %s", path)
	}

	switch value.Kind() {
	case reflect.Struct:
		body, ok := payload.(map[string]any)
		if !ok {
			if path != "" {
				return fmt.Errorf("expected JSON object at %s", path)
			}
			return errors.New("request body must be a JSON object")
		}
		return requiredFieldsForStruct(value, body, path)
	case reflect.Slice, reflect.Array:
		body, ok := payload.([]any)
		if !ok {
			if path != "" {
				return fmt.Errorf("expected JSON array at %s", path)
			}
			return errors.New("request body must be a JSON array")
		}

		for i := 0; i < value.Len(); i++ {
			itemPath := fmt.Sprintf("[%d]", i)
			if path != "" {
				itemPath = fmt.Sprintf("%s[%d]", path, i)
			}

			var itemPayload any
			if i < len(body) {
				itemPayload = body[i]
			}

			if err := requiredFieldsForValue(value.Index(i), itemPayload, itemPath); err != nil {
				return err
			}
		}

		return nil
	default:
		if path != "" {
			return fmt.Errorf("expected JSON object at %s", path)
		}
		return errors.New("target must point to a struct")
	}
}

func requiredFieldsForStruct(elem reflect.Value, body map[string]any, path string) error {
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
			return missingRequiredParameter(path, field.Name)
		}

		fieldValue := elem.Field(i)
		if fieldValue.Kind() == reflect.String && fieldValue.IsZero() {
			return missingRequiredParameter(path, field.Name)
		}
	}

	return nil
}

func indirectValue(value reflect.Value) reflect.Value {
	for value.IsValid() && value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}

	return value
}

func missingRequiredParameter(path string, name string) error {
	if path != "" {
		return fmt.Errorf("missing required parameter at %s: %s", path, name)
	}

	return fmt.Errorf("missing required parameter: %s", name)
}
