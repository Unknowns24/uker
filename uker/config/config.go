package config

import (
	"fmt"
	"os"
	"reflect"
	"strings"
)

// Loader loads configuration structs from environment variables.
type Loader struct {
	prefix string
}

// New creates a new Loader with the provided prefix.
func New(prefix string) Loader {
	return Loader{prefix: prefix}
}

// Load reads the environment variables into the provided struct pointer using the `config` tag.
func (l Loader) Load(target any) error {
	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Ptr || value.IsNil() {
		return fmt.Errorf("target must be a non-nil pointer")
	}

	elem := value.Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("target must point to a struct")
	}

	for i := 0; i < elem.NumField(); i++ {
		field := elem.Type().Field(i)
		key := field.Tag.Get("config")
		if key == "" {
			key = field.Name
		}

		envKey := strings.ToUpper(fmt.Sprintf("%s_%s", l.prefix, key))
		if value, ok := os.LookupEnv(envKey); ok {
			elem.Field(i).SetString(value)
		}
	}

	return nil
}
