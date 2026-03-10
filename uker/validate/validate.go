package validate

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

const (
	tagName          = "uker"
	tagRequiredValue = "required"
	tagMinValue      = "min"
	tagMaxValue      = "max"
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

// ValidateFields checks that tagged struct fields are valid in the decoded body.
//
// Supported tags:
// - required
// - min=<number>
// - max=<number>
//
// The min/max tags apply to string lengths and numeric values.
func ValidateFields(target any, body map[string]any) error {
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
		rules := parseRules(field.Tag.Get(tagName))
		if len(rules) == 0 {
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
		_, required := rules[tagRequiredValue]
		if required && (!ok || rawValue == nil) {
			return fmt.Errorf("missing required parameter: %s", field.Name)
		}
		if !ok || rawValue == nil {
			continue
		}

		fieldValue := elem.Field(i)
		if required && fieldValue.Kind() == reflect.String && fieldValue.IsZero() {
			return fmt.Errorf("missing required parameter: %s", field.Name)
		}

		if min, ok := rules[tagMinValue]; ok {
			if err := validateMin(field.Name, fieldValue, min); err != nil {
				return err
			}
		}

		if max, ok := rules[tagMaxValue]; ok {
			if err := validateMax(field.Name, fieldValue, max); err != nil {
				return err
			}
		}
	}

	return nil
}

// RequiredFields checks that struct fields tagged as required are present in the decoded body.
func RequiredFields(target any, body map[string]any) error {
	return ValidateFields(target, body)
}

func parseRules(tag string) map[string]string {
	rules := map[string]string{}
	for _, rule := range strings.Split(tag, ",") {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}

		if !strings.Contains(rule, "=") {
			rules[rule] = ""
			continue
		}

		parts := strings.SplitN(rule, "=", 2)
		rules[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}

	return rules
}

func validateMin(name string, value reflect.Value, rawMin string) error {
	min, err := strconv.ParseFloat(rawMin, 64)
	if err != nil {
		return fmt.Errorf("invalid min value for field %s", name)
	}

	switch value.Kind() {
	case reflect.String:
		if float64(len(value.String())) < min {
			return fmt.Errorf("field %s must be at least %.0f characters", name, min)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if float64(value.Int()) < min {
			return fmt.Errorf("field %s must be >= %.0f", name, min)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if float64(value.Uint()) < min {
			return fmt.Errorf("field %s must be >= %.0f", name, min)
		}
	case reflect.Float32, reflect.Float64:
		if value.Float() < min {
			return fmt.Errorf("field %s must be >= %v", name, min)
		}
	}

	return nil
}

func validateMax(name string, value reflect.Value, rawMax string) error {
	max, err := strconv.ParseFloat(rawMax, 64)
	if err != nil {
		return fmt.Errorf("invalid max value for field %s", name)
	}

	switch value.Kind() {
	case reflect.String:
		if float64(len(value.String())) > max {
			return fmt.Errorf("field %s must be at most %.0f characters", name, max)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if float64(value.Int()) > max {
			return fmt.Errorf("field %s must be <= %.0f", name, max)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if float64(value.Uint()) > max {
			return fmt.Errorf("field %s must be <= %.0f", name, max)
		}
	case reflect.Float32, reflect.Float64:
		if value.Float() > max {
			return fmt.Errorf("field %s must be <= %v", name, max)
		}
	}

	return nil
}
