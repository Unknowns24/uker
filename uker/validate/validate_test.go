package validate

import "testing"

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
