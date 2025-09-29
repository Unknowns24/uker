package id

import "testing"

func TestNew(t *testing.T) {
	id, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if len(id) != 32 {
		t.Fatalf("len = %d", len(id))
	}
}

func TestShort(t *testing.T) {
	id, err := Short()
	if err != nil {
		t.Fatalf("Short: %v", err)
	}

	if len(id) == 0 {
		t.Fatalf("Short returned empty string")
	}
}
