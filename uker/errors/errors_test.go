package errors

import (
	stderrors "errors"
	"testing"
)

const testCode Code = "ERR_TEST"

func TestWrap(t *testing.T) {
	base := stderrors.New("boom")
	err := Wrap(testCode, "failed", base)

	if !stderrors.Is(err, base) {
		t.Fatalf("expected wrapped error")
	}

	if err.Code != testCode {
		t.Fatalf("Code = %s", err.Code)
	}
}

func TestError(t *testing.T) {
	err := New(testCode, "failed")
	if err.Error() != "ERR_TEST: failed" {
		t.Fatalf("Error() = %s", err.Error())
	}
}
