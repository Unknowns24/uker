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

func TestValidateFields(t *testing.T) {
	type payload struct {
		Name string `json:"name" uker:"required,min=3,max=5"`
		Age  int    `json:"age" uker:"required,min=18,max=65"`
		Note string
	}

	valid := &payload{Name: "Alice", Age: 30}
	body := map[string]any{"name": "Alice", "age": 30}

	if err := ValidateFields(valid, body); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	missing := &payload{Name: "Alice"}
	bodyMissing := map[string]any{"name": "Alice"}
	if err := ValidateFields(missing, bodyMissing); err == nil {
		t.Fatalf("expected error for missing required field")
	}

	shortName := &payload{Name: "Al", Age: 25}
	bodyShortName := map[string]any{"name": "Al", "age": 25}
	if err := ValidateFields(shortName, bodyShortName); err == nil {
		t.Fatalf("expected error for min string length")
	}

	ageTooHigh := &payload{Name: "Alice", Age: 70}
	bodyAgeTooHigh := map[string]any{"name": "Alice", "age": 70}
	if err := ValidateFields(ageTooHigh, bodyAgeTooHigh); err == nil {
		t.Fatalf("expected error for max int value")
	}
}

func TestRequiredFieldsCompatibility(t *testing.T) {
	type payload struct {
		Name string `json:"name" uker:"required"`
	}

	if err := RequiredFields(&payload{Name: "ok"}, map[string]any{"name": "ok"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
