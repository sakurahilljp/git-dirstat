package model

import (
	"errors"
	"testing"
)

func TestErrors(t *testing.T) {
	err1 := NewInputError("bad %s", "input")
	if err1.Error() != "bad input" {
		t.Errorf("unexpected Error() output: %s", err1.Error())
	}
	var exitErr *ExitCodeError
	if !errors.As(err1, &exitErr) || exitErr.Code != 2 {
		t.Errorf("NewInputError should return ExitCodeError with Code 2")
	}
	if err1.Unwrap() == nil {
		t.Errorf("expected inner error to not be nil")
	}

	err2 := NewRuntimeError("runtime %s", "error")
	if err2.Error() != "runtime error" {
		t.Errorf("unexpected Error() output: %s", err2.Error())
	}
	if !errors.As(err2, &exitErr) || exitErr.Code != 1 {
		t.Errorf("NewRuntimeError should return ExitCodeError with Code 1")
	}
	if err2.Unwrap() == nil {
		t.Errorf("expected inner error to not be nil")
	}
}
