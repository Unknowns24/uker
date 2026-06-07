package validate

import (
	"strings"
	"testing"
)

func TestNotEmpty(t *testing.T) {
	if err := NotEmpty(""); err == nil {
		t.Fatalf("expected error")
	}

	if err := NotEmpty("value"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMinLength(t *testing.T) {
	if err := MinLength("go", 3); err == nil {
		t.Fatalf("expected error")
	}

	if err := MinLength("gopher", 3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRequiredFields(t *testing.T) {
	type payload struct {
		Name string `json:"name" uker:"required"`
		Age  int    `json:"age" uker:"required"`
		Note string
	}

	valid := &payload{Name: "Alice", Age: 30}
	body := map[string]any{"name": "Alice", "age": 30}

	if err := RequiredFields(valid, body); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	missing := &payload{Name: "Alice"}
	bodyMissing := map[string]any{"name": "Alice"}

	if err := RequiredFields(missing, bodyMissing); err == nil {
		t.Fatalf("expected error for missing required field")
	}

	empty := &payload{Age: 25}
	bodyEmpty := map[string]any{"name": "", "age": 25}

	if err := RequiredFields(empty, bodyEmpty); err == nil {
		t.Fatalf("expected error for empty required field")
	}
}

func TestRequiredFieldsFromPayloadSlice(t *testing.T) {
	type payload struct {
		Name string `json:"name" uker:"required"`
		Age  int    `json:"age" uker:"required"`
	}

	valid := []payload{
		{Name: "Alice", Age: 30},
		{Name: "Bob", Age: 25},
	}
	body := []any{
		map[string]any{"name": "Alice", "age": 30},
		map[string]any{"name": "Bob", "age": 25},
	}

	if err := RequiredFieldsFromPayload(&valid, body); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRequiredFieldsFromPayloadPointerSlice(t *testing.T) {
	type payload struct {
		Name string `json:"name" uker:"required"`
	}

	valid := []*payload{{Name: "Alice"}}
	body := []any{map[string]any{"name": "Alice"}}

	if err := RequiredFieldsFromPayload(&valid, body); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRequiredFieldsFromPayloadSliceMissingRequiredIncludesIndex(t *testing.T) {
	type payload struct {
		Name string `json:"name" uker:"required"`
	}

	invalid := []payload{{Name: "Alice"}, {}}
	body := []any{
		map[string]any{"name": "Alice"},
		map[string]any{},
	}

	err := RequiredFieldsFromPayload(&invalid, body)
	if err == nil {
		t.Fatalf("expected error for missing required field")
	}
	if !strings.Contains(err.Error(), "[1]") {
		t.Fatalf("expected error to include index, got %q", err.Error())
	}
}
